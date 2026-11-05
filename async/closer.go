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

// Option configures an async Closer.
type Option func(*Closer)

// WithHandleError sets the func called for each close error. Without it, Close still returns the errors.
func WithHandleError(he func(error)) Option {
	return func(async *Closer) {
		async.handler = he
	}
}

// New returns a closer that runs funcs in parallel.
func New(opts ...Option) *Closer {
	closer := new(Closer)

	for _, o := range opts {
		o(closer)
	}

	return closer
}

// Closer runs registered funcs once, all at the same time.
type Closer struct {
	mu sync.Mutex

	once     sync.Once
	done     chan struct{}
	fnc      []func() error
	handler  func(error)
	closed   bool
	closeErr error
}

// Wait waits until ctx is canceled or a signal arrives, then runs Close.
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
	internal.Notify(ctx, c.doneChan(), func() { _ = c.Close() }, sig...)
}

// Add registers funcs. It returns ErrClosed after Close has started.
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

// Close runs registered funcs once, in parallel, and returns their errors.
// A panic becomes ErrPanic. A later Close returns the same error.
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

// SetErrHandler sets the func called for each close error.
func (c *Closer) SetErrHandler(e func(error)) {
	c.mu.Lock()
	c.handler = e
	c.mu.Unlock()
}

func (c *Closer) doneChan() chan struct{} {
	return internal.DoneChan(&c.mu, &c.done)
}
