package priority

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ExampleNew() {
	cl := New(
		WithTimeout(time.Second*5),
		WithHandleError(func(e error) {
			log.Println(e)
		}),
	)
	db, _ := sql.Open("mysql", "mysql://localhost")
	cl.AddLast(db.Close)

	// your consumer events
	var consumer io.Closer
	cl.AddFirst(consumer.Close)

	s := http.Server{}
	// your http listeners
	var listener net.Listener
	go func() {
		if err := s.Serve(listener); err != nil {
			_ = cl.Close()
		}
	}()
	ctx := context.Background()

	cl.Add(func() error {
		return s.Shutdown(ctx)
	})

	cl.Wait(ctx, syscall.SIGTERM, syscall.SIGINT)
}

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
	cl.Add()
	cl.Add(closed.closeFnc("one", time.Microsecond))
	cl.AddLast(closed.closeFnc("last", time.Microsecond))
	cl.AddFirst(closed.closeFnc("first", time.Millisecond))
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
	cl.AddLast(func() error {
		return errors.New("error close")
	})

	time.AfterFunc(time.Microsecond, func() {
		require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
	})
	cl.Wait(context.Background(), syscall.SIGTERM)
	require.Equal(t, []string{"one", "last", "first"}, closed.d)
}

func TestCloser_Wait(t *testing.T) {
	cl := New(WithHandleError(func(err error) {
		require.EqualError(t, err, "error close")
	}))
	cl.AddLast(func() error {
		return errors.New("error close")
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cl.Wait(ctx)
}
