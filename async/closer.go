package async

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

// Option configure async closer.
type Option func(*Closer)

// WithHandleError configure async error handler.
func WithHandleError(he func(error)) Option {
	return func(async *Closer) {
		async.handler = he
	}
}

// New create new closer with options.
func New(opts ...Option) *Closer {
	a := &Closer{}

	for _, o := range opts {
		o(a)
	}

	return a
}

// Closer closer.
type Closer struct {
	sync.Mutex
	once    sync.Once
	done    chan struct{}
	fnc     []func() error
	handler func(error)
}

// Wait when done context or notify signals or close all.
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
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
	<-c.wait()
}

func (c *Closer) wait() chan struct{} {
	c.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	c.Unlock()

	return c.done
}

// Add close functions.
func (c *Closer) Add(f ...func() error) {
	c.Lock()
	c.fnc = append(c.fnc, f...)
	c.Unlock()
}

// Close close all closers async.
func (c *Closer) Close() error {
	c.once.Do(func() {
		defer close(c.wait())
		c.Lock()
		eh := func(error) {}
		if c.handler != nil {
			eh = c.handler
		}
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
			if err := <-errs; err != nil {
				eh(err)
			}
		}
	})

	return nil
}

func (c *Closer) SetErrHandler(e func(error)) {
	c.Lock()
	c.handler = e
	c.Unlock()
}
