# cache

[![Godoc][godoc-image]][godoc-url]
[![Report][report-image]][report-url]
[![Tests][tests-image]][tests-url]
[![Coverage][coverage-image]][coverage-url]
[![Sponsor][sponsor-image]][sponsor-url]

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

## Style

Please take a look at the [style guidelines](https://github.com/akyoto/quality/blob/master/STYLE.md) if you'd like to make a pull request.

## Sponsors

| [![Cedric Fung](https://avatars3.githubusercontent.com/u/2269238?s=70&v=4)](https://github.com/cedricfung) | [![Scott Rayapoullé](https://avatars3.githubusercontent.com/u/11772084?s=70&v=4)](https://github.com/soulcramer) | [![Eduard Urbach](https://avatars3.githubusercontent.com/u/438936?s=70&v=4)](https://eduardurbach.com) |
| --- | --- | --- |
| [Cedric Fung](https://github.com/cedricfung) | [Scott Rayapoullé](https://github.com/soulcramer) | [Eduard Urbach](https://eduardurbach.com) |

Want to see [your own name here?](https://github.com/users/akyoto/sponsorship)

[godoc-image]: https://godoc.org/github.com/akyoto/cache?status.svg
[godoc-url]: https://godoc.org/github.com/akyoto/cache
[report-image]: https://goreportcard.com/badge/github.com/akyoto/cache
[report-url]: https://goreportcard.com/report/github.com/akyoto/cache
[tests-image]: https://cloud.drone.io/api/badges/akyoto/cache/status.svg
[tests-url]: https://cloud.drone.io/akyoto/cache
[coverage-image]: https://codecov.io/gh/akyoto/cache/graph/badge.svg
[coverage-url]: https://codecov.io/gh/akyoto/cache
[sponsor-image]: https://img.shields.io/badge/github-donate-green.svg
[sponsor-url]: https://github.com/users/akyoto/sponsorship
