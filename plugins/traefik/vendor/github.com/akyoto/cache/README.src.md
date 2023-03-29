# {name}

{go:header}

Cache arbitrary data with an expiration time.

## Features

* 0 dependencies
* About 100 lines of code
* 100% test coverage

## Usage

```go
// New cache
c := cache.New(5 * time.Minute)

// Put something into the cache
c.Set("a", "b", 1 * time.Minute)

// Read from the cache
obj, found := c.Get("a")

// Convert the type
fmt.Println(obj.(string))
```

## Benchmarks

```text
BenchmarkGet-12         300000000                3.88 ns/op            0 B/op          0 allocs/op
BenchmarkSet-12         10000000               183 ns/op              48 B/op          2 allocs/op
BenchmarkNew-12         10000000               112 ns/op             352 B/op          5 allocs/op
```

{go:footer}
