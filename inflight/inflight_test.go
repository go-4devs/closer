package inflight_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-4devs/closer/inflight"
)

func TestWait_Empty(t *testing.T) {
	t.Parallel()

	err := inflight.New().Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestWait_BlocksUntilDoReturns(t *testing.T) {
	t.Parallel()

	group := inflight.New()
	release := make(chan struct{})
	started := make(chan struct{})
	doErr := make(chan error, 1)

	go func() {
		doErr <- group.Do(context.Background(), func(context.Context) error {
			close(started)
			<-release

			return nil
		})
	}()

	<-started

	waitErr := make(chan error, 1)

	go func() {
		waitErr <- group.Wait(context.Background())
	}()

	select {
	case err := <-waitErr:
		t.Fatalf("wait returned before do finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	err := <-waitErr
	if err != nil {
		t.Fatal(err)
	}

	err = <-doErr
	if err != nil {
		t.Fatal(err)
	}
}

func TestWait_CancelsOnTimeout(t *testing.T) {
	t.Parallel()

	group := inflight.New()
	started := make(chan struct{})
	doErr := make(chan error, 1)

	go func() {
		doErr <- group.Do(context.Background(), func(ctx context.Context) error {
			close(started)
			<-ctx.Done()

			return ctx.Err()
		})
	}()

	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := group.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait: %v", err)
	}

	err = <-doErr
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("do: %v", err)
	}
}

func TestDo_ReturnsError(t *testing.T) {
	t.Parallel()

	err := inflight.New().Do(context.Background(), func(context.Context) error {
		return io.EOF
	})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("got %v", err)
	}
}

func TestWait_WaitsForEveryCall(t *testing.T) {
	t.Parallel()

	group := inflight.New()
	first := make(chan struct{})
	second := make(chan struct{})
	started := make(chan struct{}, 2)
	finished := make(chan struct{})
	doErr := make(chan error, 2)

	go func() {
		doErr <- group.Do(context.Background(), func(context.Context) error {
			started <- struct{}{}

			<-first

			return nil
		})

		close(finished)
	}()

	go func() {
		doErr <- group.Do(context.Background(), func(context.Context) error {
			started <- struct{}{}

			<-second

			return nil
		})
	}()

	<-started
	<-started

	waitErr := make(chan error, 1)

	go func() {
		waitErr <- group.Wait(context.Background())
	}()

	close(first)
	<-finished

	select {
	case err := <-waitErr:
		t.Fatalf("wait returned while a call was running: %v", err)
	default:
	}

	close(second)

	err := <-waitErr
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		err = <-doErr
		if err != nil {
			t.Fatal(err)
		}
	}
}
