package primary

import (
	"log"
	"strconv"

	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/rpc"
)

// VrgoRPC defines the user RPCs exported by server.
type VrgoRPC int

func (v *VrgoRPC) Execute(req *rpc.Request, resp *rpc.Response) error {
	// TODO: If mode is not primary, then tell client who the new primary is.
	k := strconv.Itoa(req.ClientId)
	res, ok := globals.ClientTable.Get(k)

	// If the client request is already executed before, resend the response.
	if ok && req.RequestNum <= res.(rpc.Response).RequestNum {
		log.Printf("request %+v is already executed; returning previous result %+v directly\n", req, res)
		*resp = res.(rpc.Response)
		return nil
	}

	// First time receiving from this client.
	if !ok {
		log.Printf("first time receiving request %v from client %v\n", req.RequestNum, req.ClientId)
	}

	ch := AddIncomingReq(req)
	select {
	case res := <-ch:
		log.Printf("done processing request; got result %v\n", res.OpResult.Message)
		*resp = *res
	}

	return nil
}
