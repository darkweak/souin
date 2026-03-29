# HTTP cache middleware

## What is the middleware?
The middleware is the HTTP cache entrypoint of Souin. It sits in front of the upstream application, computes cache keys, serves cached responses when possible, revalidates stale content, stores fresh upstream responses, and exposes the cache status to the client.

## How to deal with it?

### In a regular HTTP request
The client sends a cacheable request to the server. The middleware computes the cache key, checks the configured storages, and either:
* serves a fresh cached response
* serves a stale response according to the cache directives
* forwards the request upstream and stores the response when it is cacheable

The middleware also sets the `Cache-Status` response header so the client can understand if the response was a cache hit, a stale hit, a revalidation, or a miss.

### In a soft purge request
The client sends a `PURGE` request to the API endpoint, either for surrogate keys or for a direct cache key pattern, and sets:
* `Souin-Purge-Mode: soft`

The middleware-related behavior is different from a hard purge:
* the cached object is kept in storage
* the associated mapping is marked stale
* a soft purge marker is attached to the stored cache entry

The next matching request is then served as stale immediately. If the cached response has validators or `stale-while-revalidate`, the middleware also starts a detached background refresh.

### During the background refresh
When a soft-purged response is served, the middleware only refreshes it in background when the cached response can be revalidated or refreshed safely:
* if the cached response has `ETag` or `Last-Modified`, it prefers a conditional revalidation
* if the cached response exposes `stale-while-revalidate`, it can do a background fetch
* if neither validators nor `stale-while-revalidate` are present, the stale response is still served but no refresh is started

Concurrent refreshes are deduplicated per stored key so only one background refresh runs for the same soft-purged object at a time.

### In the client response
When the response comes from a soft-purged entry, the middleware returns it as stale and adds a dedicated `Cache-Status` detail such as:
* `SOFT-PURGE-REVALIDATE`
* `SOFT-PURGE-SWR`
* `SOFT-PURGE-SIE`
* `SOFT-PURGED`

Once the background refresh succeeds, the middleware stores the refreshed response and clears the soft purge marker so future requests behave like normal cache hits again.

## Hard purge vs soft purge
A hard purge removes the cached response and the related mapping immediately.
A soft purge keeps the cached response available for stale serving and marks the mapping stale. The next request serves that stale response immediately, and only triggers a background refresh when validators or `stale-while-revalidate` allow it.
