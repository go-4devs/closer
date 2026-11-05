package test

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

var ErrClose = errors.New("some error")

type Closed struct {
	mu sync.Mutex
	d  []string
}

func (c *Closed) TestLen(t *testing.T, exp int) {
	t.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.d) == exp {
		return
	}

	t.Fatalf("got %d (%v), want %d", len(c.d), c.d, exp)
}

func (c *Closed) TestEqual(t *testing.T, exp []string) {
	t.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()

	if slices.Equal(c.d, exp) {
		return
	}

	t.Fatalf("got %v, want %v", c.d, exp)
}

func (c *Closed) CloseFnc(name string, sleep time.Duration) func() error {
	return func() error {
		time.Sleep(sleep)
		c.mu.Lock()
		c.d = append(c.d, name)
		c.mu.Unlock()

		return nil
	}
}

func RequireNil(t *testing.T, exp any) {
	t.Helper()

	if exp != nil {
		t.Fatalf("expected nil, got %v", exp)
	}
}

func RequireError(t *testing.T, exp, target error) {
	t.Helper()

	if !errors.Is(exp, target) {
		t.Fatalf("expected %s, got %s", target, exp)
	}
}
