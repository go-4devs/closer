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

	c := async.Closer{}
	closedFn := &test.Closed{}
	c.Add(closedFn.CloseFnc("one", 0))
	test.RequireNil(t, c.Close())
	closedFn.TestLen(t, 1)

	c.Wait(context.Background())
	test.RequireNil(t, c.Close())
	closedFn.TestLen(t, 1)

	as := async.New(async.WithHandleError(func(e error) {
		if !errors.Is(e, test.ErrClose) {
			t.Fatalf("expect: %s, got:%s", test.ErrClose, e)
		}
	}))
	as.Add(closedFn.CloseFnc("two", 0), func() error {
		test.RequireNil(t, closedFn.CloseFnc("two", 0)())

		return test.ErrClose
	})
	test.RequireNil(t, as.Close())
	closedFn.TestLen(t, 3)

	c = async.Closer{}
	closedFn = &test.Closed{}
	c.Add(
		closedFn.CloseFnc("one", time.Millisecond/2),
		closedFn.CloseFnc("two", time.Microsecond),
		closedFn.CloseFnc("three", time.Millisecond/5))
	test.RequireNil(t, c.Close())

	closedFn.TestLen(t, 3)
}

func TestAsyncWait_Timeout(t *testing.T) {
	t.Parallel()

	c := &async.Closer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()

	var cnt int32

	go func() {
		c.Wait(ctx)
		atomic.AddInt32(&cnt, 1)
	}()

	cl := func() error {
		atomic.AddInt32(&cnt, 1)

		return nil
	}

	c.Add(cl)
	c.Wait(context.Background())

	if atomic.LoadInt32(&cnt) != 1 {
		t.Fail()
	}
}

func TestAsyncWait_Syscall(t *testing.T) {
	t.Parallel()

	c := async.New()
	cl := &test.Closed{}

	time.AfterFunc(time.Second, func() {
		test.RequireNil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})

	c.Add(cl.CloseFnc("one", 0), cl.CloseFnc("one", 0))
	c.Wait(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	cl.TestLen(t, 2)
}
