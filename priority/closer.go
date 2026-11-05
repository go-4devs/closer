package priority

import (
	"context"
	"errors"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-4devs/closer/internal"
)

// First, Normal and Last are close priorities. A higher value runs earlier.
const (
	First  = 250
	Normal = 100
	Last   = 5
)

var (
	// ErrClosed is returned when a func is added after Close has started.
	ErrClosed = internal.ErrClosed

	// ErrPanic is returned when a close func panics.
	ErrPanic = internal.ErrPanic
)

// Option configures a priority Closer.
type Option func(*Closer)

// WithHandleError sets the func called for each close error. The default handler logs the error.
func WithHandleError(he func(error)) Option {
	return func(async *Closer) {
		async.handler = he
	}
}

// WithTimeout sets the deadline for the whole Close. A non-positive duration means no deadline.
func WithTimeout(d time.Duration) Option {
	return func(async *Closer) {
		async.timeout = d
	}
}

// New returns a closer that runs funcs by priority.
func New(opts ...Option) *Closer {
	closer := new(Closer)

	for _, o := range opts {
		o(closer)
	}

	return closer
}

// Closer runs registered funcs once, highest priority first.
type Closer struct {
	mu sync.Mutex

	fnc         map[uint8][]func() error
	priority    prioritySlice
	sumPriority int64
	once        sync.Once
	handler     func(error)
	timeout     time.Duration
	len         int
	done        chan struct{}
	closed      bool
	closeErr    error
}

// Add registers funcs at normal priority. It returns ErrClosed after Close has started.
func (c *Closer) Add(f ...func() error) error {
	return c.AddByPriority(Normal, f...)
}

// AddLast registers funcs at Last. It returns ErrClosed after Close has started.
func (c *Closer) AddLast(f ...func() error) error {
	return c.AddByPriority(Last, f...)
}

// AddFirst registers funcs at First. It returns ErrClosed after Close has started.
func (c *Closer) AddFirst(f ...func() error) error {
	return c.AddByPriority(First, f...)
}

// AddByPriority registers funcs at priority. A higher value runs first, from 255 down to 0.
// It returns ErrClosed after Close has started.
func (c *Closer) AddByPriority(priority uint8, fnc ...func() error) error {
	if len(fnc) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClosed
	}

	if c.fnc == nil {
		c.fnc = make(map[uint8][]func() error)
	}

	if len(c.fnc[priority]) == 0 {
		c.priority = append(c.priority, priority)
		sort.Sort(c.priority)
		c.sumPriority += int64(priority)
	}

	c.len += len(fnc)
	c.fnc[priority] = append(c.fnc[priority], fnc...)

	return nil
}

// Wait waits until ctx is canceled or a signal arrives, then runs Close.
// Canceling ctx does not skip the funcs.
func (c *Closer) Wait(ctx context.Context, sig ...os.Signal) {
	internal.Notify(ctx, c.doneChan(), func() { _ = c.shutdown(ctx) }, sig...)
}

// Close runs registered funcs once, highest priority first, and returns their errors.
// A panic becomes ErrPanic. A later Close returns the same error.
func (c *Closer) Close() error {
	return c.shutdown(context.Background())
}

// SetTimeout sets the deadline for the whole Close. A non-positive duration means no deadline.
func (c *Closer) SetTimeout(t time.Duration) {
	c.mu.Lock()
	c.timeout = t
	c.mu.Unlock()
}

// SetErrHandler sets the func called for each close error. The default handler logs the error.
func (c *Closer) SetErrHandler(e func(error)) {
	c.mu.Lock()
	c.handler = e
	c.mu.Unlock()
}

func (c *Closer) shutdown(ctx context.Context) error {
	// The context only chooses when Close starts. Its cancel must not skip the funcs.
	ctx = context.WithoutCancel(ctx)

	c.once.Do(func() {
		start := time.Now()

		c.mu.Lock()

		errHandler := func(err error) {
			log.Print(err)
		}

		if c.handler != nil {
			errHandler = c.handler
		}

		waitByPriority := &wait{
			timeout:  c.timeout,
			priority: c.sumPriority,
		}
		c.closed = true
		funcs := c.fnc
		order := append(prioritySlice(nil), c.priority...)
		count := c.len
		c.fnc = nil
		c.mu.Unlock()

		deadline := ctx

		if waitByPriority.timeout > 0 {
			var cancel context.CancelFunc

			deadline, cancel = context.WithTimeout(ctx, waitByPriority.timeout)
			defer cancel()
		}

		errs := make(chan error, count)

		var got []error

		go func() {
			defer close(c.doneChan())

			// A stuck func never sends, so the deadline has to unblock the wait.
			for range cap(errs) {
				select {
				case <-deadline.Done():
					return
				case err := <-errs:
					if err != nil {
						errHandler(err)
						got = append(got, err)
					}
				}
			}
		}()

		for _, pri := range order {
			if deadline.Err() != nil {
				break
			}

			var wgPri sync.WaitGroup

			wgPri.Add(len(funcs[pri]))

			for _, f := range funcs[pri] {
				go func(f func() error) {
					defer wgPri.Done()

					errs <- internal.Recover(f)
				}(f)
			}

			waitByPriority.done(deadline, start, int64(pri), &wgPri)
		}

		<-c.doneChan()
		c.closeErr = errors.Join(got...)
	})

	return c.closeErr
}

func (c *Closer) doneChan() chan struct{} {
	return internal.DoneChan(&c.mu, &c.done)
}

type wait struct {
	timeout  time.Duration
	priority int64
}

func (w *wait) done(ctx context.Context, start time.Time, priority int64, priWg *sync.WaitGroup) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		priWg.Wait()
	}()

	select {
	case <-ctx.Done():
	case <-w.after(start, priority):
	case <-done:
	}
}

func (w *wait) after(start time.Time, priority int64) <-chan time.Time {
	// Priority 0 is a valid bucket and adds nothing to the sum.
	if w.priority == 0 {
		return nil
	}

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
