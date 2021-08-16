# ConnPool [![Go Reference](https://pkg.go.dev/badge/github.com/buraksezer/connpool.svg)](https://pkg.go.dev/github.com/buraksezer/connpool) ![Tests](https://github.com/buraksezer/connpool/actions/workflows/ci.yaml/badge.svg)


ConnPool is a thread safe connection pool for net.Conn interface. It can be used to
manage and reuse connections.

This package is a fork of [fatih/pool](https://github.com/fatih/pool). ConnPool is a vital 
part of [buraksezer/olric](https://github.com/buraksezer/olric). So I need to maintain it myself.

Fixed bugs:

* https://github.com/fatih/pool/issues/26
* https://github.com/fatih/pool/issues/27

## Install and Usage

Install the package with:

```bash
go get github.com/buraksezer/connpool
```

## Example

```go
// create a factory() to be used with channel based pool
factory := func() (net.Conn, error) { return net.Dial("tcp", "127.0.0.1:4000") }

// create a new channel based pool with an initial capacity of 5 and maximum
// capacity of 30. The factory will create 5 initial connections and put it
// into the pool.
p, err := connpool.NewChannelPool(5, 30, factory)

// now you can get a connection from the pool, if there is no connection
// available it will create a new one via the factory function.
conn, err := p.Get(context.Background())

// do something with conn and put it back to the pool by closing the connection
// (this doesn't close the underlying connection instead it's putting it back
// to the pool).
conn.Close()

// close the underlying connection instead of returning it to pool
// it is useful when acceptor has already closed connection and conn.Write() returns error
if pc, ok := conn.(*connpool.PoolConn); ok {
  pc.MarkUnusable()
  pc.Close()
}

// close pool any time you want, this closes all the connections inside a pool
p.Close()

// currently available connections in the pool
current := p.Len()
```


## Credits

 * [Fatih Arslan](https://github.com/fatih)
 * [sougou](https://github.com/sougou)

## License

The MIT License (MIT) - see LICENSE for more details
