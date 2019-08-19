package routine

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGo(t *testing.T) {
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
	require.Equal(t, 0, dt["once"])
	Go(fnc("once"))
	Run(fnc("twice"), fnc("twice"))
	Wait()
	require.Equal(t, 1, dt["once"])
	require.Equal(t, 2, dt["twice"])
}

func TestClose(t *testing.T) {
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
	Go(fnc("once"))
	Run(fnc("twice"), fnc("twice"))
	require.Nil(t, Close())
	require.Equal(t, 1, dt["once"])
	require.Equal(t, 2, dt["twice"])
}
