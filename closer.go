package closer

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-4devs/closer/priority"
)

var closer = priority.New()

// SetTimeout before close func.
func SetTimeout(t time.Duration) {
	closer.SetTimeout(t)
}

// SetErrHandler before close func.
func SetErrHandler(eh func(error)) {
	closer.SetErrHandler(eh)
}

// ErrClosed is returned when a func is added after Close has started.
var ErrClosed = priority.ErrClosed

// Add add closed func.
func Add(f ...func() error) error {
	return wrap(closer.Add(f...))
}

// AddByPriority add close by priority 255 its close first 0 - last.
func AddByPriority(priority uint8, f ...func() error) error {
	return wrap(closer.AddByPriority(priority, f...))
}

// AddLast add closer which execute at the end.
func AddLast(f ...func() error) error {
	return wrap(closer.AddLast(f...))
}

// AddFirst add closer which execute at the begin.
func AddFirst(f ...func() error) error {
	return wrap(closer.AddFirst(f...))
}

// Close all func.
func Close() error {
	return wrap(closer.Close())
}

func wrap(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%w", err)
}

// Wait cancel ctx or signal.
func Wait(ctx context.Context, sig ...os.Signal) {
	closer.Wait(ctx, sig...)
}
