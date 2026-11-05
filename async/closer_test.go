package async_test

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/go-4devs/closer/async"
	"github.com/go-4devs/closer/test"
)

func TestAsyncClose(t *testing.T) {
	t.Parallel()

	var a1closer async.Closer

	closedFn := new(test.Closed)
	test.RequireNil(t, a1closer.Add(closedFn.CloseFnc("one", 0)))
	test.RequireNil(t, a1closer.Close())
	closedFn.TestLen(t, 1)

	a1closer.Wait(context.Background())
	test.RequireNil(t, a1closer.Close())
	closedFn.TestLen(t, 1)

	asCloser := async.New(async.WithHandleError(func(err error) {
		if !errors.Is(err, test.ErrClose) {
			t.Fatalf("expect: %s, got:%s", test.ErrClose, err)
		}
	}))
	test.RequireNil(t, asCloser.Add(closedFn.CloseFnc("two", 0), func() error {
		test.RequireNil(t, closedFn.CloseFnc("two", 0)())

		return test.ErrClose
	}))
	test.RequireError(t, asCloser.Close(), test.ErrClose)
	closedFn.TestLen(t, 3)

	var a2closer async.Closer

	closedFn = &test.Closed{}
	test.RequireNil(t, a2closer.Add(
		closedFn.CloseFnc("one", time.Millisecond/2),
		closedFn.CloseFnc("two", time.Microsecond),
		closedFn.CloseFnc("three", time.Millisecond/5)))
	test.RequireNil(t, a2closer.Close())

	closedFn.TestLen(t, 3)
}

func TestAsyncWait_Timeout(t *testing.T) {
	t.Parallel()

	aCloser := async.New()

	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()

	ran := 0

	test.RequireNil(t, aCloser.Add(func() error {
		ran++

		return nil
	}))

	aCloser.Wait(ctx)

	if ran != 1 {
		t.Fatalf("close func ran %d times", ran)
	}
}

func TestAsyncWait_Syscall(t *testing.T) {
	t.Parallel()

	aCloser := async.New()
	tClosed := &test.Closed{}

	test.RequireNil(t, aCloser.Add(tClosed.CloseFnc("one", 0), tClosed.CloseFnc("one", 0)))

	time.AfterFunc(100*time.Millisecond, func() {
		test.RequireNil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})

	aCloser.Wait(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	tClosed.TestLen(t, 2)
}

func TestAsyncAddAfterClose(t *testing.T) {
	t.Parallel()

	aCloser := async.New()
	test.RequireNil(t, aCloser.Close())

	ran := false

	err := aCloser.Add(func() error {
		ran = true

		return nil
	})
	if !errors.Is(err, async.ErrClosed) {
		t.Fatalf("expected closed, got %v", err)
	}

	if ran {
		t.Fatal("closer added after Close ran")
	}
}

func TestAsyncClose_Timeout(t *testing.T) {
	t.Parallel()

	aCloser := async.New(
		async.WithTimeout(30*time.Millisecond),
		async.WithHandleError(func(error) {}),
	)
	test.RequireNil(t, aCloser.Add(func() error {
		return test.ErrClose
	}, func() error {
		<-make(chan struct{})

		return nil
	}))

	done := make(chan error, 1)

	go func() {
		done <- aCloser.Close()
	}()

	select {
	case err := <-done:
		test.RequireError(t, err, test.ErrClose)
	case <-time.After(time.Second):
		t.Fatal("Close blocked on the stuck func")
	}
}

func TestAsyncClose_Panic(t *testing.T) {
	t.Parallel()

	var panicErr error

	ran := false
	aCloser := async.New(async.WithHandleError(func(err error) {
		panicErr = err
	}))
	test.RequireNil(t, aCloser.Add(func() error {
		panic("boom")
	}, func() error {
		ran = true

		return nil
	}))

	done := make(chan struct{})

	go func() {
		_ = aCloser.Close()

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close blocked after panic")
	}

	test.RequireError(t, panicErr, async.ErrPanic)

	if !ran {
		t.Fatal("closer after panic did not run")
	}
}
