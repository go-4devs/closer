package closer_test

import (
	"context"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/go-4devs/closer"
	"github.com/go-4devs/closer/priority"
	"github.com/go-4devs/closer/test"
)

func ExampleWait() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
	defer cancel()

	closer.SetErrHandler(func(err error) {
		fmt.Print("\nlogged err:", err.Error())
	})

	_ = closer.Add(func() error {
		fmt.Print("do some close. ")

		return nil
	})

	_ = closer.AddFirst(func() error {
		fmt.Print("close http server for example. ")

		return nil
	})

	_ = closer.AddLast(func() error {
		fmt.Print("close db for example.")

		return nil
	})

	_ = closer.AddByPriority(priority.Last-1, func() error {
		return test.ErrClose
	})

	closer.Wait(ctx)
	// Output:
	// close http server for example. do some close. close db for example.
	// logged err:some error
}

func ExampleClose() {
	_ = closer.Add(func() error {
		// normal stop.
		return nil
	}, func() error {
		time.Sleep(time.Millisecond)
		// long normal stop.

		return nil
	})

	_ = closer.AddFirst(func() error {
		// first stop.
		return nil
	})
	_ = closer.AddLast(func() error {
		// last stop.
		return nil
	})

	_ = closer.AddByPriority(priority.First+1, func() error {
		// run before first.
		return nil
	})

	_ = closer.AddByPriority(priority.Normal-1, func() error {
		// run after normal.
		return nil
	})

	closer.Close()
	// Output:
}

func ExampleWait_cancelContext() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond)
	defer cancel()

	_ = closer.Add(func() error {
		// do some close with cancel context
		return nil
	})

	closer.Wait(ctx)
	// Output:
}

func ExampleSetErrHandler() {
	closer.SetErrHandler(func(err error) {
		log.Print("logged err:", err.Error())
	})

	_ = closer.Add(func() error {
		return test.ErrClose
	})

	closer.Close()
	// Output:
}

func ExampleWait_syscall() {
	time.AfterFunc(time.Millisecond, func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	})

	_ = closer.Add(func() error {
		// do some close with SIGTERM
		return nil
	})

	closer.Wait(context.TODO(), syscall.SIGTERM)
	// Output:
}
