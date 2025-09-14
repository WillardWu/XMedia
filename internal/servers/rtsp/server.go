package rtsp

import (
	"XMedia/internal/conf"
	"XMedia/internal/logger"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
)

type serverParent interface {
	logger.Writer
}

type Server struct {
	Address        string
	ReadTimeout    conf.Duration
	WriteTimeout   conf.Duration
	WriteQueueSize int
	IsTLS          bool
	Parent         serverParent

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
	srv       *gortsplib.Server
	mutex     sync.RWMutex
	conns     map[*gortsplib.ServerConn]*conn
	sessions  map[*gortsplib.ServerSession]*session
}

func printAddresses(srv *gortsplib.Server) string {
	var ret []string

	ret = append(ret, fmt.Sprintf("%s (TCP)", srv.RTSPAddress))

	if srv.UDPRTPAddress != "" {
		ret = append(ret, fmt.Sprintf("%s (UDP/RTP)", srv.UDPRTPAddress))
	}

	if srv.UDPRTCPAddress != "" {
		ret = append(ret, fmt.Sprintf("%s (UDP/RTCP)", srv.UDPRTCPAddress))
	}

	return strings.Join(ret, ", ")
}

func (s *Server) Initialize() error {
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())

	s.conns = make(map[*gortsplib.ServerConn]*conn)
	s.sessions = make(map[*gortsplib.ServerSession]*session)

	s.srv = &gortsplib.Server{
		Handler:        s,
		ReadTimeout:    time.Duration(s.ReadTimeout),
		WriteTimeout:   time.Duration(s.WriteTimeout),
		WriteQueueSize: s.WriteQueueSize,
		RTSPAddress:    s.Address,
	}

	err := s.srv.Start()
	if err != nil {
		return err
	}

	s.Log(logger.Info, "listener opened on %s", printAddresses(s.srv))

	s.wg.Add(1)
	go s.run()
	return nil
}

// Log implements logger.Writer.
func (s *Server) Log(level logger.Level, format string, args ...interface{}) {
	label := func() string {
		if s.IsTLS {
			return "RTSPS"
		}
		return "RTSP"
	}()
	s.Parent.Log(level, "[%s] "+format, append([]interface{}{label}, args...)...)
}

// Close closes the server.
func (s *Server) Close() {
	s.Log(logger.Info, "listener is closing")
	s.ctxCancel()
	s.wg.Wait()
}

func (s *Server) run() {
	defer s.wg.Done()

	serverErr := make(chan error)
	go func() {
		serverErr <- s.srv.Wait()
	}()

outer:
	select {
	case err := <-serverErr:
		s.Log(logger.Error, "%s", err)
		break outer

	case <-s.ctx.Done():
		s.srv.Close()
		<-serverErr
		break outer
	}
	s.ctxCancel()
}

// ServerHandlerOnConnOpen can be implemented by a ServerHandler.
func (s *Server) OnConnOpen(connCtx *gortsplib.ServerHandlerOnConnOpenCtx) {
	s.Log(logger.Info, "new connection comming...")
}
