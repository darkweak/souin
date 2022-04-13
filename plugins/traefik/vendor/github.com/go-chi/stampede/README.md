# Stampede

![](https://github.com/go-chi/stampede/workflows/build/badge.svg?branch=master)

Prevents cache stampede https://en.wikipedia.org/wiki/Cache_stampede by only running a
single data fetch operation per expired / missing key regardless of number of requests to that key.


## Example 1: HTTP Middleware

```go
import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/stampede"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("index"))
	})

	cached := stampede.Handler(512, 1 * time.Second)

	r.With(cached).Get("/cached", func(w http.ResponseWriter, r *http.Request) {
		// processing..
		time.Sleep(1 * time.Second)

		w.WriteHeader(200)
		w.Write([]byte("...hi"))
	})

	http.ListenAndServe(":3333", r)
}
```


## Example 2: Raw

```go
import (
	"net/http"

	"github.com/go-chi/stampede"
)

var (
	reqCache = stampede.NewCache(512, 5*time.Second, 10*time.Second)
)

func handler(w http.ResponseWriter, r *http.Request) {	
	data, err := reqCache.Get(r.URL.Path, fetchData)
	if err != nil {	
		w.WriteHeader(503)
		return	
	}

	w.Write(data.([]byte))
}

func fetchData(ctx context.Context) (interface{}, error) {
	// fetch from remote source.. or compute/render..
	data := []byte("some response data")

	return data, nil	
}
```

## Notes

* Requests passed through the stampede handler will be batched into a single request
when there are parallel requests for the same endpoint/resource. This is also known
as request coalescing.
* Parallel requests for the same endpoint / resource, will be just a single handler call
and the remaining requests will receive the response of the first request handler.
* The response payload for the endpoint / resource will then be cached for up to `ttl`
time duration for subequence requests, which offers further caching. You may also
use a `ttl` value of 0 if you want the response to be as fresh as possible, and still
prevent a stampede scenario on your handler.
* *Security note:* response headers will be the same for all requests, so make sure
to not include anything sensitive or user specific. In the case you require user-specific
stampede handlers, make sure you pass a custom `keyFunc` to the `stampede.Handler` and
split the cache by an account's id.
See [example](_example/with-key.go) for a variety of examples.


## LICENSE

MIT
