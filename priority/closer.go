package priority

import (
	"context"
	"math"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"
)

const (
	First  = 250
	Normal = 100
	Last   = 5
)

// Options configure closer
type Options func(*Closer)

// WithHandleError configure error handler
func WithHandleError(he func(error)) Options {
	return func(async *Closer) {
		async.ErrHandler = he
	}
}

// WithHandleError configure error handler
func WithTimeout(d time.Duration) Options {
	return func(async *Closer) {
		async.Timeout = d
	}
}

// New create new closer with options
func New(opts ...Options) *Closer {
	a := &Closer{}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Closer by priority
type Closer struct {
	sync.Mutex
	fnc        map[uint8][]func() error
	priority   prioritySlice
	once       sync.Once
	ErrHandler func(error)
	Timeout    time.Duration
	len        int
	d          chan struct{}
}

func (c *Closer) Add(f ...func() error) {
	c.AddByPriority(Normal, f...)
}

// AddLast add closer which execute at the end
func (c *Closer) AddLast(f ...func() error) {
	c.AddByPriority(Last, f...)
}

// AddLast add closer which execute at the begin
func (c *Closer) AddFirst(f ...func() error) {
	c.AddByPriority(First, f...)
}

// AddByPriority add close by priority 255 its close first 0 - last
func (c *Closer) AddByPriority(priority uint8, f ...func() error) {
	if len(f) == 0 {
		return
	}
	c.Lock()
	if c.fnc == nil {
		c.fnc = make(map[uint8][]func() error)
	}
	if len(c.fnc[priority]) == 0 {
		c.priority = append(c.priority, priority)
		sort.Sort(c.priority)
	}
	c.len += len(f)
	c.fnc[priority] = append(c.fnc[priority], f...)
	c.Unlock()
}

func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
	go func() {
		ch := make(chan os.Signal, 1)
		if len(sig) > 0 {
			signal.Notify(ch, sig...)
			defer signal.Stop(ch)
		}
		select {
		case <-ch:
		case <-ctx.Done():
		}
		_ = c.Close()
	}()
	<-c.done()
}

func (c *Closer) done() chan struct{} {
	c.Lock()
	if c.d == nil {
		c.d = make(chan struct{})
	}
	c.Unlock()
	return c.d
}

type wait struct {
	priority prioritySlice
	timeout  time.Duration
	sync.WaitGroup
}

func (w *wait) wait(start time.Time, idx int) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Wait()
	}()
	select {
	case <-w.after(start, idx):
	case <-done:
	}
}

func (w *wait) after(start time.Time, idx int) <-chan time.Time {
	timeout := w.timeout - time.Since(start)
	if len(w.priority) != idx+1 {
		timeout -= w.timeout / math.MaxUint8 * time.Duration(w.priority[idx+1])
	}
	if timeout <= 0 {
		return nil
	}

	return time.After(timeout)
}

func (c *Closer) Close() error {
	c.once.Do(func() {
		start := time.Now()
		c.Lock()
		eh := func(error) {}
		if c.ErrHandler != nil {
			eh = c.ErrHandler
		}
		w := &wait{
			priority: c.priority,
			timeout:  c.Timeout,
		}
		funcs := c.fnc
		c.fnc = nil
		c.Unlock()
		errs := make(chan error, c.len)
		go func() {
			defer close(c.done())
			for i := 0; i < cap(errs); i++ {
				err := <-errs
				if err != nil {
					go eh(err)
				}
			}
		}()
		for i, p := range c.priority {
			w.Add(len(funcs[p]))
			for _, f := range funcs[p] {
				go func(f func() error) {
					errs <- f()
					w.Done()
				}(f)
			}
			w.wait(start, i)
		}
		w.Wait()
	})
	return nil
}

type prioritySlice []uint8

func (p prioritySlice) Len() int           { return len(p) }
func (p prioritySlice) Less(i, j int) bool { return p[i] > p[j] }
func (p prioritySlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
