package routine

import "sync"

var global = new(WaitGroup)

// Go run routine and add wait group.
func Go(fnc func()) {
	global.Go(fnc)
}

// Run run routines and add wait group.
func Run(fnc ...func()) {
	global.Run(fnc...)
}

// Close global routines.
func Close() error {
	return global.Close()
}

// Wait wait all go routines.
func Wait() {
	global.Wait()
}

// WaitGroup run func and wait when done.
type WaitGroup struct {
	mu      sync.Mutex
	waiting int

	sync.WaitGroup
}

// Close wait all routines and implement Closer.
func (wg *WaitGroup) Close() error {
	wg.Wait()

	return nil
}

// Go add wait group to routines.
func (wg *WaitGroup) Go(fnc func()) {
	wg.Run(fnc)
}

// Run functions in routine and add wait group.
func (wg *WaitGroup) Run(fnc ...func()) {
	if len(fnc) == 0 {
		return
	}

	wg.mu.Lock()
	// Add while Wait is in progress panics sync.WaitGroup.
	if wg.waiting > 0 {
		wg.mu.Unlock()

		return
	}

	wg.Add(len(fnc))
	wg.mu.Unlock()

	for i := range fnc {
		go func(i int) {
			defer wg.Done()

			fnc[i]()
		}(i)
	}
}

// Wait wait all go routines.
func (wg *WaitGroup) Wait() {
	wg.mu.Lock()
	wg.waiting++
	wg.mu.Unlock()

	wg.WaitGroup.Wait()

	wg.mu.Lock()
	wg.waiting--
	wg.mu.Unlock()
}
