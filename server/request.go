package server

import (
	"log"
	"strconv"

	cache "github.com/patrickmn/go-cache"
)

// VrgoRPC defines the user RPCs exported by server.
type VrgoRPC int

// Request is the input argument type to RequestRPC.
type Request struct {
	Op         Operation
	ClientId   int
	RequestNum int
	// Do we need view number as well?
}

// Response is the output type of RequestRPC.
type Response struct {
	ViewNum    int
	RequestNum int
	OpResult   OperationResult
}

// Operation is the user operation.
type Operation struct {
	Message string
}

// OperationResult is the result of the user operation.
type OperationResult struct {
	Message string
}

func (v *VrgoRPC) Execute(req *Request, resp *Response) error {
	k := strconv.Itoa(req.ClientId)
	res, ok := clientTable.Get(k)

	// If the client request is already executed before, resend the response.
	if ok && req.RequestNum <= res.(Response).RequestNum {
		log.Printf("request %v is already executed; returning previous result %v directly.\n", req, res)
		*resp = res.(Response)
		return nil
	}

	// First time receiving from this client.
	if !ok {
		log.Printf("first time receiving request %v from client %v\n", req.RequestNum, req.ClientId)
		clientTable.Set(k,
			Response{
				ViewNum:    0,
				RequestNum: req.RequestNum,
				OpResult:   OperationResult{},
			}, cache.NoExpiration)
		// Push request to imcoming channel.
	}

	// Third case.
	ch := AddIncomingReq(req)
	select {
	case _ = <-ch:
		*resp = Response{}
	}

	return nil
}
