package internal

import "context"

// ReadErrs keeps errors already sent. A stuck func sends nothing, so a finished deadline ends the wait.
// shortcut: an error sent after this returns is dropped, upgrade if Close must include funcs that finish later.
func ReadErrs(deadline context.Context, errs <-chan error, handle func(error)) []error {
	var got []error

	for range cap(errs) {
		select {
		case <-deadline.Done():
			return drainErrs(got, errs, handle)
		case err := <-errs:
			got = keepErr(got, err, handle)
		}
	}

	return got
}

func drainErrs(got []error, errs <-chan error, handle func(error)) []error {
	for {
		select {
		case err := <-errs:
			got = keepErr(got, err, handle)
		default:
			return got
		}
	}
}

func keepErr(got []error, err error, handle func(error)) []error {
	if err == nil {
		return got
	}

	handle(err)

	return append(got, err)
}
