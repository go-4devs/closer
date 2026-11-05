package inflight_test

import (
	"context"
	"fmt"

	"github.com/go-4devs/closer/inflight"
)

func ExampleGroup_Do() {
	group := inflight.New()
	started := make(chan struct{})

	go func() {
		_ = group.Do(context.Background(), func(context.Context) error {
			close(started)
			fmt.Print("ran. ")

			return nil
		})
	}()

	<-started

	err := group.Wait(context.Background())
	if err != nil {
		fmt.Print(err)

		return
	}

	fmt.Print("waited")
	// Output: ran. waited
}

func ExampleGroup_Wait() {
	group := inflight.New()
	started := make(chan struct{})

	go func() {
		_ = group.Do(context.Background(), func(ctx context.Context) error {
			close(started)
			<-ctx.Done()

			return ctx.Err()
		})
	}()

	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := group.Wait(ctx)
	fmt.Print(err)
	// Output: wait:context canceled
}
