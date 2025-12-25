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
	acl := async.New()

	acl.Add(func() error {
		fmt.Print("do some close")

		return nil
	})

	acl.Close()
	// Output: do some close
}

func ExampleCloser_Wait_cancelContext() {
	acl := async.New()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
	defer cancel()

	acl.Add(func() error {
		fmt.Print("do some close with cancel context")

		return nil
	})

	acl.Wait(ctx)
	// Output: do some close with cancel context
}

func ExampleWithHandleError() {
	acl := async.New(async.WithHandleError(func(err error) {
		fmt.Printf("logged err:%s", err)
	}))

	acl.Add(func() error {
		return test.ErrClose
	})

	acl.Close()
	// Output: logged err:some error
}

func ExampleCloser_Wait_syscall() {
	time.AfterFunc(time.Millisecond, func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	})

	acl := async.New()

	acl.Add(func() error {
		fmt.Print("do some close with SIGTERM")

		return nil
	})

	acl.Wait(context.TODO(), syscall.SIGTERM)
	// Output: do some close with SIGTERM
}
