package async

import (
	"context"
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type closed struct {
	mu sync.Mutex
	d  []string
}

func (c *closed) closeFnc(name string, sleep time.Duration) func() error {
	return func() error {
		time.Sleep(sleep)
		c.mu.Lock()
		c.d = append(c.d, name)
		c.mu.Unlock()
		return nil
	}
}

func TestAsync_Add(t *testing.T) {
	c := Closer{}
	c.Add(func() error { return nil })
	require.Len(t, c.fnc, 1)
}

func TestAsync_Close(t *testing.T) {
	c := Closer{}
	closedFn := &closed{}
	c.Add(closedFn.closeFnc("one", 0))
	require.Nil(t, c.Close())
	require.Equal(t, []string{"one"}, closedFn.d)
	c.Wait(context.Background())
	require.Nil(t, c.Close())
	require.Equal(t, []string{"one"}, closedFn.d)

	as := New(WithHandleError(func(e error) {
		require.EqualError(t, e, "some error")
	}))
	as.Add(closedFn.closeFnc("two", 0), func() error {
		require.Nil(t, closedFn.closeFnc("two", 0)())
		return errors.New("some error")
	})
	require.Nil(t, as.Close())
	require.Len(t, closedFn.d, 3)

	c = Closer{}
	closedFn = &closed{}
	c.Add(
		closedFn.closeFnc("one", time.Millisecond/2),
		closedFn.closeFnc("two", time.Microsecond),
		closedFn.closeFnc("three", time.Millisecond/5))
	require.Nil(t, c.Close())
	require.Len(t, closedFn.d, 3)
}

func TestAsync_Wait_Syscall(t *testing.T) {
	c := New()
	cl := &closed{}
	time.AfterFunc(time.Microsecond, func() {
		require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	c.Add(cl.closeFnc("one", 0), cl.closeFnc("one", 0))
	c.Wait(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	require.Len(t, cl.d, 2)
}

func TestAsync_Wait(t *testing.T) {
	c := Closer{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()
	var cnt int
	go func() {
		c.Wait(ctx)
		cnt++
	}()
	cl := func() error { cnt++; return nil }
	c.Add(cl)
	c.Wait(context.Background())
	require.Equal(t, 1, cnt)
}
