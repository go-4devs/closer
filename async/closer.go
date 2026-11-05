package async

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

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

// WithHandleError sets the func called for each close error. The default handler logs the error.
func WithHandleError(he func(error)) Option {
	return func(async *Closer) {
		async.handler = he
	}
}

// WithTimeout sets the deadline for the whole Close. A non-positive duration means no deadline.
func WithTimeout(d time.Duration) Option {
	return func(closer *Closer) {
		closer.timeout = d
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
	timeout  time.Duration
	closed   bool
	closeErr error
}

// Wait waits until ctx is canceled or a signal arrives, then runs Close.
// Canceling ctx does not skip the funcs.
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
	internal.Notify(ctx, c.doneChan(), func() { _ = c.shutdown(ctx) }, sig...)
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
// After the deadline, errors already sent are kept and a func still running is left out.
func (c *Closer) Close() error {
	return c.shutdown(context.Background())
}

// SetTimeout sets the deadline for the whole Close. A non-positive duration means no deadline.
func (c *Closer) SetTimeout(t time.Duration) {
	c.mu.Lock()
	c.timeout = t
	c.mu.Unlock()
}

// SetErrHandler sets the func called for each close error. The default handler logs the error.
func (c *Closer) SetErrHandler(e func(error)) {
	c.mu.Lock()
	c.handler = e
	c.mu.Unlock()
}

func (c *Closer) shutdown(ctx context.Context) error {
	// The context only chooses when Close starts. Its cancel must not skip the funcs.
	ctx = context.WithoutCancel(ctx)

	c.once.Do(func() {
		defer close(c.doneChan())

		c.mu.Lock()

		errHandler := func(err error) {
			log.Print(err)
		}
		if c.handler != nil {
			errHandler = c.handler
		}

		timeout := c.timeout
		c.closed = true
		funcs := c.fnc
		c.fnc = nil
		c.mu.Unlock()

		deadline := ctx

		if timeout > 0 {
			var cancel context.CancelFunc

			deadline, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		errs := make(chan error, len(funcs))

		for _, f := range funcs {
			go func(f func() error) {
				errs <- internal.Recover(f)
			}(f)
		}

		c.closeErr = errors.Join(internal.ReadErrs(deadline, errs, errHandler)...)
	})

	return c.closeErr
}

func (c *Closer) doneChan() chan struct{} {
	return internal.DoneChan(&c.mu, &c.done)
}
