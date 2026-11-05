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

	time.AfterFunc(100*time.Millisecond, func() {
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

func TestPriority_Close_TimeoutKeepsSentErrors(t *testing.T) {
	t.Parallel()

	// One pass can still read the buffered error by luck. Repeat so a dropped error fails the test.
	for range 12 {
		seen, closeErr := closeWithStuck(t)
		test.RequireError(t, closeErr, test.ErrClose)

		if seen != 2 {
			t.Fatalf("seen %d, want 2", seen)
		}
	}
}

func closeWithStuck(t *testing.T) (int, error) {
	t.Helper()

	const finished = 2

	release := make(chan struct{})
	started := make(chan struct{})
	arrived := make(chan struct{}, finished)
	seen := 0
	pcl := priority.New(
		priority.WithTimeout(15*time.Millisecond),
		priority.WithHandleError(func(err error) {
			test.RequireError(t, err, test.ErrClose)

			seen++

			if seen == 1 {
				close(started)

				<-release
			}
		}),
	)
	funcs := make([]func() error, 0, finished+1)

	for range finished {
		funcs = append(funcs, func() error {
			arrived <- struct{}{}

			return test.ErrClose
		})
	}

	funcs = append(funcs, func() error {
		<-make(chan struct{})

		return nil
	})
	test.RequireNil(t, pcl.Add(funcs...))

	done := make(chan error, 1)

	go func() {
		done <- pcl.Close()
	}()

	waitChan(t, arrived, finished)
	waitChan(t, started, 1)
	time.Sleep(50 * time.Millisecond)
	close(release)

	select {
	case err := <-done:
		return seen, err
	case <-time.After(time.Second):
		t.Fatal("stuck func blocked Close")

		return seen, nil
	}
}

func waitChan(t *testing.T, ch <-chan struct{}, n int) {
	t.Helper()

	for range n {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("close func did not finish")
		}
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
