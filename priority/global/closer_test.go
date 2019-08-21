package global

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type closed struct {
	mu sync.Mutex
	d  []string
}

func (c *closed) closeFnc(name string) func() error {
	return func() error {
		c.mu.Lock()
		c.d = append(c.d, name)
		c.mu.Unlock()
		return nil
	}
}

func TestClose(t *testing.T) {
	cl := &closed{}
	SetErrHandler(func(e error) {
		require.EqualError(t, e, "error close")
	})
	SetTimeout(time.Second)
	Add(cl.closeFnc("one"), cl.closeFnc("one"))
	AddByPriority(50, cl.closeFnc("two"), cl.closeFnc("two"))
	AddLast(cl.closeFnc("last"))
	AddFirst(func() error {
		return errors.New("error close")
	})

	time.AfterFunc(time.Second/2, func() {
		require.Nil(t, Close())
	})

	Wait(context.Background())
}
