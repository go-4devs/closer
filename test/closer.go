package test

import (
	"errors"
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
	if len(c.d) != exp {
		t.Fail()
	}
}

func (c *Closed) TestEqual(t *testing.T, exp []string) {
	if len(c.d) != len(exp) {
		t.Fail()
	}

	for i := range exp {
		if exp[i] != c.d[i] {
			t.Fail()
		}
	}
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

func RequireNil(t *testing.T, exp interface{}) {
	t.Helper()

	if exp != nil {
		t.Fatalf("expected nil, got %v", exp)
	}
}

func RequireError(t *testing.T, exp, target error) {
	t.Helper()

	if !errors.Is(exp, target) {
		t.Fatalf("expected %s, got %s", exp, target)
	}
}
