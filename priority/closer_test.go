package priority_test

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/go-4devs/closer/priority"
	"github.com/go-4devs/closer/test"
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

func TestPriority_Zero(t *testing.T) {
	t.Parallel()

	onlyZero := priority.New()
	closedZero := &test.Closed{}
	onlyZero.AddByPriority(0, closedZero.CloseFnc("zero", 0))
	test.RequireNil(t, onlyZero.Close())
	closedZero.TestEqual(t, []string{"zero"})

	pcl := priority.New(priority.WithTimeout(time.Second))
	closed := &test.Closed{}
	pcl.Add(closed.CloseFnc("one", 0))
	pcl.AddByPriority(0, closed.CloseFnc("zero", 0))
	test.RequireNil(t, pcl.Close())
	closed.TestEqual(t, []string{"one", "zero"})
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
	// "first" sleeps past the timeout, so shutdown must not wait for it.
	closed.TestEqual(t, []string{"one", "last"})
}

func TestPriority_Close_Timeout(t *testing.T) {
	t.Parallel()

	closed := &test.Closed{}
	pcl := priority.New(priority.WithTimeout(30 * time.Millisecond))
	pcl.AddByPriority(priority.First, func() error {
		<-make(chan struct{})

		return nil
	})
	pcl.Add(closed.CloseFnc("one", 0))

	done := make(chan struct{})
	go func() {
		_ = pcl.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close blocked past timeout")
	}

	closed.TestEqual(t, []string{"one"})
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

func TestPriority_AddAfterClose(t *testing.T) {
	t.Parallel()

	pcl := priority.New()
	started := make(chan struct{})
	hold := make(chan struct{})
	test.RequireNil(t, pcl.Add(func() error {
		close(started)
		<-hold

		return nil
	}))

	done := make(chan struct{})
	go func() {
		_ = pcl.Close()
		close(done)
	}()
	<-started

	ran := false
	err := pcl.Add(func() error {
		ran = true

		return nil
	})
	test.RequireError(t, err, priority.ErrClosed)

	close(hold)
	<-done

	if ran {
		t.Fatal("closer added during Close ran")
	}

	err = pcl.Add(func() error {
		ran = true

		return nil
	})
	test.RequireError(t, err, priority.ErrClosed)

	if ran {
		t.Fatal("closer added after Close ran")
	}
}

func TestPriority_AddCloseRace(t *testing.T) {
	t.Parallel()

	for range 20 {
		pcl := priority.New()
		hold := make(chan struct{})
		test.RequireNil(t, pcl.Add(func() error {
			<-hold

			return nil
		}))

		done := make(chan struct{})
		go func() {
			_ = pcl.Close()
			close(done)
		}()
		_ = pcl.AddByPriority(1, func() error { return nil })
		close(hold)
		<-done
	}
}
