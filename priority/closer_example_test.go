package priority_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"gitoa.ru/go-4devs/closer/priority"
)

type closer struct {
	msg string
}

// nolint: forbidigo
func (c *closer) Close() error {
	fmt.Print(c.msg)

	return nil
}

func OpenDB() io.Closer {
	return &closer{msg: "close db. "}
}

func NewConsumer() io.Closer {
	return &closer{msg: "close consumer. "}
}

func ExampleCloser_Wait() {
	time.AfterFunc(time.Second/2, func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	})

	ctx := context.Background()

	cl := priority.New(
		priority.WithTimeout(time.Second*5),
		priority.WithHandleError(func(e error) {
			fmt.Print(e)
		}),
	)
	db := OpenDB()
	cl.AddLast(db.Close)

	// your consumer events
	consumer := NewConsumer()
	cl.AddFirst(consumer.Close)

	s := http.Server{}
	// your http listeners
	listener, _ := net.Listen("tcp", "127.0.0.1:0")

	go func() {
		if err := s.Serve(listener); err != nil {
			_ = cl.Close()
		}
	}()

	cl.Add(func() error {
		fmt.Print("stop server. ")

		return s.Shutdown(ctx)
	})

	cl.Wait(ctx, syscall.SIGTERM, syscall.SIGINT)
	// Output: close consumer. stop server. close db.
}

func ExampleCloser_Close() {
	cl := priority.New()

	cl.Add(func() error {
		fmt.Print("normal stop. ")

		return nil
	}, func() error {
		time.Sleep(time.Millisecond)
		fmt.Print("long normal stop. ")

		return nil
	})

	cl.AddFirst(func() error {
		fmt.Print("first stop. ")

		return nil
	})
	cl.AddLast(func() error {
		fmt.Print("last stop. ")

		return nil
	})

	cl.AddByPriority(priority.First+1, func() error {
		fmt.Print("run before first. ")

		return nil
	})
	cl.AddByPriority(priority.Normal-1, func() error {
		fmt.Print("run after normal. ")

		return nil
	})

	cl.Close()
	// Output: run before first. first stop. normal stop. long normal stop. run after normal. last stop.
}
