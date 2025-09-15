package rtsp

import (
	"XMedia/internal/conf"
	"XMedia/internal/logger"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/google/uuid"
	"github.com/pion/rtp"
)

type session struct {
	isTLS      bool
	transports conf.RTSPTransports
	rsession   *gortsplib.ServerSession
	rconn      *gortsplib.ServerConn
	rserver    *gortsplib.Server
	//externalCmdPool *externalcmd.Pool
	//pathManager     serverPathManager
	parent logger.Writer

	uuid    uuid.UUID
	created time.Time
	//path         defs.Path
	//stream       *stream.Stream
	//onUnreadHook func()
	mutex     sync.Mutex
	state     gortsplib.ServerSessionState
	transport *gortsplib.Transport
	pathName  string
	query     string
}

func (s *session) initialize() {
	s.uuid = uuid.New()
	s.created = time.Now()

	s.Log(logger.Info, "created by %v", s.rconn.NetConn().RemoteAddr())
}

// Log implements logger.Writer.
func (s *session) Log(level logger.Level, format string, args ...interface{}) {
	id := hex.EncodeToString(s.uuid[:4])
	s.parent.Log(level, "[session %s] "+format, append([]interface{}{id}, args...)...)
}

// onClose is called by rtspServer.
func (s *session) onClose(err error) {
	// if s.rsession.State() == gortsplib.ServerSessionStatePlay {
	// 	s.onUnreadHook()
	// }

	// switch s.rsession.State() {
	// case gortsplib.ServerSessionStatePrePlay, gortsplib.ServerSessionStatePlay:
	// 	s.path.RemoveReader(defs.PathRemoveReaderReq{Author: s})

	// case gortsplib.ServerSessionStatePreRecord, gortsplib.ServerSessionStateRecord:
	// 	s.path.RemovePublisher(defs.PathRemovePublisherReq{Author: s})
	// }

	// s.path = nil
	// s.stream = nil

	s.Log(logger.Info, "destroyed: %v", err)
}

// onAnnounce is called by rtspServer.
func (s *session) onAnnounce(c *conn, ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	if len(ctx.Path) == 0 || ctx.Path[0] != '/' {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, fmt.Errorf("invalid path")
	}
	ctx.Path = ctx.Path[1:]

	//s.path = path

	s.mutex.Lock()
	s.state = gortsplib.ServerSessionStatePreRecord
	s.pathName = ctx.Path
	s.query = ctx.Query
	s.mutex.Unlock()

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

// onSetup is called by rtspServer.
func (s *session) onSetup(c *conn, ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	if len(ctx.Path) == 0 || ctx.Path[0] != '/' {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil, fmt.Errorf("invalid path")
	}
	ctx.Path = ctx.Path[1:]

	// in case the client is setupping a stream with UDP or UDP-multicast, and these
	// transport protocols are disabled, gortsplib already blocks the request.
	// we have only to handle the case in which the transport protocol is TCP
	// and it is disabled.
	if ctx.Transport == gortsplib.TransportTCP {
		if _, ok := s.transports[gortsplib.TransportTCP]; !ok {
			return &base.Response{
				StatusCode: base.StatusUnsupportedTransport,
			}, nil, nil
		}
	}

	switch s.rsession.State() {
	case gortsplib.ServerSessionStateInitial, gortsplib.ServerSessionStatePrePlay: // play
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil, nil
	default: // record
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil, nil
	}
}

// onRecord is called by rtspServer.
func (s *session) onRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	s.mutex.Lock()
	s.state = gortsplib.ServerSessionStateRecord
	s.transport = s.rsession.SetuppedTransport()
	s.mutex.Unlock()

	ctx.Session.OnPacketRTPAny(func(medi *description.Media, ft format.Format, pkt *rtp.Packet) {
		// route the RTP packet to all readers
		fmt.Printf("RTP packet received track: %s,codec: %s, payload: %d, seq: %d, ts: %d, size: %d\n",
			medi.Type, ft.Codec(), pkt.Header.PayloadType, pkt.Header.SequenceNumber, pkt.Header.Timestamp,
			len(pkt.Payload))
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}
