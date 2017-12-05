// globals defines the global variables shared between primary and backup.
package globals

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"sync"

	"github.com/BoolLi/vrgo/flags"
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

func Log(f, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%v, %20v] %v", *flags.Id, f, msg)
}

var (
	// The port of the replica.
	Port int

	// The Operation request ID.
	OpNum int

	// The current view number.
	ViewNum int

	// The current commit number.
	CommitNum int

	// The mode of the replica. Only monitor is supposed to change this.
	Mode string

	// The operation log.
	OpLog *oplog.OpRequestLog

	// The client table.
	ClientTable *table.ClientTable

	// The global cancellable context.
	CtxCancel context.Context

	// AllPorts is a map from id to port.
	AllPorts = map[int]int{}

	// clients is a map from hostname to *rpc.Client.
	// This way each node only creates one outgoing client to another node,
	// and more requests to the same node will reuse the same client.
	clients = map[string]*rpc.Client{}
)

func init() {
	log.Printf("entering globals.init; id: %v", *flags.Id)
	csvFile, err := os.Open(*flags.ConfigPath)
	if err != nil {
		log.Fatalf("failed to open csv file: %v", err)
	}
	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read line from config %v: %v", csvFile, err)
		}

		id, err := strconv.Atoi(line[1])
		if err != nil {
			log.Fatalf("failed to convert id to int: %v", err)
		}
		port, err := strconv.Atoi(line[2])
		if err != nil {
			log.Fatalf("failed to convert port to int: %v", err)
		}
		log.Printf("globals.Init: id: %v, port: %v", id, port)
		AllPorts[id] = port

		// Initialize own mode and port.
		if id == *flags.Id {
			Mode = line[0]
			Port = port
			Log("globals.init", "initial mode: %v; port: %v", Mode, Port)
		}
	}
}

// AllOtherPorts returns all the other replica ports except for that of the current node.
func AllOtherPorts() []int {
	var ps []int
	for _, p := range AllPorts {
		if p != Port {
			ps = append(ps, p)
		}
	}
	return ps
}

// GetOrCreateClient returns a cached rpc.Client or creates a new rpc.Client.
func GetOrCreateClient(hostname string) (*rpc.Client, error) {
	if client, ok := clients[hostname]; ok == true {
		return client, nil
	}
	client, err := rpc.DialHTTP("tcp", hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %v: %v", hostname, err)
	}
	clients[hostname] = client
	return client, nil
}
