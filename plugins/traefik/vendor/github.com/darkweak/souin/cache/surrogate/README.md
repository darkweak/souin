# Surrogate-Keys system

## What is a Surrogate-Key?
A Surrogate-Key allows you to tag a pool of resources and purge them targeting a specific key.  
A resource can be tagged by none or multiple surrogate keys and vice-versa.

## How to deal with it?

### In a HTTP request
The client send to the server an HTTP request which one contains the header Surrogate-Keys containing one or multiple 
keys.  
Either the user send the previous request with the PURGE method, it must delete the cache for the associated keys and 
returns a `204 no-content` response.  
Either the user send the previous request with the GET method, it must store the resource URL to the provided keys in 
the header. If one of the keys doesn't already exist, the storage system must add the key in the storage.

### In a HTTP response
The client receives an HTTP response which one contains the header Surrogate-Keys. It mentions that the resource is associate
to the given keys.  
With this information, the client can store the keys, and invalidate manually in the future 
[as mentioned in the previous section](#in-a-http-request). 

## Surrogate-Keys specification for the cache

You can refer to the [specification file](specification.md). The goal is to propose the specification as an RFC and make
the `Surrogate-Keys`.
