package async

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"

	"github.com/go-4devs/closer/internal"
)

var (
	// ErrClosed is returned when a func is added after Close has started.
	ErrClosed = internal.ErrClosed

	// ErrPanic is returned when a close func panics.
	ErrPanic = internal.ErrPanic
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
	closer := new(Closer)

	for _, o := range opts {
		o(closer)
	}

	return closer
}

// Closer closer.
type Closer struct {
	sync.Mutex

	once     sync.Once
	done     chan struct{}
	fnc      []func() error
	handler  func(error)
	closed   bool
	closeErr error
}

// Wait when done context or notify signals or close all.
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
	go func() {
		chs := make(chan os.Signal, 1)

		if len(sig) > 0 {
			signal.Notify(chs, sig...)
			defer signal.Stop(chs)
		}

		select {
		case <-chs:
		case <-ctx.Done():
		}

		_ = c.Close()
	}()

	<-c.wait()
}

// Add close functions.
func (c *Closer) Add(funcs ...func() error) error {
	if len(funcs) == 0 {
		return nil
	}

	c.Lock()
	defer c.Unlock()

	if c.closed {
		return ErrClosed
	}

	c.fnc = append(c.fnc, funcs...)

	return nil
}

// Close close all closers async.
func (c *Closer) Close() error {
	c.once.Do(func() {
		defer close(c.wait())

		c.Lock()

		errHandler := func(error) {}
		if c.handler != nil {
			errHandler = c.handler
		}

		c.closed = true
		funcs := c.fnc
		c.fnc = nil
		c.Unlock()

		errs := make(chan error, len(funcs))

		for _, f := range funcs {
			go func(f func() error) {
				errs <- internal.Recover(f)
			}(f)
		}

		var got []error

		for range cap(errs) {
			err := <-errs
			if err != nil {
				errHandler(err)
				got = append(got, err)
			}
		}

		c.closeErr = errors.Join(got...)
	})

	return c.closeErr
}

func (c *Closer) SetErrHandler(e func(error)) {
	c.Lock()
	c.handler = e
	c.Unlock()
}

func (c *Closer) wait() chan struct{} {
	c.Lock()

	if c.done == nil {
		c.done = make(chan struct{})
	}

	c.Unlock()

	return c.done
}
