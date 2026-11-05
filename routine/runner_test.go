package routine_test

import (
	"sync"
	"testing"
	"time"

	"github.com/go-4devs/closer/routine"
)

func equal(t *testing.T, exp, res int) {
	t.Helper()

	if exp != res {
		t.Fail()
	}
}

func TestGo(t *testing.T) {
	t.Parallel()

	data := make(map[string]int)

	var fmu sync.Mutex

	fnc := func(name string) func() {
		return func() {
			time.Sleep(time.Millisecond)
			fmu.Lock()

			data[name]++

			fmu.Unlock()
		}
	}

	equal(t, 0, data["once"])

	wg := new(routine.WaitGroup)
	wg.Go(fnc("once"))
	wg.Run(fnc("twice"), fnc("twice"))
	wg.Wait()

	equal(t, 1, data["once"])
	equal(t, 2, data["twice"])
}

func TestClose(t *testing.T) {
	t.Parallel()

	data := make(map[string]int)

	var fmu sync.Mutex

	fnc := func(name string) func() {
		return func() {
			time.Sleep(time.Millisecond)
			fmu.Lock()

			data[name]++

			fmu.Unlock()
		}
	}
	wg := new(routine.WaitGroup)
	wg.Go(fnc("once"))
	wg.Run(fnc("twice"), fnc("twice"))

	err := wg.Close()
	if err != nil {
		t.Fail()
	}

	equal(t, 1, data["once"])
	equal(t, 2, data["twice"])
}

func TestGoDuringWait(t *testing.T) {
	t.Parallel()

	wg := new(routine.WaitGroup)
	for range 2000 {
		go wg.Go(func() {})

		wg.Wait()
	}
}
