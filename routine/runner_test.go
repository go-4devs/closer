package routine_test

import (
	"sync"
	"testing"
	"time"

	"gitoa.ru/go-4devs/closer/routine"
)

func equal(t *testing.T, exp, res int) {
	t.Helper()

	if exp != res {
		t.Fail()
	}
}

func TestGo(t *testing.T) {
	t.Parallel()

	dt := make(map[string]int)

	var mu sync.Mutex

	fnc := func(name string) func() {
		return func() {
			time.Sleep(time.Millisecond)
			mu.Lock()
			dt[name]++
			mu.Unlock()
		}
	}

	equal(t, 0, dt["once"])

	routine.Go(fnc("once"))
	routine.Run(fnc("twice"), fnc("twice"))
	routine.Wait()

	equal(t, 1, dt["once"])
	equal(t, 2, dt["twice"])
}

func TestClose(t *testing.T) {
	t.Parallel()

	dt := make(map[string]int)

	var mu sync.Mutex

	fnc := func(name string) func() {
		return func() {
			time.Sleep(time.Millisecond)
			mu.Lock()
			dt[name]++
			mu.Unlock()
		}
	}
	routine.Go(fnc("once"))
	routine.Run(fnc("twice"), fnc("twice"))

	if err := routine.Close(); err != nil {
		t.Fail()
	}

	equal(t, 1, dt["once"])
	equal(t, 2, dt["twice"])
}
