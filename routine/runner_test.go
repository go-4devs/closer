package routine

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGo(t *testing.T) {
	var run int
	var mu sync.Mutex
	fnc := func() {
		time.Sleep(time.Millisecond)
		mu.Lock()
		run++
		mu.Unlock()
	}
	Go(fnc, fnc)
	mu.Lock()
	require.Equal(t, 0, run)
	mu.Unlock()
	Wait()
	require.Equal(t, 2, run)
}

func TestWaitGroup_Close(t *testing.T) {
	var run int
	var mu sync.Mutex
	fnc := func() {
		time.Sleep(time.Millisecond)
		mu.Lock()
		run++
		mu.Unlock()
	}
	wg := WaitGroup{}
	wg.Go(fnc, fnc)
	require.Nil(t, wg.Close())
	require.Equal(t, 2, run)
}
