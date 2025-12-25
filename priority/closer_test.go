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

	var pcl priority.Closer

	closed := &test.Closed{}

	go pcl.Wait(context.Background())

	pcl.Add()
	pcl.Add(closed.CloseFnc("one", time.Microsecond))
	pcl.AddLast(closed.CloseFnc("last", time.Microsecond))
	pcl.AddFirst(closed.CloseFnc("first", time.Millisecond))
	pcl.AddByPriority(
		priority.Normal,
		closed.CloseFnc("one", time.Microsecond),
		closed.CloseFnc("one", time.Microsecond),
	)
	test.RequireNil(t, pcl.Close())
	closed.TestEqual(t, []string{"first", "one", "one", "one", "last"})
}

func TestPriority_Wait_Timeout(t *testing.T) {
	t.Parallel()

	closed := &test.Closed{}

	pcl := priority.New(
		priority.WithTimeout(time.Second/5),
		priority.WithHandleError(func(error) {}),
	)
	pcl.Add(closed.CloseFnc("one", 0))
	pcl.AddByPriority(priority.First, closed.CloseFnc("first", time.Second))
	pcl.AddByPriority(priority.Last, closed.CloseFnc("last", 0))
	pcl.AddLast(func() error {
		return test.ErrClose
	})

	time.AfterFunc(time.Second/3, func() {
		test.RequireNil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	pcl.Wait(context.Background(), syscall.SIGTERM)
	closed.TestEqual(t, []string{"one", "last", "first"})
}

func TestPriority_Wait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pcl := priority.New(priority.WithHandleError(func(err error) {
		test.RequireError(t, err, test.ErrClose)
	}))
	pcl.AddLast(func() error {
		return test.ErrClose
	})
	pcl.Wait(ctx)
}
