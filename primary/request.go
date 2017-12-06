package primary

import (
	"fmt"
	"strconv"

	"github.com/BoolLi/vrgo/globals"
	vrrpc "github.com/BoolLi/vrgo/rpc"
)

// VrgoRPC defines the user RPCs exported by server.
type VrgoRPC int

func (v *VrgoRPC) Execute(req *vrrpc.Request, resp *vrrpc.Response) error {
	// If mode is not primary, then tell client who the new primary is.
	mode := globals.Mode

	if mode != "primary" {
		globals.Log("Execute", "not primary; view num: %v", globals.ViewNum)
		var err string
		if mode == "backup" {
			globals.Log("Execute", "I am not primary anymore; view num: %v", globals.ViewNum)
			err = fmt.Sprintf("not primary")
		} else if mode == "viewchange" || mode == "viewchange-init" {
			globals.Log("Execute", "under view change")
			err = fmt.Sprintf("view change")
		}
		*resp = vrrpc.Response{
			ViewNum: globals.ViewNum,
			Err:     err,
		}
		return nil
	}

	k := strconv.Itoa(req.ClientId)
	res, ok := globals.ClientTable.Get(k)

	// If the client request is already executed before, resend the response.
	if ok && req.RequestNum <= res.(vrrpc.Response).RequestNum {
		globals.Log("Execute", "request %+v is already executed; returning previous result %+v directly", req, res)
		*resp = res.(vrrpc.Response)
		return nil
	}

	// First time receiving from this client.
	if !ok {
		globals.Log("Execute", "first time receiving request %v from client %v\n", req.RequestNum, req.ClientId)
	}

	ch := AddIncomingReq(req)
	select {
	case res := <-ch:
		globals.Log("Execute", "done processing request; got result %v\n", res.OpResult.Message)
		*resp = *res
	}

	return nil
}
