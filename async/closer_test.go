package async_test

import (
	"context"
	"errors"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"gitoa.ru/go-4devs/closer/async"
	"gitoa.ru/go-4devs/closer/test"
)

func TestAsyncClose(t *testing.T) {
	t.Parallel()

	var a1closer async.Closer

	closedFn := new(test.Closed)
	a1closer.Add(closedFn.CloseFnc("one", 0))
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
	asCloser.Add(closedFn.CloseFnc("two", 0), func() error {
		test.RequireNil(t, closedFn.CloseFnc("two", 0)())

		return test.ErrClose
	})
	test.RequireNil(t, asCloser.Close())
	closedFn.TestLen(t, 3)

	var a2closer async.Closer

	closedFn = &test.Closed{}
	a2closer.Add(
		closedFn.CloseFnc("one", time.Millisecond/2),
		closedFn.CloseFnc("two", time.Microsecond),
		closedFn.CloseFnc("three", time.Millisecond/5))
	test.RequireNil(t, a2closer.Close())

	closedFn.TestLen(t, 3)
}

func TestAsyncWait_Timeout(t *testing.T) {
	t.Parallel()

	aCloser := async.New()

	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()

	var cnt int32

	go func() {
		aCloser.Wait(ctx)
		atomic.AddInt32(&cnt, 1)
	}()

	closeFn := func() error {
		atomic.AddInt32(&cnt, 1)

		return nil
	}

	aCloser.Add(closeFn)
	aCloser.Wait(context.Background())

	if atomic.LoadInt32(&cnt) != 1 {
		t.Fail()
	}
}

func TestAsyncWait_Syscall(t *testing.T) {
	t.Parallel()

	aCloser := async.New()
	tClosed := &test.Closed{}

	time.AfterFunc(time.Second, func() {
		test.RequireNil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})

	aCloser.Add(tClosed.CloseFnc("one", 0), tClosed.CloseFnc("one", 0))
	aCloser.Wait(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	tClosed.TestLen(t, 2)
}
