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

	routine.Go(fnc("once"))
	routine.Run(fnc("twice"), fnc("twice"))
	routine.Wait()

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
	routine.Go(fnc("once"))
	routine.Run(fnc("twice"), fnc("twice"))

	err := routine.Close()
	if err != nil {
		t.Fail()
	}

	equal(t, 1, data["once"])
	equal(t, 2, data["twice"])
}
