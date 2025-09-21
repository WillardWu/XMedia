package core

import (
	"XMedia/internal/conf"
	"XMedia/internal/defs"
	"XMedia/internal/logger"
	"XMedia/internal/stream"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/description"
)

type pathParent interface {
	logger.Writer
	pathReady(*path)
	closePath(*path)
}

type path struct {
	parentCtx         context.Context
	rtspAddress       string
	readTimeout       conf.Duration
	writeTimeout      conf.Duration
	writeQueueSize    int
	udpMaxPayloadSize int
	name              string
	matches           []string
	wg                *sync.WaitGroup
	parent            pathParent

	ctx            context.Context
	ctxCancel      func()
	source         defs.Source
	publisherQuery string
	stream         *stream.Stream
	readyTime      time.Time

	chAddPublisher   chan defs.PathAddPublisherReq
	chStartPublisher chan defs.PathStartPublisherReq

	// out
	done chan struct{}
}

func (pa *path) initialize() {
	ctx, ctxCancel := context.WithCancel(pa.parentCtx)

	pa.ctx = ctx
	pa.ctxCancel = ctxCancel
	pa.chAddPublisher = make(chan defs.PathAddPublisherReq)
	pa.chStartPublisher = make(chan defs.PathStartPublisherReq)

	pa.done = make(chan struct{})

	pa.Log(logger.Info, "created")

	pa.wg.Add(1)
	go pa.run()
}

func (pa *path) close() {
	pa.ctxCancel()
}

func (pa *path) wait() {
	<-pa.done
}

// Log implements logger.Writer.
func (pa *path) Log(level logger.Level, format string, args ...interface{}) {
	pa.parent.Log(level, "[path "+pa.name+"] "+format, args...)
}

func (pa *path) Name() string {
	return pa.name
}

// addPublisher is called by a publisher through pathManager.
func (pa *path) addPublisher(req defs.PathAddPublisherReq) (defs.Path, error) {
	select {
	case pa.chAddPublisher <- req:
		res := <-req.Res
		return res.Path, res.Err
	case <-pa.ctx.Done():
		return nil, fmt.Errorf("terminated")
	}
}

func (pa *path) run() {
	defer close(pa.done)
	defer pa.wg.Done()

	err := pa.runInner()

	// call before destroying context
	pa.parent.closePath(pa)

	pa.ctxCancel()

	// if pa.stream != nil {
	// 	pa.setNotReady()
	// }

	// if pa.source != nil {
	// 	if source, ok := pa.source.(*staticsources.Handler); ok {
	// 		if !pa.conf.SourceOnDemand || pa.onDemandStaticSourceState != pathOnDemandStateInitial {
	// 			source.Close("path is closing")
	// 		}
	// 	} else if source, ok := pa.source.(defs.Publisher); ok {
	// 		source.Close()
	// 	}
	// }

	// if pa.onUnDemandHook != nil {
	// 	pa.onUnDemandHook("path destroyed")
	// }

	pa.Log(logger.Info, "destroyed: %v", err)
}

func (pa *path) runInner() error {
	for {
		select {
		case req := <-pa.chAddPublisher:
			pa.doAddPublisher(req)
		case req := <-pa.chStartPublisher:
			pa.doStartPublisher(req)
		}
	}
}

func (pa *path) doAddPublisher(req defs.PathAddPublisherReq) {
	pa.source = req.Author
	pa.publisherQuery = req.AccessRequest.Query

	req.Res <- defs.PathAddPublisherRes{Path: pa}
}

func (pa *path) doStartPublisher(req defs.PathStartPublisherReq) {
	if pa.source != req.Author {
		req.Res <- defs.PathStartPublisherRes{Err: fmt.Errorf("publisher is not assigned to this path anymore")}
		return
	}

	err := pa.setReady(req.Desc, req.GenerateRTPPackets)
	if err != nil {
		req.Res <- defs.PathStartPublisherRes{Err: err}
		return
	}

	req.Author.Log(logger.Info, "is publishing to path '%s', %s",
		pa.name,
		defs.MediasInfo(req.Desc.Medias))

	// if pa.conf.HasOnDemandPublisher() && pa.onDemandPublisherState != pathOnDemandStateInitial {
	// 	pa.onDemandPublisherReadyTimer.Stop()
	// 	pa.onDemandPublisherReadyTimer = emptyTimer()
	// 	pa.onDemandPublisherScheduleClose()
	// }

	// pa.consumeOnHoldRequests()

	req.Res <- defs.PathStartPublisherRes{Stream: pa.stream}
}

func (pa *path) StartPublisher(req defs.PathStartPublisherReq) (*stream.Stream, error) {
	req.Res = make(chan defs.PathStartPublisherRes)
	select {
	case pa.chStartPublisher <- req:
		res := <-req.Res
		return res.Stream, res.Err
	case <-pa.ctx.Done():
		return nil, fmt.Errorf("terminated")
	}
}

func (pa *path) setReady(desc *description.Session, allocateEncoder bool) error {
	pa.stream = &stream.Stream{
		WriteQueueSize:     pa.writeQueueSize,
		UDPMaxPayloadSize:  pa.udpMaxPayloadSize,
		Desc:               desc,
		GenerateRTPPackets: allocateEncoder,
		Parent:             pa.source,
	}
	err := pa.stream.Initialize()
	if err != nil {
		return err
	}

	pa.readyTime = time.Now()

	pa.parent.pathReady(pa)

	return nil
}
