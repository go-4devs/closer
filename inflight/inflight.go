// Package inflight tracks calls that are already running.
package inflight

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// Group tracks calls that are already running.
// Wait returns when they finish. When the Wait context ends, their contexts are canceled.
type Group struct {
	mu    sync.Mutex
	idle  chan struct{}
	calls []*call
}

type call struct {
	cancel context.CancelFunc
}

// New returns an empty group.
func New() *Group {
	return &Group{
		mu:    sync.Mutex{},
		idle:  nil,
		calls: nil,
	}
}

// Do runs fnc with a context that Wait cancels when its own context ends.
// A call that has not entered Do is not seen by Wait.
func (g *Group) Do(parent context.Context, fnc func(context.Context) error) error {
	ctx, cancel := context.WithCancel(parent)
	current := &call{cancel: cancel}

	g.mu.Lock()
	g.calls = append(g.calls, current)
	g.mu.Unlock()

	defer func() {
		cancel()

		g.mu.Lock()
		g.calls = slices.DeleteFunc(g.calls, func(item *call) bool { return item == current })

		if len(g.calls) == 0 && g.idle != nil {
			close(g.idle)
			g.idle = nil
		}

		g.mu.Unlock()
	}()

	return fnc(ctx)
}

// Wait blocks until calls from Do finish.
// When ctx ends, those calls are canceled and ctx.Err is returned.
func (g *Group) Wait(ctx context.Context) error {
	g.mu.Lock()

	if len(g.calls) == 0 {
		g.mu.Unlock()

		return nil
	}

	idle := make(chan struct{})
	g.idle = idle
	g.mu.Unlock()

	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		g.cancel()

		return fmt.Errorf("wait:%w", ctx.Err())
	}
}

func (g *Group) cancel() {
	g.mu.Lock()
	calls := slices.Clone(g.calls)
	g.mu.Unlock()

	for _, item := range calls {
		item.cancel()
	}
}
