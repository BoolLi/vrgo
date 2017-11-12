// server defines an HTTP RPC server.
package server

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

var port = flag.Int("port", 1234, "server port")

// Register registers a RPC receiver.
func Register(rcvr interface{}) error {
	return rpc.Register(rcvr)
}

// Serve starts an HTTP server to handle RPC requests.
func Serve() {
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", *port))
	if err != nil {
		log.Fatalf("failed to listen on port %v: %v", *port, err)
	}
	http.Serve(l, nil)
}
