package priority

import (
	"context"
	"os"
	"sync"
)

const (
	First  = 250
	Normal = 100
	Last   = 5
)

type Closer struct {
	sync.Mutex
	fnc map[uint8][]func() error
}

func (c *Closer) Add(f ...func() error) {
	c.AddByPriority(Normal, f...)
}

func (c *Closer) AddByPriority(priority uint8, f ...func() error) {
	c.Lock()
	if c.fnc == nil {
		c.fnc = make(map[uint8][]func() error)
	}
	c.fnc[priority] = append(c.fnc[priority], f...)
	c.Unlock()
}

func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {

}

func (c *Closer) Close() error {
	return nil
}
