# fifo

[![godoc](https://pkg.go.dev/badge/gopkg.in/fifo.v0.svg)](https://pkg.go.dev/gopkg.in/fifo.v0)
[![codecov](https://codecov.io/gh/go-fifo/fifo/graph/badge.svg?token=463HXD6XJY)](https://codecov.io/gh/go-fifo/fifo)
[![codacy](https://app.codacy.com/project/badge/Grade/8cda947ddc8443dfa59effbf8b337dd1)](https://app.codacy.com/gh/go-fifo/fifo/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
[![codeclimate](https://api.codeclimate.com/v1/badges/b71ad41ce072787fb1fe/maintainability)](https://codeclimate.com/github/go-fifo/fifo/maintainability)
[![goreportcard](https://goreportcard.com/badge/gopkg.in/fifo.v0)](https://goreportcard.com/report/gopkg.in/fifo.v0)

The missing queue package for Go.

## Overview

The `fifo` package provides a thread-safe FIFO (First In, First Out) queue with resizable capacity. It is designed to be used in concurrent environments and supports blocking and non-blocking operations.

## Features

- Thread-safe operations
- Resizable capacity
- Blocking and non-blocking enqueue and dequeue operations
- Graceful handling of closed queues

## Installation

To install the package, run:

```bash
go get gopkg.in/fifo.v0
```

## Usage

### Creating a Queue

Create a new queue with a specified initial capacity:

```go
import "gopkg.in/fifo.v0"

q := fifo.New[int](3)
```

### Enqueue Operations

Non-blocking enqueue:

```go
err := q.TryEnqueue(1)
if err != nil {
    // Handle error (e.g., queue is full or closed)
}
```

Blocking enqueue:

```go
err := q.Enqueue(2)
if err != nil {
    // Handle error (e.g., queue is closed)
}
```

### Dequeue Operations

Non-blocking dequeue:

```go
item, err := q.TryDequeue()
if err != nil {
    // Handle error (e.g., queue is empty or closed)
}
```

Blocking dequeue:

```go
item, err := q.Dequeue()
if err != nil {
    // Handle error (e.g., queue is closed)
}
```

### Resizing the Queue

Resize the queue to a new capacity:

```go
err := q.Resize(5)
if err != nil {
    // Handle error (e.g., invalid capacity or queue is closed)
}
```

### Closing the Queue

Close the queue to prevent further enqueues:

```go
err := q.Close()
if err != nil {
    // Handle error (e.g., queue already closed)
}
```

### Checking Length and Capacity

Get the current length and capacity of the queue:

```go
length := q.Len()
capacity := q.Cap()
```

## Example

Here's a complete example demonstrating the usage of the `fifo` package:

```go
package main

import (
    "fmt"
    "sync"
    "time"
    "gopkg.in/fifo.v0"
)

func main() {
    q := fifo.New[string](3)

    // Enqueue items
    q.TryEnqueue("A")
    q.TryEnqueue("B")
    q.TryEnqueue("C")

    fmt.Println("Length after enqueueing 3 items:", q.Len())
    fmt.Println("Capacity after enqueueing 3 items:", q.Cap())

    // Try to enqueue when full
    err := q.TryEnqueue("D")
    fmt.Println("TryEnqueue when full:", err)

    // Dequeue an item
    item, _ := q.TryDequeue()
    fmt.Println("Dequeued item:", item)

    // Enqueue another item
    q.TryEnqueue("D")

    // Resize the queue
    err = q.Resize(5)
    fmt.Println("Resize result:", err)
    fmt.Println("Capacity after resize:", q.Cap())

    // Enqueue more items
    q.TryEnqueue("E")
    q.TryEnqueue("F")

    // Dequeue all items
    for q.Len() > 0 {
        item, _ := q.TryDequeue()
        fmt.Println("Dequeued item:", item)
    }

    // Concurrent operations
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        for i := 0; i < 5; i++ {
            q.Enqueue(fmt.Sprintf("G%d", i))
            time.Sleep(10 * time.Millisecond)
        }
    }()

    go func() {
        defer wg.Done()
        for i := 0; i < 5; i++ {
            item, _ := q.Dequeue()
            fmt.Println("Concurrent dequeued item:", item)
            time.Sleep(15 * time.Millisecond)
        }
    }()

    wg.Wait()

    // Close the queue
    q.Close()

    // Try operations after closing
    err = q.TryEnqueue("H")
    fmt.Println("TryEnqueue after close:", err)

    _, err = q.TryDequeue()
    fmt.Println("TryDequeue after close:", err)

    err = q.Resize(10)
    fmt.Println("Resize after close:", err)
}
```

## License

This project is licensed under the MIT License.
