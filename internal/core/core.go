package core

import (
	"XMedia/internal/conf"
	"XMedia/internal/logger"
	"context"
	"fmt"
	"time"
)

type Core struct {
	product   string
	ctx       context.Context
	ctxCancel func()
	confPath  string
	conf      *conf.Config
	logger    *logger.AsyncLogQueue

	// out
	done chan struct{}
}

func NewCore(params map[string]interface{}) (*Core, error) {
	var err error
	ctx, ctxCancel := context.WithCancel(context.Background())
	c := &Core{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		confPath:  conf.CONFIG_FILE,
		conf:      nil,
		done:      make(chan struct{}, 1),
	}
	for k, v := range params {
		switch k {
		case "product":
			if val, ok := v.(string); ok {
				c.product = val
			}
		case "confPath":
			if val, ok := v.(string); ok {
				c.confPath = val
			}
		}
	}
	c.conf, err = conf.Load(c.confPath)
	if err != nil {
		return nil, err
	}

	err = c.createResources(true)
	if err != nil {
		if c.logger != nil {
			c.Log(logger.Error, "Core.createResources %v", err)
		} else {
			fmt.Printf("ERR: %s\n", err)
		}
		c.closeResources()
		return nil, err
	}

	c.Log(logger.Info, "%s begin start...", c.product)

	return c, nil
}

func (p *Core) Start() {
	go p.run()
}

// Close closes Core and waits for all goroutines to return.
func (p *Core) Close() {
	p.ctxCancel()
	<-p.done
}

// Wait waits for the Core to exit.
func (p *Core) Wait() {
	<-p.done
}

func (p *Core) run() {
	defer close(p.done)
	p.logger.RegisterCategory("rtsp")
outer:
	for {
		select {
		case <-p.ctx.Done():
			break outer
		default:
			time.Sleep(1 * time.Second)
			//fmt.Println("hello")

		}
	}
	p.ctxCancel()
}

func (p *Core) createResources(initial bool) error {
	var err error
	if p.logger == nil {
		p.logger, err = logger.NewAsyncLogQueue(
			p.product,
			logger.WithLogMaxSize(p.conf.Log.LogMaxSize),
			logger.WithLogMaxBackup(p.conf.Log.LogMaxBackup),
			logger.WithLogQueueSize(p.conf.Log.LogQueueSize),
		)
		if err != nil {
			return err
		}
	}

	return err
}

func (p *Core) closeResources() {
	return
}

// Log implements log.Writer.
func (p *Core) Log(level logger.Level, format string, args ...interface{}) {
	p.logger.Log(level, format, args...)
}
