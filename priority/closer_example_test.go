package priority_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/go-4devs/closer/priority"
	"github.com/go-4devs/closer/test"
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
	ctx := context.Background()

	pcl := priority.New(
		priority.WithTimeout(time.Second*5),
		priority.WithHandleError(func(e error) {
			fmt.Print(e)
		}),
	)
	db := OpenDB()
	_ = pcl.AddLast(db.Close)

	// your consumer events
	consumer := NewConsumer()
	_ = pcl.AddFirst(consumer.Close)

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

	_ = pcl.Add(func() error {
		fmt.Print("stop server. ")

		return srv.Shutdown(ctx)
	})

	time.AfterFunc(100*time.Millisecond, func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	})

	pcl.Wait(ctx, syscall.SIGTERM, syscall.SIGINT)
	// Output: close consumer. stop server. close db.
}

func ExampleCloser_Close() {
	pcl := priority.New()

	normalDone := make(chan struct{})

	_ = pcl.Add(func() error {
		fmt.Print("normal stop. ")
		close(normalDone)

		return nil
	}, func() error {
		<-normalDone
		fmt.Print("long normal stop. ")

		return nil
	})

	_ = pcl.AddFirst(func() error {
		fmt.Print("first stop. ")

		return nil
	})
	_ = pcl.AddLast(func() error {
		fmt.Print("last stop. ")

		return nil
	})

	_ = pcl.AddByPriority(priority.First+1, func() error {
		fmt.Print("run before first. ")

		return nil
	})
	_ = pcl.AddByPriority(priority.Normal-1, func() error {
		fmt.Print("run after normal. ")

		return nil
	})

	pcl.Close()
	// Output: run before first. first stop. normal stop. long normal stop. run after normal. last stop.
}

func ExampleCloser_Wait_cancelContext() {
	pcl := priority.New()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Microsecond)
	defer cancel()

	_ = pcl.Add(func() error {
		fmt.Print("do some close with cancel context")

		return nil
	})

	pcl.Wait(ctx)
	// Output: do some close with cancel context
}

func ExampleCloser_SetErrHandler() {
	pcl := priority.New()
	pcl.SetErrHandler(func(err error) {
		fmt.Print("logged err:", err.Error())
	})

	_ = pcl.Add(func() error {
		return test.ErrClose
	})

	pcl.Close()
	// Output: logged err:some error
}

func ExampleCloser_Add() {
	pcl := priority.New()
	pcl.Close()

	err := pcl.Add(func() error {
		fmt.Print("must not run")

		return nil
	})
	fmt.Print(err)
	// Output: closer is closed
}
