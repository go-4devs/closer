package routine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGo(t *testing.T) {
	var run int

	fnc := func() {
		time.Sleep(time.Millisecond)
		run++
	}
	Go(fnc, fnc)
	require.Equal(t, 0, run)
	Wait()
	require.Equal(t, 2, run)
}

func TestWaitGroup_Close(t *testing.T) {
	var run int
	fnc := func() {
		time.Sleep(time.Millisecond)
		run++
	}
	wg := WaitGroup{}
	wg.Go(fnc, fnc)
	require.Nil(t, wg.Close())
	require.Equal(t, 2, run)
}
