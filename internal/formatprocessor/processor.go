// Package formatprocessor cleans and normalizes streams.
package formatprocessor

import (
	"XMedia/internal/logger"
	"XMedia/internal/unit"
	"crypto/rand"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

func randUint32() (uint32, error) {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return 0, err
	}
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), nil
}

// Processor is the codec-dependent part of the processing that happens inside stream.Stream.
type Processor interface {
	// process a Unit.
	ProcessUnit(unit.Unit) error

	// process a RTP packet and convert it into a unit.
	ProcessRTPPacket(
		pkt *rtp.Packet,
		ntp time.Time,
		pts int64,
		hasNonRTSPReaders bool,
	) (unit.Unit, error)

	initialize() error
}

// New allocates a Processor.
func New(
	udpMaxPayloadSize int,
	forma format.Format,
	generateRTPPackets bool,
	parent logger.Writer,
) (Processor, error) {
	var proc Processor

	switch forma := forma.(type) {

	case *format.H264:
		proc = &h264{
			UDPMaxPayloadSize:  udpMaxPayloadSize,
			Format:             forma,
			GenerateRTPPackets: generateRTPPackets,
			Parent:             parent,
		}

	case *format.MPEG4Audio:
		proc = &mpeg4Audio{
			UDPMaxPayloadSize:  udpMaxPayloadSize,
			Format:             forma,
			GenerateRTPPackets: generateRTPPackets,
			Parent:             parent,
		}
	default:
		// proc = &generic{
		// 	UDPMaxPayloadSize:  udpMaxPayloadSize,
		// 	Format:             forma,
		// 	GenerateRTPPackets: generateRTPPackets,
		// 	Parent:             parent,
		// }
	}

	err := proc.initialize()
	return proc, err
}
