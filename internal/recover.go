package internal

import (
	"errors"
	"fmt"
)

var (
	// ErrClosed is returned when a func is added after Close has started.
	ErrClosed = errors.New("closer is closed")

	// ErrPanic is returned when a close func panics.
	ErrPanic = errors.New("panic")
)

// Recover runs closeFn and turns a panic into ErrPanic.
func Recover(closeFn func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: %v", ErrPanic, recovered)
		}
	}()

	return closeFn()
}
