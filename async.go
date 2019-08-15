package closer

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

// AsyncOptions configure closer
type AsyncOptions func(*Async)

// WithHandleError configure error handler
func WithHandleError(he func(error)) AsyncOptions {
	return func(async *Async) {
		async.he = he
	}
}

// NewAsync create new closer with options
func NewAsync(opts ...AsyncOptions) *Async {
	a := &Async{}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Async closer
type Async struct {
	sync.Mutex
	once sync.Once
	done chan struct{}
	fnc  []func() error
	he   func(error)
}

func (c *Async) Wait(ctx context.Context, sig ...os.Signal) {
	if c.done == nil {
		c.done = make(chan struct{})
	}
	go func() {
		ch := make(chan os.Signal, 1)
		if len(sig) > 0 {
			signal.Notify(ch, sig...)
			defer signal.Stop(ch)
		}
		select {
		case <-ch:
		case <-ctx.Done():
		}
		_ = c.Close()
		close(c.done)
	}()
	<-c.done
}

func (c *Async) Add(f ...func() error) {
	c.Lock()
	c.fnc = append(c.fnc, f...)
	c.Unlock()
}

func (c *Async) Close() error {
	c.once.Do(func() {
		if c.he == nil {
			c.he = func(e error) {}
		}
		c.Lock()
		funcs := c.fnc
		c.fnc = nil
		c.Unlock()

		errs := make(chan error, len(funcs))
		for _, f := range funcs {
			go func(f func() error) {
				errs <- f()
			}(f)
		}
		for i := 0; i < cap(errs); i++ {
			err := <-errs
			if err != nil {
				c.he(err)
			}
		}
	})

	return nil
}
