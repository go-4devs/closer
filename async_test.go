package closer

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAsync_Add(t *testing.T) {
	c := Async{}
	c.Add(func() error { return nil })
	require.Len(t, c.fnc, 1)
}

func TestAsync_Close(t *testing.T) {
	c := Async{}
	var cnt int
	c.Add(func() error {
		cnt++
		return nil
	})
	require.Nil(t, c.Close())
	require.Equal(t, 1, cnt)

	require.Nil(t, c.Close())
	require.Equal(t, 1, cnt)

	var e int
	as := NewAsync(WithHandleError(func(error) {
		e++
	}))
	as.Add(func() error { cnt++; return nil }, func() error { cnt++; return errors.New("some error") })
	require.Nil(t, as.Close())
	require.Equal(t, 3, cnt)
	require.Equal(t, 1, e)

	c = Async{}
	var res string
	tc := func(d string, t time.Duration) func() error {
		return func() error { time.Sleep(t); res += d; return nil }
	}
	c.Add(tc("one", time.Second/2), tc("two", time.Microsecond), tc("three", time.Second/3))
	require.Nil(t, c.Close())
	require.Equal(t, "twothreeone", res)
}

func TestAsync_Wait(t *testing.T) {
	c := Async{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()
	var cnt int
	cl := func() error { cnt++; return nil }
	c.Add(cl)
	c.Wait(ctx)
	require.Equal(t, 1, cnt)

	c = Async{}
	time.AfterFunc(time.Microsecond, func() {
		require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	c.Add(cl)
	c.Wait(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	require.Equal(t, 2, cnt)
}
