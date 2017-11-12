package table

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

func New(defaultExpiration, cleanupInterval time.Duration) *cache.Cache {
	return cache.New(defaultExpiration, cleanupInterval)
}
