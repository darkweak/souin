package httpcache

import (
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
)

func (s *SouinCaddyMiddleware) parseStorages(ctx caddy.Context) {
	if s.Configuration.DefaultCache.Badger.Found {
		e := dispatchStorage(ctx, "badger", s.Configuration.DefaultCache.Badger, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Badger init, did you include the Badger storage (--with github.com/darkweak/storages/badger/caddy)? %v", e)
		} else {
			badger := s.Configuration.DefaultCache.Badger
			dir := ""
			vdir := ""
			if c := badger.Configuration; c != nil {
				p, ok := c.(map[string]interface{})
				if ok {
					if d, ok := p["Dir"]; ok {
						dir = fmt.Sprint(d)
						vdir = fmt.Sprint(d)
					}
					if d, ok := p["ValueDir"]; ok {
						vdir = fmt.Sprint(d)
					}
				}
			}
			s.Configuration.DefaultCache.Badger.Uuid = fmt.Sprintf(
				"BADGER-%s-%s-%s",
				dir,
				vdir,
				s.Configuration.DefaultCache.GetStale(),
			)
		}
	}
	if s.Configuration.DefaultCache.Etcd.Found {
		e := dispatchStorage(ctx, "etcd", s.Configuration.DefaultCache.Etcd, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Etcd init, did you include the Etcd storage (--with github.com/darkweak/storages/etcd/caddy)? %v", e)
		} else {
			etcd := s.Configuration.DefaultCache.Etcd
			endpoints := etcd.URL
			username := ""
			password := ""
			if c := etcd.Configuration; c != nil {
				p, ok := c.(map[string]interface{})
				if ok {
					if d, ok := p["Endpoints"]; ok {
						endpoints = fmt.Sprint(d)
					}
					if d, ok := p["Username"]; ok {
						username = fmt.Sprint(d)
					}
					if d, ok := p["Password"]; ok {
						password = fmt.Sprint(d)
					}
				}
			}
			s.Configuration.DefaultCache.Etcd.Uuid = fmt.Sprintf(
				"ETCD-%s-%s-%s-%s",
				endpoints,
				username,
				password,
				s.Configuration.DefaultCache.GetStale(),
			)
		}
	}
	if s.Configuration.DefaultCache.Nats.Found {
		e := dispatchStorage(ctx, "nats", s.Configuration.DefaultCache.Nats, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Nats init, did you include the Nats storage (--with github.com/darkweak/storages/nats/caddy)? %v", e)
		} else {
			s.Configuration.DefaultCache.Nuts.Uuid = fmt.Sprintf("NATS-%s-%s", s.Configuration.DefaultCache.Nats.URL, s.Configuration.DefaultCache.GetStale())
		}
	}
	if s.Configuration.DefaultCache.Nuts.Found {
		e := dispatchStorage(ctx, "nuts", s.Configuration.DefaultCache.Nuts, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Nuts init, did you include the Nuts storage (--with github.com/darkweak/storages/nuts/caddy)? %v", e)
		} else {
			nuts := s.Configuration.DefaultCache.Nuts
			dir := "/tmp/souin-nuts"
			if c := nuts.Configuration; c != nil {
				p, ok := c.(map[string]interface{})
				if ok {
					if d, ok := p["Dir"]; ok {
						dir = fmt.Sprint(d)
					}
				}
			} else if nuts.Path != "" {
				dir = nuts.Path
			}
			s.Configuration.DefaultCache.Nuts.Uuid = fmt.Sprintf("NUTS-%s-%s", dir, s.Configuration.DefaultCache.GetStale())
		}
	}
	if s.Configuration.DefaultCache.Olric.Found {
		e := dispatchStorage(ctx, "olric", s.Configuration.DefaultCache.Olric, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Olric init, did you include the Olric storage (--with github.com/darkweak/storages/olric/caddy)? %v", e)
		} else {
			s.Configuration.DefaultCache.Nuts.Uuid = fmt.Sprintf("OLRIC-%s-%s", s.Configuration.DefaultCache.Olric.URL, s.Configuration.DefaultCache.GetStale())
		}
	}
	if s.Configuration.DefaultCache.Otter.Found {
		e := dispatchStorage(ctx, "otter", s.Configuration.DefaultCache.Otter, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Otter init, did you include the Otter storage (--with github.com/darkweak/storages/otter/caddy)? %v", e)
		} else {
			s.Configuration.DefaultCache.Otter.Uuid = fmt.Sprintf("OTTER-%s", s.Configuration.DefaultCache.GetStale())
		}
	}
	if s.Configuration.DefaultCache.Redis.Found {
		e := dispatchStorage(ctx, "redis", s.Configuration.DefaultCache.Redis, s.Configuration.DefaultCache.GetStale())
		if e != nil {
			s.logger.Errorf("Error during Redis init, did you include the Redis storage (--with github.com/darkweak/storages/redis/caddy or github.com/darkweak/storages/go-redis/caddy)? %v", e)
		} else {
			redis := s.Configuration.DefaultCache.Redis
			address := redis.URL
			username := ""
			dbname := "0"
			cname := ""
			if c := redis.Configuration; c != nil {
				p, ok := c.(map[string]interface{})
				if ok {
					// shared between go-redis and rueidis
					if d, ok := p["Username"]; ok {
						username = fmt.Sprint(d)
					}
					if d, ok := p["ClientName"]; ok {
						cname = fmt.Sprint(d)
					}

					// rueidis
					if d, ok := p["InitAddress"]; ok {
						elements := make([]string, 0)

						for _, elt := range d.([]interface{}) {
							elements = append(elements, elt.(string))
						}

						address = strings.Join(elements, ",")
					}
					if d, ok := p["SelectDB"]; ok {
						dbname = fmt.Sprint(d)
					}

					// go-redis
					if d, ok := p["Addrs"]; ok {
						elements := make([]string, 0)

						for _, elt := range d.([]interface{}) {
							elements = append(elements, elt.(string))
						}

						address = strings.Join(elements, ",")
					}
					if d, ok := p["DB"]; ok {
						dbname = fmt.Sprint(d)
					}
				}
			}
			s.Configuration.DefaultCache.Redis.Uuid = fmt.Sprintf(
				"REDIS-%s-%s-%s-%s-%s",
				address,
				username,
				dbname,
				cname,
				s.Configuration.DefaultCache.GetStale(),
			)
		}
	}
}
