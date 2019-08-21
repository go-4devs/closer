package global

import (
	"context"
	"os"
	"time"

	"github.com/go-4devs/closer/priority"
)

// nolint: gochecknoglobals
var closer = &priority.Closer{}

// SetTimeout before close func
func SetTimeout(t time.Duration) {
	closer.Timeout = t
}

// SetErrHandler before close func
func SetErrHandler(eh func(error)) {
	closer.ErrHandler = eh
}

// Add add closed func
func Add(f ...func() error) {
	closer.Add(f...)
}

// AddByPriority add func by priority
func AddByPriority(priority uint8, f ...func() error) {
	closer.AddByPriority(priority, f...)
}

// AddLast add closer which execute at the end
func AddLast(f ...func() error) {
	closer.AddLast(f...)
}

// AddLast add closer which execute at the begin
func AddFirst(f ...func() error) {
	closer.AddFirst(f...)
}

// Close all func
func Close() error {
	return closer.Close()
}

// Wait cancel ctx or signal
func Wait(ctx context.Context, sig ...os.Signal) {
	closer.Wait(ctx, sig...)
}
