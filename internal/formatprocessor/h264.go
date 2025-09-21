// Package formatprocessor cleans and normalizes streams.
package formatprocessor

import (
	"XMedia/internal/logger"
	"XMedia/internal/unit"
	"bytes"
	"errors"
	"time"

	mch264 "github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"

	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/pion/rtp"
)

// extract SPS and PPS without decoding RTP packets
func rtpH264ExtractParams(payload []byte) ([]byte, []byte) {
	if len(payload) < 1 {
		return nil, nil
	}

	typ := mch264.NALUType(payload[0] & 0x1F)

	switch typ {
	case mch264.NALUTypeSPS:
		return payload, nil

	case mch264.NALUTypePPS:
		return nil, payload

	case mch264.NALUTypeSTAPA:
		payload = payload[1:]
		var sps []byte
		var pps []byte

		for len(payload) > 0 {
			if len(payload) < 2 {
				break
			}

			size := uint16(payload[0])<<8 | uint16(payload[1])
			payload = payload[2:]

			if size == 0 {
				break
			}

			if int(size) > len(payload) {
				return nil, nil
			}

			nalu := payload[:size]
			payload = payload[size:]

			typ = mch264.NALUType(nalu[0] & 0x1F)

			switch typ {
			case mch264.NALUTypeSPS:
				sps = nalu

			case mch264.NALUTypePPS:
				pps = nalu
			}
		}

		return sps, pps

	default:
		return nil, nil
	}
}

type h264 struct {
	UDPMaxPayloadSize  int
	Format             *format.H264
	GenerateRTPPackets bool
	Parent             logger.Writer

	encoder     *rtph264.Encoder
	decoder     *rtph264.Decoder
	randomStart uint32
}

func (t *h264) initialize() error {
	if t.GenerateRTPPackets {
		err := t.createEncoder(nil, nil)
		if err != nil {
			return err
		}

		t.randomStart, err = randUint32()
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *h264) createEncoder(
	ssrc *uint32,
	initialSequenceNumber *uint16,
) error {
	t.encoder = &rtph264.Encoder{
		PayloadMaxSize:        t.UDPMaxPayloadSize - 12,
		PayloadType:           t.Format.PayloadTyp,
		SSRC:                  ssrc,
		InitialSequenceNumber: initialSequenceNumber,
		PacketizationMode:     t.Format.PacketizationMode,
	}
	return t.encoder.Init()
}

func (t *h264) updateTrackParametersFromRTPPacket(payload []byte) {
	sps, pps := rtpH264ExtractParams(payload)

	if (sps != nil && !bytes.Equal(sps, t.Format.SPS)) ||
		(pps != nil && !bytes.Equal(pps, t.Format.PPS)) {
		if sps == nil {
			sps = t.Format.SPS
		}
		if pps == nil {
			pps = t.Format.PPS
		}
		t.Format.SafeSetParams(sps, pps)
	}
}

func (t *h264) ProcessUnit(unit.Unit) error {
	return nil
}

// process a RTP packet and convert it into a unit.
func (t *h264) ProcessRTPPacket(
	pkt *rtp.Packet,
	ntp time.Time,
	pts int64,
	hasNonRTSPReaders bool,
) (unit.Unit, error) {
	u := &unit.H264{
		Base: unit.Base{
			RTPPackets: []*rtp.Packet{pkt}, //给rtsp读者使用的
			NTP:        ntp,
			PTS:        pts,
		},
	}
	//依据最新负载信息  动态更新系统sps/pps
	t.updateTrackParametersFromRTPPacket(pkt.Payload)

	//是否需要对rtp进行二次编码
	if t.encoder == nil {
		// remove padding
		pkt.Padding = false
		pkt.PaddingSize = 0

		// RTP packets exceed maximum size: start re-encoding them
		if pkt.MarshalSize() > t.UDPMaxPayloadSize {
			t.Parent.Log(logger.Info, "RTP packets are too big, remuxing them into smaller ones")

			v1 := pkt.SSRC
			v2 := pkt.SequenceNumber
			err := t.createEncoder(&v1, &v2)
			if err != nil {
				return nil, err
			}
		}
	}

	// decode from RTP  这里判断结果是 要生成 u.AU ,u.AU是给非RTSP读者使用的，或者作为二次RTP编码的输入
	if hasNonRTSPReaders || t.decoder != nil || t.encoder != nil {
		if t.decoder == nil {
			var err error
			t.decoder, err = t.Format.CreateDecoder()
			if err != nil {
				return nil, err
			}
		}
		//解码RTP数据包
		au, err := t.decoder.Decode(pkt)

		if t.encoder != nil {
			//说明RTP需要二次编码，这里后面会重新更新数据
			u.RTPPackets = nil
		}

		if err != nil {
			//FU-A 数据不够底层会缓存数据，下次一并返回完整的，本次u.AU 为空 上层就不发送. RTSP的接收者会发送
			if errors.Is(err, rtph264.ErrNonStartingPacketAndNoPrevious) ||
				errors.Is(err, rtph264.ErrMorePacketsNeeded) {
				return u, nil
			}
			return nil, err
		}
		//修改必要信息 如I帧前面加SPS/PPS
		u.AU = t.remuxAccessUnit(au)
	}

	// route packet as is 上传需要的数据都满足 RTP是原样的数据，NALU已经解码更新
	if t.encoder == nil {
		return u, nil
	}

	// encode into RTP
	if len(u.AU) != 0 {
		pkts, err := t.encoder.Encode(u.AU)
		if err != nil {
			return nil, err
		}
		u.RTPPackets = pkts

		for _, newPKT := range u.RTPPackets {
			newPKT.Timestamp = pkt.Timestamp
		}
	}

	return u, nil
}

func (t *h264) remuxAccessUnit(au [][]byte) [][]byte {
	isKeyFrame := false
	n := 0

	for _, nalu := range au {
		typ := mch264.NALUType(nalu[0] & 0x1F)

		switch typ {
		case mch264.NALUTypeSPS, mch264.NALUTypePPS: // parameters: remove
			continue

		case mch264.NALUTypeAccessUnitDelimiter: // AUD: remove
			continue

		case mch264.NALUTypeIDR: // key frame
			if !isKeyFrame {
				isKeyFrame = true

				// prepend parameters
				if t.Format.SPS != nil && t.Format.PPS != nil {
					n += 2
				}
			}
		}
		n++
	}

	if n == 0 {
		return nil
	}

	filteredNALUs := make([][]byte, n)
	i := 0

	if isKeyFrame && t.Format.SPS != nil && t.Format.PPS != nil {
		filteredNALUs[0] = t.Format.SPS
		filteredNALUs[1] = t.Format.PPS
		i = 2
	}

	for _, nalu := range au {
		typ := mch264.NALUType(nalu[0] & 0x1F)

		switch typ {
		case mch264.NALUTypeSPS, mch264.NALUTypePPS:
			continue

		case mch264.NALUTypeAccessUnitDelimiter:
			continue
		}

		filteredNALUs[i] = nalu
		i++
	}

	return filteredNALUs
}
