package core

import (
	"context"
	"fmt"
	"time"
)

type Core struct {
	product   string
	ctx       context.Context
	ctxCancel func()

	// out
	done chan struct{}
}

func NewCore(params map[string]interface{}) *Core {
	ctx, ctxCancel := context.WithCancel(context.Background())
	c := &Core{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		done:      make(chan struct{}, 1),
	}
	for k, v := range params {
		switch k {
		case "product":
			if val, ok := v.(string); ok {
				c.product = val
			}
		}
	}
	return c
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
outer:
	for {
		select {
		case <-p.ctx.Done():
			break outer
		default:
			time.Sleep(1 * time.Second)
			fmt.Println("hello")
			// p.Log(logger.Info, "hello")
		}
	}
	p.ctxCancel()
}
