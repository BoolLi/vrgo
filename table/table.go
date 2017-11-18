// table provides the client table interface.
package table

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

// ClientTable represents a client table database.
type ClientTable struct {
	lastRecords map[string]interface{}
	clientTable *cache.Cache
}

// New creates a new client table.
func New(defaultExpiration, cleanupInterval time.Duration) *ClientTable {
	return &ClientTable{
		lastRecords: make(map[string]interface{}),
		clientTable: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Set sets a value for a key.
func (t *ClientTable) Set(k string, x interface{}) {
	res, ok := t.Get(k)
	if ok {
		t.lastRecords[k] = res
	}

	t.clientTable.Set(k, x, cache.NoExpiration)
}

// Undo reverts to the last record for a key.
func (t *ClientTable) Undo(k string) {
	t.clientTable.Set(k, t.lastRecords[k], cache.NoExpiration)
}

// Get returns the record for a key.
func (t *ClientTable) Get(k string) (interface{}, bool) {
	return t.clientTable.Get(k)
}
