package async

import (
	"context"
	"errors"
	"os"
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
	mu sync.Mutex

	once     sync.Once
	done     chan struct{}
	fnc      []func() error
	handler  func(error)
	closed   bool
	closeErr error
}

// Wait when done context or notify signals or close all.
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
	internal.Notify(ctx, c.doneChan(), func() { _ = c.Close() }, sig...)
}

// Add close functions.
func (c *Closer) Add(funcs ...func() error) error {
	if len(funcs) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClosed
	}

	c.fnc = append(c.fnc, funcs...)

	return nil
}

// Close close all closers async.
func (c *Closer) Close() error {
	c.once.Do(func() {
		defer close(c.doneChan())

		c.mu.Lock()

		errHandler := func(error) {}
		if c.handler != nil {
			errHandler = c.handler
		}

		c.closed = true
		funcs := c.fnc
		c.fnc = nil
		c.mu.Unlock()

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
	c.mu.Lock()
	c.handler = e
	c.mu.Unlock()
}

func (c *Closer) doneChan() chan struct{} {
	return internal.DoneChan(&c.mu, &c.done)
}
