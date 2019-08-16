package async

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

// Options configure closer
type Options func(*Closer)

// WithHandleError configure error handler
func WithHandleError(he func(error)) Options {
	return func(async *Closer) {
		async.he = he
	}
}

// New create new closer with options
func New(opts ...Options) *Closer {
	a := &Closer{}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Closer closer
type Closer struct {
	sync.Mutex
	once sync.Once
	done chan struct{}
	fnc  []func() error
	he   func(error)
}

// Wait when done context or notify signals or close all
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
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
	}()
	<-c.done
}

// Add close functions
func (c *Closer) Add(f ...func() error) {
	c.Lock()
	c.fnc = append(c.fnc, f...)
	c.Unlock()
}

// Close close all closers async
func (c *Closer) Close() error {
	if c.done == nil {
		c.done = make(chan struct{})
	}
	c.once.Do(func() {
		defer close(c.done)
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
