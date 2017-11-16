// server defines an HTTP RPC server.
package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/BoolLi/vrgo/flags"

	vrrpc "github.com/BoolLi/vrgo/rpc"
	cache "github.com/patrickmn/go-cache"
)

var clientTable *cache.Cache

// Init initializes the server with necessary dependencies.
func Init(t *cache.Cache) error {
	clientTable = t
	return nil
}

// Register registers a RPC receiver.
func Register(rcvr vrrpc.VrgoService) error {
	return rpc.Register(rcvr)
}

// Serve starts an HTTP server to handle RPC requests.
func Serve() {
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", *flags.Port))
	if err != nil {
		log.Fatalf("failed to listen on port %v: %v", *flags.Port, err)
	}
	http.Serve(l, nil)
}
