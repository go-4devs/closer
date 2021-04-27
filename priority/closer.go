package priority

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"
)

// defaults.
const (
	First  = 250
	Normal = 100
	Last   = 5
)

// Option configure priority closer.
type Option func(*Closer)

// WithHandleError configure priority error handler.
func WithHandleError(he func(error)) Option {
	return func(async *Closer) {
		async.handler = he
	}
}

// WithTimeout configure priority error handler.
func WithTimeout(d time.Duration) Option {
	return func(async *Closer) {
		async.timeout = d
	}
}

// New create new closer with options.
func New(opts ...Option) *Closer {
	a := &Closer{}

	for _, o := range opts {
		o(a)
	}

	return a
}

// Closer close by priority.
type Closer struct {
	sync.Mutex
	fnc         map[uint8][]func() error
	priority    prioritySlice
	sumPriority uint
	once        sync.Once
	handler     func(error)
	timeout     time.Duration
	len         int
	done        chan struct{}
}

// Add adds closed func by normal priority.
func (c *Closer) Add(f ...func() error) {
	c.AddByPriority(Normal, f...)
}

// AddLast add closer which execute at the end.
func (c *Closer) AddLast(f ...func() error) {
	c.AddByPriority(Last, f...)
}

// AddFirst add closer which execute at the begin.
func (c *Closer) AddFirst(f ...func() error) {
	c.AddByPriority(First, f...)
}

// AddByPriority add close by priority 255 its close first 0 - last.
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
		c.sumPriority += uint(priority)
	}

	c.len += len(f)
	c.fnc[priority] = append(c.fnc[priority], f...)
	c.Unlock()
}

// Wait wait signal or cancel context.
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
	<-c.wait()
}

// Close closes all func by priority.
func (c *Closer) Close() error {
	c.once.Do(func() {
		start := time.Now()
		c.Lock()
		eh := func(err error) {
			log.Print(err)
		}

		if c.handler != nil {
			eh = c.handler
		}

		w := &wait{
			timeout:  c.timeout,
			priority: c.sumPriority,
		}
		funcs := c.fnc
		c.fnc = nil
		c.Unlock()
		errs := make(chan error, c.len)
		go func() {
			defer close(c.wait())
			for i := 0; i < cap(errs); i++ {
				if err := <-errs; err != nil {
					eh(err)
				}
			}
		}()
		for _, p := range c.priority {
			var wg sync.WaitGroup
			wg.Add(len(funcs[p]))
			for _, f := range funcs[p] {
				go func(f func() error) {
					errs <- f()
					wg.Done()
				}(f)
			}
			w.done(start, uint(p), &wg)
		}
		<-c.wait()
	})

	return nil
}

// SetTimeout before close func.
func (c *Closer) SetTimeout(t time.Duration) {
	c.Lock()
	c.timeout = t
	c.Unlock()
}

// SetErrHandler before close func.
func (c *Closer) SetErrHandler(e func(error)) {
	c.Lock()
	c.handler = e
	c.Unlock()
}

func (c *Closer) wait() chan struct{} {
	c.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	c.Unlock()

	return c.done
}

type wait struct {
	timeout  time.Duration
	priority uint
}

func (w *wait) done(start time.Time, priority uint, wg *sync.WaitGroup) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-w.after(start, priority):
	case <-done:
	}
}

func (w *wait) after(start time.Time, priority uint) <-chan time.Time {
	timeout := (w.timeout - time.Since(start)) / time.Duration(w.priority) * time.Duration(priority)
	w.priority -= priority

	if timeout <= 0 {
		return nil
	}

	return time.After(timeout)
}

type prioritySlice []uint8

func (p prioritySlice) Len() int           { return len(p) }
func (p prioritySlice) Less(i, j int) bool { return p[i] > p[j] }
func (p prioritySlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
