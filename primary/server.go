package primary

import (
	"net/rpc"

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
