// globals defines the global variables shared between primary and backup.
package globals

import (
	"context"
	"sync"

	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/table"
)

// MutexInt is a thread-safe int.
type MutexInt struct {
	sync.Mutex
	V int
}

// Locked locks the int.
func (m *MutexInt) Locked(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

// MutexString is a thread-safe string.
type MutexString struct {
	sync.Mutex
	V string
}

// Locked locks the string.
func (m *MutexString) Locked(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

// MutexBool is a thread-safe bool.
type MutexBool struct {
	sync.Mutex
	V bool
}

// Locked locks the bool.
func (m *MutexBool) Locked(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

var (
	// The Operation request ID.
	OpNum int

	// The current view number.
	ViewNum int

	// The current commit number.
	CommitNum int

	// The operation log.
	OpLog *oplog.OpRequestLog

	// The client table.
	ClientTable *table.ClientTable

	// The global cancellable context.
	CtxCancel context.Context

	// AllPorts is a temporary hardcoded map from id to port.
	// TODO: Generate this dynamically based on a config file.
	AllPorts = map[int]int{0: 1234, 1: 9000, 2: 9001, 3: 9002, 4: 9003}
)
