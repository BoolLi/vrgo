// rpc defines all the RPC interfaces.
package rpc

// TODO: Also define interfaces in this package.

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
