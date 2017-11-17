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

// Register registers a RPC receiver.
func RegisterRPC(rcvr vrrpc.VrgoService) error {
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
