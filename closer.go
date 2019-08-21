package closer

import (
	"context"
	"io"
	"os"
)

// Closer interface
type Closer interface {
	Add(f ...func() error)
	Wait(ctx context.Context, s ...os.Signal)
	io.Closer
}

// ByPriority close by priority
type ByPriority interface {
	Closer
	AddByPriority(priority uint8, f ...func() error)
}
