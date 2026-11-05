package internal

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

// DoneChan returns the lazy done channel, creating it under mu.
func DoneChan(mu *sync.Mutex, done *chan struct{}) chan struct{} {
	mu.Lock()
	defer mu.Unlock()

	if *done == nil {
		*done = make(chan struct{})
	}

	return *done
}

// Notify waits until ctx is canceled or a signal arrives, runs closeFn, and blocks until done is closed.
func Notify(ctx context.Context, done <-chan struct{}, closeFn func(), sig ...os.Signal) {
	go func(ctx context.Context) {
		chs := make(chan os.Signal, 1)

		if len(sig) > 0 {
			signal.Notify(chs, sig...)
			defer signal.Stop(chs)
		}

		select {
		case <-chs:
		case <-ctx.Done():
		}

		closeFn()
	}(ctx)

	<-done
}
