// rpc defines all the RPC interfaces.
package rpc

// VrgoService defines the APIs Vrgo exposes to users.
type VrgoService interface {
	Execute(*Request, *Response) error
}

// Request is the input argument type to RequestRPC.
type Request struct {
	Op         Operation
	ClientId   int
	RequestNum int
	// Do we need view number as well?
}

// OpRequest represents an operation record that has a Request and a operation number.
type OpRequest struct {
	Request Request
	OpNum   int
}

// Response is the output type of RequestRPC.
type Response struct {
	ViewNum    int
	RequestNum int
	OpResult   OperationResult
	Err        string
}

// Operation is the user operation.
type Operation struct {
	Message string
}

// OperationResult is the result of the user operation.
type OperationResult struct {
	Message string
}
