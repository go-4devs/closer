package closer_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-4devs/closer"
)

func TestShutdown_NoTimeout(t *testing.T) {
	t.Parallel()

	for _, timeout := range []time.Duration{0, -time.Second} {
		closeFn := closer.Shutdown(func(ctx context.Context) error {
			return ctx.Err()
		}, timeout)

		err := closeFn()
		if err != nil {
			t.Fatalf("timeout %s: %v", timeout, err)
		}
	}
}
