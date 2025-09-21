package core

import (
	"XMedia/internal/conf"
	"XMedia/internal/defs"
	"XMedia/internal/logger"
	"context"
	"fmt"
	"sync"
)

type pathData struct {
	path  *path
	ready bool
}

type pathManagerParent interface {
	logger.Writer
}

type pathManager struct {
	rtspAddress       string
	readTimeout       conf.Duration
	writeTimeout      conf.Duration
	writeQueueSize    int
	udpMaxPayloadSize int
	parent            pathManagerParent

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
	paths     map[string]*pathData

	chAddPublisher chan defs.PathAddPublisherReq
	chClosePath    chan *path
	chPathReady    chan *path
}

func (pm *pathManager) initialize() {
	ctx, ctxCancel := context.WithCancel(context.Background())

	pm.ctx = ctx
	pm.ctxCancel = ctxCancel
	pm.paths = make(map[string]*pathData)

	pm.chAddPublisher = make(chan defs.PathAddPublisherReq)
	pm.chClosePath = make(chan *path)
	pm.chPathReady = make(chan *path)

	pm.Log(logger.Info, "path manager created")

	pm.wg.Add(1)
	go pm.run()
}

// Log implements logger.Writer.
func (pm *pathManager) Log(level logger.Level, format string, args ...interface{}) {
	pm.parent.Log(level, format, args...)
}

func (pm *pathManager) run() {
	defer pm.wg.Done()
outer:
	for {
		select {
		case req := <-pm.chAddPublisher:
			pm.doAddPublisher(req)
		case pa := <-pm.chClosePath:
			pm.doClosePath(pa)
		case pa := <-pm.chPathReady:
			pm.doPathReady(pa)
		case <-pm.ctx.Done():
			break outer
		}
	}
	pm.ctxCancel()
}

// AddPublisher is called by a publisher.
func (pm *pathManager) AddPublisher(req defs.PathAddPublisherReq) (defs.Path, error) {
	req.Res = make(chan defs.PathAddPublisherRes)
	select {
	case pm.chAddPublisher <- req:
		res := <-req.Res
		if res.Err != nil {
			return nil, res.Err
		}

		return res.Path.(*path).addPublisher(req)

	case <-pm.ctx.Done():
		return nil, fmt.Errorf("terminated")
	}
}

func (pm *pathManager) doAddPublisher(req defs.PathAddPublisherReq) {
	// pathConf, pathMatches, err := conf.FindPathConf(pm.pathConfs, req.AccessRequest.Name)
	// if err != nil {
	// 	req.Res <- defs.PathAddPublisherRes{Err: err}
	// 	return
	// }

	// if !req.AccessRequest.SkipAuth {
	// 	err = pm.authManager.Authenticate(req.AccessRequest.ToAuthRequest())
	// 	if err != nil {
	// 		req.Res <- defs.PathAddPublisherRes{Err: err}
	// 		return
	// 	}
	// }

	// create path if it doesn't exist
	if _, ok := pm.paths[req.AccessRequest.Name]; !ok {
		pm.createPath(req.AccessRequest.Name)
	}

	pd := pm.paths[req.AccessRequest.Name]
	req.Res <- defs.PathAddPublisherRes{Path: pd.path}
}

func (pm *pathManager) createPath(name string) {
	pa := &path{
		parentCtx:         pm.ctx,
		rtspAddress:       pm.rtspAddress,
		readTimeout:       pm.readTimeout,
		writeTimeout:      pm.writeTimeout,
		writeQueueSize:    pm.writeQueueSize,
		udpMaxPayloadSize: pm.udpMaxPayloadSize,
		name:              name,
		wg:                &pm.wg,
		parent:            pm,
	}
	pa.initialize()

	pm.paths[name] = &pathData{path: pa}
}

// closePath is called by path.
func (pm *pathManager) closePath(pa *path) {
	select {
	case pm.chClosePath <- pa:
	case <-pm.ctx.Done():
	case <-pa.ctx.Done(): // in case pathManager is blocked by path.wait()
	}
}

// pathReady is called by path.
func (pm *pathManager) pathReady(pa *path) {
	select {
	case pm.chPathReady <- pa:
	case <-pm.ctx.Done():
	case <-pa.ctx.Done(): // in case pathManager is blocked by path.wait()
	}
}

func (pm *pathManager) doClosePath(pa *path) {
	if pd, ok := pm.paths[pa.name]; !ok || pd.path != pa {
		return
	}
	pm.removePath(pa)
}

func (pm *pathManager) removePath(pa *path) {
	delete(pm.paths, pa.name)
}

func (pm *pathManager) doPathReady(pa *path) {
	if pd, ok := pm.paths[pa.name]; !ok || pd.path != pa {
		return
	}
	pm.paths[pa.name].ready = true
}
