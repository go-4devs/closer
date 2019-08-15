package closer

import (
	"context"
	"io"
	"os"
)

//Closer interface
type Closer interface {
	Add(f ...func() error)
	Wait(ctx context.Context, s ...os.Signal)
	io.Closer
}
