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

	pcl := priority.New(
		priority.WithTimeout(time.Second*5),
		priority.WithHandleError(func(e error) {
			fmt.Print(e)
		}),
	)
	db := OpenDB()
	pcl.AddLast(db.Close)

	// your consumer events
	consumer := NewConsumer()
	pcl.AddFirst(consumer.Close)

	var (
		srv    http.Server
		listen net.ListenConfig
	)
	// your http listeners
	listener, _ := listen.Listen(ctx, "tcp", "127.0.0.1:0")

	go func() {
		err := srv.Serve(listener)
		if err != nil {
			_ = pcl.Close()
		}
	}()

	pcl.Add(func() error {
		fmt.Print("stop server. ")

		return srv.Shutdown(ctx)
	})

	pcl.Wait(ctx, syscall.SIGTERM, syscall.SIGINT)
	// Output: close consumer. stop server. close db.
}

func ExampleCloser_Close() {
	pcl := priority.New()

	pcl.Add(func() error {
		fmt.Print("normal stop. ")

		return nil
	}, func() error {
		time.Sleep(time.Millisecond)
		fmt.Print("long normal stop. ")

		return nil
	})

	pcl.AddFirst(func() error {
		fmt.Print("first stop. ")

		return nil
	})
	pcl.AddLast(func() error {
		fmt.Print("last stop. ")

		return nil
	})

	pcl.AddByPriority(priority.First+1, func() error {
		fmt.Print("run before first. ")

		return nil
	})
	pcl.AddByPriority(priority.Normal-1, func() error {
		fmt.Print("run after normal. ")

		return nil
	})

	pcl.Close()
	// Output: run before first. first stop. normal stop. long normal stop. run after normal. last stop.
}
