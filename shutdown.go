package closer

import (
	"context"
	"time"
)

func Shutdown(fnc func(context.Context) error, timeout time.Duration) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		return fnc(ctx)
	}
}
