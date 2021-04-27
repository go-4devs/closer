package async_test

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"gitoa.ru/go-4devs/closer/async"
	"gitoa.ru/go-4devs/closer/test"
)

func ExampleCloser_Close() {
	cl := async.New()

	cl.Add(func() error {
		fmt.Print("do some close")

		return nil
	})

	cl.Close()
	// Output: do some close
}

func ExampleCloser_Wait_cancelContext() {
	cl := async.New()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
	defer cancel()

	cl.Add(func() error {
		fmt.Print("do some close with cancel context")

		return nil
	})

	cl.Wait(ctx)
	// Output: do some close with cancel context
}

func ExampleWithHandleError() {
	cl := async.New(async.WithHandleError(func(err error) {
		fmt.Printf("logged err:%s", err)
	}))

	cl.Add(func() error {
		return test.ErrClose
	})

	cl.Close()
	// Output: logged err:some error
}

func ExampleCloser_Wait_syscall() {
	time.AfterFunc(time.Millisecond, func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	})

	cl := async.New()

	cl.Add(func() error {
		fmt.Print("do some close with SIGTERM")

		return nil
	})

	cl.Wait(context.TODO(), syscall.SIGTERM)
	// Output: do some close with SIGTERM
}
