package priority_test

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/go-4devs/closer/priority"
	"github.com/go-4devs/closer/test"
)

const (
	nameLast = "last"
	nameOne  = "one"
	nameZero = "zero"
)

func TestPriority_Close(t *testing.T) {
	t.Parallel()

	var pcl priority.Closer

	closed := &test.Closed{}

	go pcl.Wait(context.Background())

	test.RequireNil(t, pcl.Add())
	test.RequireNil(t, pcl.Add(closed.CloseFnc(nameOne, time.Microsecond)))
	test.RequireNil(t, pcl.AddLast(closed.CloseFnc(nameLast, time.Microsecond)))
	test.RequireNil(t, pcl.AddFirst(closed.CloseFnc("first", time.Millisecond)))
	test.RequireNil(t, pcl.AddByPriority(
		priority.Normal,
		closed.CloseFnc(nameOne, time.Microsecond),
		closed.CloseFnc(nameOne, time.Microsecond),
	))
	test.RequireNil(t, pcl.Close())
	closed.TestEqual(t, []string{"first", nameOne, nameOne, nameOne, nameLast})
}

func TestPriority_Zero(t *testing.T) {
	t.Parallel()

	onlyZero := priority.New()
	closedZero := &test.Closed{}
	test.RequireNil(t, onlyZero.AddByPriority(0, closedZero.CloseFnc(nameZero, 0)))
	test.RequireNil(t, onlyZero.Close())
	closedZero.TestEqual(t, []string{nameZero})

	pcl := priority.New(priority.WithTimeout(time.Second))
	closed := &test.Closed{}
	test.RequireNil(t, pcl.Add(closed.CloseFnc(nameOne, 0)))
	test.RequireNil(t, pcl.AddByPriority(0, closed.CloseFnc(nameZero, 0)))
	test.RequireNil(t, pcl.Close())
	closed.TestEqual(t, []string{nameOne, nameZero})
}

func TestPriority_Wait_Timeout(t *testing.T) {
	t.Parallel()

	closed := &test.Closed{}

	pcl := priority.New(
		priority.WithTimeout(time.Second/5),
		priority.WithHandleError(func(error) {}),
	)
	test.RequireNil(t, pcl.Add(closed.CloseFnc(nameOne, 0)))
	test.RequireNil(t, pcl.AddByPriority(priority.First, closed.CloseFnc("first", time.Second)))
	test.RequireNil(t, pcl.AddByPriority(priority.Last, closed.CloseFnc(nameLast, 0)))
	test.RequireNil(t, pcl.AddLast(func() error {
		return test.ErrClose
	}))

	time.AfterFunc(time.Second/3, func() {
		test.RequireNil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	pcl.Wait(context.Background(), syscall.SIGTERM)
	// "first" sleeps past the timeout, so shutdown must not wait for it.
	closed.TestEqual(t, []string{nameOne, nameLast})
}

func TestPriority_Close_Timeout(t *testing.T) {
	t.Parallel()

	closed := &test.Closed{}
	pcl := priority.New(priority.WithTimeout(30 * time.Millisecond))
	test.RequireNil(t, pcl.AddByPriority(priority.First, func() error {
		<-make(chan struct{})

		return nil
	}))
	test.RequireNil(t, pcl.Add(closed.CloseFnc(nameOne, 0)))

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

	closed.TestEqual(t, []string{nameOne})
}

func TestPriority_Close_Panic(t *testing.T) {
	t.Parallel()

	var panicErr error

	ran := false
	pcl := priority.New(priority.WithHandleError(func(err error) {
		panicErr = err
	}))
	test.RequireNil(t, pcl.AddByPriority(priority.First, func() error {
		panic("boom")
	}))
	test.RequireNil(t, pcl.Add(func() error {
		ran = true

		return nil
	}))

	done := make(chan struct{})

	var closeErr error

	go func() {
		closeErr = pcl.Close()

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close blocked after panic")
	}

	test.RequireError(t, closeErr, priority.ErrPanic)
	test.RequireError(t, panicErr, priority.ErrPanic)

	if !ran {
		t.Fatal("closer after panic did not run")
	}
}

func TestPriority_Wait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pcl := priority.New(priority.WithHandleError(func(err error) {
		test.RequireError(t, err, test.ErrClose)
	}))
	test.RequireNil(t, pcl.AddLast(func() error {
		return test.ErrClose
	}))
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
