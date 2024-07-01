package core

import "time"

type Configuration struct {
	Provider CacheProvider `json:"provider"`
	Stale    time.Duration `json:"stale"`
}
