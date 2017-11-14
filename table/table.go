// table provides the client table interface.
package table

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

// New creates a new client table.
func New(defaultExpiration, cleanupInterval time.Duration) *cache.Cache {
	return cache.New(defaultExpiration, cleanupInterval)
}
