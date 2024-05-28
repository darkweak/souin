package tlsutil

import (
	"sync"
)

type credentialsCacheElement struct {
	sni     string
	renewer *Renewer
}

type credentialsCache struct {
	CacheStore *sync.Map
}

func newCredentialsCache() *credentialsCache {
	return &credentialsCache{
		CacheStore: new(sync.Map),
	}
}

func (c *credentialsCache) Load(domain string) (*credentialsCacheElement, bool) {
	v, ok := c.CacheStore.Load(domain)
	if !ok {
		return nil, false
	}
	e, ok := v.(*credentialsCacheElement)
	return e, ok
}

func (c *credentialsCache) Store(domain string, v *credentialsCacheElement) {
	c.CacheStore.Store(domain, v)
}

func (c *credentialsCache) Delete(domain string) {
	c.CacheStore.Delete(domain)
}

func (c *credentialsCache) Range(fn func(domain string, v *credentialsCacheElement) bool) {
	c.CacheStore.Range(func(k, v interface{}) bool {
		if domain, ok := k.(string); ok {
			if e, ok := v.(*credentialsCacheElement); ok {
				return fn(domain, e)
			}
		}
		return true
	})
}
