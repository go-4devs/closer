package closer

import (
	"context"
	"time"
)

func Shutdown(fnc func(context.Context) error, timeout time.Duration) func() error {
	return func() error {
		// A non-positive timeout is no deadline. WithTimeout(0) is already canceled.
		if timeout <= 0 {
			return fnc(context.Background())
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		return fnc(ctx)
	}
}
