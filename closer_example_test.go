package closer_test

import (
	"context"
	"fmt"
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
	}, func() error {
		time.Sleep(time.Millisecond)
		// long normal stop.

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

	_ = closer.AddByPriority(priority.First+1, func() error {
		// run before first.
		return nil
	})

	_ = closer.AddByPriority(priority.Normal-1, func() error {
		// run after normal.
		return nil
	})

	_ = closer.AddByPriority(priority.Last-1, func() error {
		return test.ErrClose
	})

	closer.Wait(ctx, syscall.SIGTERM)

	err := closer.Add(func() error {
		fmt.Print("must not run")

		return nil
	})
	fmt.Print("\n", err)
	// Output:
	// close http server for example. do some close. close db for example.
	// logged err:some error
	// closer is closed
}
