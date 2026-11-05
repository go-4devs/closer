package routine

import "sync"

var global = new(WaitGroup)

// Go runs fnc in a goroutine. A call during Wait is ignored.
func Go(fnc func()) {
	global.Go(fnc)
}

// Run runs each func in its own goroutine. Calls during Wait are ignored.
func Run(fnc ...func()) {
	global.Run(fnc...)
}

// Close waits for funcs started on the global group and returns nil.
func Close() error {
	return global.Close()
}

// Wait blocks until funcs started on the global group have returned.
func Wait() {
	global.Wait()
}

// WaitGroup runs funcs and waits for them. It can be used again after Wait returns.
type WaitGroup struct {
	sync.WaitGroup

	mu      sync.Mutex
	waiting int
}

// Close waits for funcs started before it and returns nil.
func (wg *WaitGroup) Close() error {
	wg.Wait()

	return nil
}

// Go runs fnc in a goroutine. A call during Wait is ignored.
func (wg *WaitGroup) Go(fnc func()) {
	wg.Run(fnc)
}

// Run runs each func in its own goroutine. Calls during Wait are ignored.
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

// Wait blocks until funcs started before this call have returned.
func (wg *WaitGroup) Wait() {
	wg.mu.Lock()
	wg.waiting++
	wg.mu.Unlock()

	wg.WaitGroup.Wait()

	wg.mu.Lock()
	wg.waiting--
	wg.mu.Unlock()
}
