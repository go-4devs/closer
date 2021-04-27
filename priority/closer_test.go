package priority_test

import (
	"context"
	"syscall"
	"testing"
	"time"

	"gitoa.ru/go-4devs/closer/priority"
	"gitoa.ru/go-4devs/closer/test"
)

func TestPriority_Close(t *testing.T) {
	t.Parallel()

	cl := priority.Closer{}
	closed := &test.Closed{}

	go cl.Wait(context.Background())

	cl.Add()
	cl.Add(closed.CloseFnc("one", time.Microsecond))
	cl.AddLast(closed.CloseFnc("last", time.Microsecond))
	cl.AddFirst(closed.CloseFnc("first", time.Millisecond))
	cl.AddByPriority(priority.Normal, closed.CloseFnc("one", time.Microsecond), closed.CloseFnc("one", time.Microsecond))
	test.RequireNil(t, cl.Close())
	closed.TestEqual(t, []string{"first", "one", "one", "one", "last"})
}

func TestPriority_Wait_Timeout(t *testing.T) {
	t.Parallel()

	closed := &test.Closed{}

	cl := priority.New(priority.WithTimeout(time.Second/5), priority.WithHandleError(func(error) {}))
	cl.Add(closed.CloseFnc("one", 0))
	cl.AddByPriority(priority.First, closed.CloseFnc("first", time.Second))
	cl.AddByPriority(priority.Last, closed.CloseFnc("last", 0))
	cl.AddLast(func() error {
		return test.ErrClose
	})

	time.AfterFunc(time.Second/3, func() {
		test.RequireNil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	cl.Wait(context.Background(), syscall.SIGTERM)
	closed.TestEqual(t, []string{"one", "last", "first"})
}

func TestPriority_Wait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cl := priority.New(priority.WithHandleError(func(err error) {
		test.RequireError(t, err, test.ErrClose)
	}))
	cl.AddLast(func() error {
		return test.ErrClose
	})
	cl.Wait(ctx)
}
