package closer

import (
	"context"
	"os"
	"time"

	"github.com/go-4devs/closer/priority"
)

var closer = priority.New()

// SetTimeout sets the deadline for the whole Close. A non-positive duration means no deadline.
func SetTimeout(t time.Duration) {
	closer.SetTimeout(t)
}

// SetErrHandler sets the func called for each close error. The default handler logs the error.
func SetErrHandler(eh func(error)) {
	closer.SetErrHandler(eh)
}

var (
	// ErrClosed is returned when a func is added after Close has started.
	ErrClosed = priority.ErrClosed

	// ErrPanic is returned when a close func panics.
	ErrPanic = priority.ErrPanic
)

// Add registers funcs at normal priority. It returns ErrClosed after Close has started.
func Add(f ...func() error) error {
	return closer.Add(f...)
}

// AddByPriority registers funcs at priority. A higher value runs first, from 255 down to 0.
// It returns ErrClosed after Close has started.
func AddByPriority(priority uint8, f ...func() error) error {
	return closer.AddByPriority(priority, f...)
}

// AddLast registers funcs near the end of Close. It returns ErrClosed after Close has started.
func AddLast(f ...func() error) error {
	return closer.AddLast(f...)
}

// AddFirst registers funcs near the start of Close. It returns ErrClosed after Close has started.
func AddFirst(f ...func() error) error {
	return closer.AddFirst(f...)
}

// Close runs registered funcs once, highest priority first, and returns their errors.
// A later Close returns the same error.
func Close() error {
	return closer.Close()
}

// Wait waits until ctx is canceled or a signal arrives, then runs Close.
// Canceling ctx does not skip the funcs.
func Wait(ctx context.Context, sig ...os.Signal) {
	closer.Wait(ctx, sig...)
}
