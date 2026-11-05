# closer

![Build Status](https://github.com/go-4devs/closer/actions/workflows/goaction.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-4devs/closer)](https://goreportcard.com/report/github.com/go-4devs/closer)
[![GoDoc](https://godoc.org/github.com/go-4devs/closer?status.svg)](http://godoc.org/github.com/go-4devs/closer)

`closer` registers shutdown funcs and runs them once when the process should stop. A higher priority runs first, from 255 down to 0. `Close` returns the funcs' errors. `Add` after `Close` has started returns `ErrClosed`. A panic becomes `ErrPanic`.

`Wait` blocks until the context is canceled or a signal arrives, then runs `Close`. Canceling the context does not skip the funcs. A non-positive timeout means no deadline.

The root package is one process-wide closer.

```go
closer.Add(func() error {
    return db.Close()
})
closer.Wait(ctx, syscall.SIGTERM)
```

`AddFirst` runs near the start. `AddLast` runs near the end. `AddByPriority` takes any value from 255 down to 0. `First` is 250, `Normal` is 100, `Last` is 5.

```go
_ = closer.AddFirst(func() error {
    return srv.Shutdown(ctx)
})
_ = closer.AddLast(func() error {
    return db.Close()
})
```

`SetTimeout` is the deadline for one `Close`. A func still running after it is left out. Errors already returned are kept. `SetErrHandler` is called for each of them. The default handler logs the error.

```go
closer.SetTimeout(5 * time.Second)
closer.SetErrHandler(func(err error) {
    log.Print(err)
})
```

`Shutdown` turns a func that takes a context into a close func. A positive timeout cancels that context. Zero means no deadline.

```go
_ = closer.Add(closer.Shutdown(func(ctx context.Context) error {
    return srv.Shutdown(ctx)
}, 5*time.Second))
```

`priority` is the same API on your own value.

```go
stop := priority.New(priority.WithTimeout(5 * time.Second))
_ = stop.AddFirst(func() error { return srv.Close() })
err := stop.Close()
```

`async` runs every func at the same time. Timeout and error handling match `priority`.

```go
stop := async.New()
_ = stop.Add(
    func() error { return db.Close() },
    func() error { return cache.Close() },
)
err := stop.Close()
```

`routine` tracks goroutines. `Wait` blocks until they return. A new `Go` during `Wait` is ignored. `Close` waits and returns nil, so it can be registered on a closer.

```go
routine.Go(func() {
    worker()
})
_ = closer.Add(routine.Close)
```

`inflight` tracks calls already inside `Do`. `Wait` returns when they finish. If that context ends first, the calls are canceled and `Wait` returns without waiting for them.

```go
group := inflight.New()

err := group.Do(r.Context(), func(ctx context.Context) error {
    return handle(ctx)
})

err = group.Wait(shutdown)
```
