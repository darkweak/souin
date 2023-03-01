%%%
title = "Surrogate HTTP headers"
area = "Internet"
workgroup = "Network Working Group"
submissiontype = "IETF"

[seriesInfo]
name = "Internet-Draft"
value = "draft-darkweak-Surrogate-headers-01"
stream = "IETF"
status = "standard"

[[author]]
initials="S."
surname="Combraque"
fullname="Sylvain Combraque"
abbrev = ""
organization = ""
[author.address]
email = "darkweak@protonmail.com"
[author.address.postal]
country = "France"
%%%

.# Abstract

The Surrogate headers allow to manage the cache invalidation by the surrogates keys.
The Surrogate-Key HTTP header is useful to get the information about a cached resource, and provide a way to invalidate
properly a pool of stored resources.
The Surrogate-Control allow the management directive of the `Surrogate-Key`.

{mainmatter}

# Terminology

The keywords **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD
NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL**, when they appear in this document, are to be
interpreted as described in [@!RFC2119].

*  Surrogate-Key: Identifier of a resources pool.

*  Client: The originating endpoint of a request; the destination endpoint of
   a response as describe in the [@!RFC1856].

*  Server: The destination endpoint of a request; the originating endpoint of
   a response as describe in the [@!RFC1856].

*  Service: Application that serves content.

*  Cache: The system that will store the resources, handle then incoming requests and try to serve the content matching
   a key if it already has in his storage.

This specification defines the following key parameter:

stored       = sf-boolean
bypass       = sf-boolean
detail       = sf-token / sf-string

## The stored directive
"stored", when true, indicates that the key has been stored and can be used and will be treated while passed in a PURGE 
request.  
"stored" and "bypass" are exclusive; only one of them should appear on each list member.

## The bypass directive
"bypass", when true, indicates that the key has not been stored because it was not satisfied by the validation rules.

## The detail directive
"detail", allows implementations to convey additional information of why the server didn't store the key; for example, 
implementation-specific states.

For example:

Surrogate-Key: abc;bypass;details=PRESENT, def;bypass;details=NOT_ALLOWED

Examples:

Surrogate-Key: abc;stored, def;bypass

Surrogate-Key: abc;bypass;details=ANOTHER


# Invalidation

The invalidation mechanism is trigger from a `PURGE` request to the API endpoint.  
The request **MUST** set the header `Surrogate-Key` with at least one value to invalidate the provided ones, and the 
server **MUST** invalidate the resources associated to the targeted keys if it exists.

The invalidation process **SHOULD** return either:
* a `202 Accepted` HTTP code if the server doesn't invalidate synchronously the keys
  ~~~ http
  PURGE /surrogate-api-endpoint HTTP/1.1
  Host: example.com
  Surrogate-Key: my-key, my-second-key
  
  HTTP/1.1 202 Accepted
  ~~~
* a `204 No Content` HTTP code if the server invalidate synchronously the keys
  ~~~ http
  PURGE /surrogate-api-endpoint HTTP/1.1
  Host: example.com
  Surrogate-Key: my-key, my-second-key
  
  HTTP/1.1 204 No Content
  ~~~
In the both cases, the server **MUST** invalidate the cache for the associated resource URLs to at least one of the 
`Surrogate-Key` item.

# Set a resource to one or many Surrogate-Key

The resource setting can be done from the application target by the server.
The application **MUST** return a response with the header `Surrogate-Key` which contains the keys to add the resource 
URL to. The server will store the resource URL in each provided surrogate key.
  ~~~ http
  GET /any/path HTTP/1.1
  Host: example.com
  HTTP/1.1 200 OK
  Surrogate-Key: my-key, my-second-key

  My awesome content
  ~~~
The server **MUST** store the URL resource inside the `my-key` and `my-second-key` surrogate-keys.  
For each key not already stored for this resource, the server **SHOULD** return the directive `stored`.
  ~~~ http
  GET /any/path HTTP/1.1
  Host: example.com
  HTTP/1.1 200 OK
  Surrogate-Key: my-key;stored, my-second-key;stored

  My awesome content
  ~~~

# Surrogate-Control

The `Surrogate-Key` header **CAN** be driven by the `Surrogate-Control` header to define if the server **MUST** store 
the URL resource or not. 

The server **MUST** handle the `no-store` directive by key in the `Surrogate-Control` header. The
application will send to the server the headers `Surrogate-Key` and `Surrogate-Control` to drive – per-key – the cache 
directive. The application sends an HTTP response with `Surrogate-Key: my-key, my-second-key` and 
`Surrogate-Control: no-store;my-key` headers to the server.
  ~~~ http
  GET /any/path HTTP/1.1
  Host: example.com
  HTTP/1.1 200 OK
  Surrogate-Key: my-key, my-second-key
  Surrogate-Control: no-store;my-key

  My awesome content
  ~~~
The server **MUST** store the URL resource inside the `my-second-key` surrogate-key list and **MUST NOT** store the URL 
resource inside the `my-second-key` surrogate-key list due to the `my-key, no-store` directive presence.

The application sends an HTTP response with `Surrogate-Key: my-key, my-second-key` and `Surrogate-Control: no-store` to
the server
  ~~~ http
  GET /any/path HTTP/1.1
  Host: example.com
  HTTP/1.1 200 OK
  Surrogate-Key: my-key, my-second-key
  Surrogate-Control: no-store

  My awesome content
  ~~~
The server **MUST NOT** store the URL resource inside any surrogate-key list due to the `no-store` global directive presence.

{backmatter}
