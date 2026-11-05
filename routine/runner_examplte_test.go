package routine_test

import (
	"fmt"
	"time"

	"github.com/go-4devs/closer/routine"
)

func ExampleGo() {
	defer routine.Close()

	routine.Go(func() {
		time.Sleep(time.Microsecond)
		fmt.Print("do some job")
	})

	// Output: do some job
}

func ExampleRun() {
	defer routine.Close()

	printed := make(chan struct{})

	routine.Run(func() {
		<-printed
		fmt.Print("do some job. ")
	}, func() {
		fmt.Print("fast job in goroutine. ")
		close(printed)
	})

	// Output: fast job in goroutine. do some job.
}
