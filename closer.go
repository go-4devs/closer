package closer

import (
	"context"
	"os"
	"time"

	"gitoa.ru/go-4devs/closer/priority"
)

// nolint: gochecknoglobals
var closer = &priority.Closer{}

// SetTimeout before close func.
func SetTimeout(t time.Duration) {
	closer.SetTimeout(t)
}

// SetErrHandler before close func.
func SetErrHandler(eh func(error)) {
	closer.SetErrHandler(eh)
}

// Add add closed func.
func Add(f ...func() error) {
	closer.Add(f...)
}

// AddByPriority add close by priority 255 its close first 0 - last.
func AddByPriority(priority uint8, f ...func() error) {
	closer.AddByPriority(priority, f...)
}

// AddLast add closer which execute at the end.
func AddLast(f ...func() error) {
	closer.AddLast(f...)
}

// AddFirst add closer which execute at the begin.
func AddFirst(f ...func() error) {
	closer.AddFirst(f...)
}

// Close all func.
// nolint: wrapcheck
func Close() error {
	return closer.Close()
}

// Wait cancel ctx or signal.
func Wait(ctx context.Context, sig ...os.Signal) {
	closer.Wait(ctx, sig...)
}
