package routine

import "sync"

// nolint: gochecknoglobals
var global = &WaitGroup{}

// Go run routines and add wait group
func Go(fnc ...func()) {
	global.Go(fnc...)
}

// Wait wait all go routines
func Wait() {
	global.Wait()
}

// WaitGroup run func and wait when done
type WaitGroup struct {
	wg sync.WaitGroup
}

// Wait wait all routines
func (r *WaitGroup) Wait() {
	r.wg.Wait()
}

// Close wait all routines and implement Closer
func (r *WaitGroup) Close() error {
	r.Wait()
	return nil
}

// Go add wait group to routines
func (r *WaitGroup) Go(fnc ...func()) {
	r.wg.Add(len(fnc))
	for i := range fnc {
		go func(i int) {
			defer r.wg.Done()
			fnc[i]()
		}(i)
	}
}
