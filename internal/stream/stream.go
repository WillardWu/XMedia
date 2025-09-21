package stream

import (
	"XMedia/internal/logger"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

// Stream is a media stream.
// It stores tracks, readers and allows to write data to readers, converting it when needed.
type Stream struct {
	WriteQueueSize     int
	UDPMaxPayloadSize  int
	Desc               *description.Session
	GenerateRTPPackets bool
	Parent             logger.Writer

	bytesReceived *uint64
	bytesSent     *uint64
	streamMedias  map[*description.Media]*streamMedia
	mutex         sync.RWMutex
	rtspStream    *gortsplib.ServerStream

	readerRunning chan struct{}
}

func (s *Stream) Initialize() error {
	s.bytesReceived = new(uint64)
	s.bytesSent = new(uint64)
	s.streamMedias = make(map[*description.Media]*streamMedia)
	s.readerRunning = make(chan struct{})

	for _, media := range s.Desc.Medias {
		s.streamMedias[media] = &streamMedia{
			udpMaxPayloadSize:  s.UDPMaxPayloadSize,
			media:              media,
			generateRTPPackets: s.GenerateRTPPackets,
			parent:             s.Parent,
		}
		err := s.streamMedias[media].initialize()
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteRTPPacket writes a RTP packet.
func (s *Stream) WriteRTPPacket(
	medi *description.Media,
	forma format.Format,
	pkt *rtp.Packet,
	ntp time.Time,
	pts int64,
) {

	sm := s.streamMedias[medi]
	sf := sm.formats[forma]

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sf.writeRTPPacket(s, medi, pkt, ntp, pts)
}
