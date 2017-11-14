package server

import (
	"log"
	"strconv"

	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/rpc"
)

// VrgoRPC defines the user RPCs exported by server.
type VrgoRPC int

func (v *VrgoRPC) Execute(req *rpc.Request, resp *rpc.Response) error {
	k := strconv.Itoa(req.ClientId)
	res, ok := clientTable.Get(k)

	// If the client request is already executed before, resend the response.
	if ok && req.RequestNum <= res.(rpc.Response).RequestNum {
		log.Printf("request %v is already executed; returning previous result %v directly.\n", req, res)
		*resp = res.(rpc.Response)
		return nil
	}

	// First time receiving from this client.
	if !ok {
		log.Printf("first time receiving request %v from client %v\n", req.RequestNum, req.ClientId)
	}

	ch := primary.ProcessIncomingReq(req)
	select {
	case _ = <-ch:
		*resp = rpc.Response{}
	}

	return nil
}
