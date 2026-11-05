# closer

![Build Status](https://github.com/go-4devs/closer/actions/workflows/goaction.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-4devs/closer)](https://goreportcard.com/report/github.com/go-4devs/closer)
[![GoDoc](https://godoc.org/github.com/go-4devs/closer?status.svg)](http://godoc.org/github.com/go-4devs/closer)

`closer` registers shutdown funcs and runs them once when the process should stop. A higher priority runs first, from 255 down to 0. `Close` returns the funcs' errors. `Add` after `Close` has started returns `ErrClosed`. A panic becomes `ErrPanic`.

`Wait` blocks until the context is canceled or a signal arrives, then runs `Close`. Canceling the context does not skip the funcs. A non-positive timeout means no deadline.

The root package is one process-wide closer. `priority` is the same API on your own value, with funcs ordered by priority. `async` runs every func at the same time. `routine` tracks goroutines and drops new ones while `Wait` is in progress.

```go
closer.Add(func() error {
    return db.Close()
})
closer.Wait(ctx, syscall.SIGTERM)
```
