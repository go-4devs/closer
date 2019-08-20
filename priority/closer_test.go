package priority

import (
	"context"
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

func TestCloser_Close(t *testing.T) {
	cl := Closer{}
	closed := &closed{
		d: make([]string, 0),
	}
	cl.Add(closed.closeFnc("one", time.Microsecond))
	cl.AddByPriority(Last, closed.closeFnc("last", time.Microsecond))
	cl.AddByPriority(First, closed.closeFnc("first", time.Millisecond))
	cl.AddByPriority(Normal, closed.closeFnc("one", time.Microsecond), closed.closeFnc("one", time.Microsecond))
	require.Nil(t, cl.Close())
	require.Equal(t, []string{"first", "one", "one", "one", "last"}, closed.d)
}

func TestCloser_Wait_Timeout(t *testing.T) {
	t.Parallel()
	closed := &closed{
		d: make([]string, 0),
	}

	cl := New(WithTimeout(time.Second / 5))
	cl.Add(closed.closeFnc("one", 0))
	cl.AddByPriority(First, closed.closeFnc("first", time.Second))
	cl.AddByPriority(Last, closed.closeFnc("last", 0))

	time.AfterFunc(time.Microsecond, func() {
		require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	cl.Wait(context.Background(), syscall.SIGTERM)
	require.Equal(t, []string{"one", "last", "first"}, closed.d)
}
