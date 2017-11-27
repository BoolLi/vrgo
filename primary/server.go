package primary

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/BoolLi/vrgo/flags"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

// RegisterVrgo registers a Vrgo RPC receiver.
func RegisterVrgo(rcvr vrrpc.VrgoService) error {
	return rpc.Register(rcvr)
}

// RegisterView registers a View RPC receiver.
func RegisterView(rcvr vrrpc.ViewService) error {
	return rpc.Register(rcvr)
}

// Serve starts an HTTP server to handle RPC requests.
func ServeHTTP() {
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", *flags.Port))
	if err != nil {
		log.Fatalf("failed to listen on port %v: %v", *flags.Port, err)
	}
	http.Serve(l, nil)
}
