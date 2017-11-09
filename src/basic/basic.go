// basic defines the basic RPCs the server exports.
package basic

import "fmt"

// Basic defines the basic RPCs exported by server.
type Basic int

// EchoArgs is the input argument type to Echo.
type EchoArgs struct {
	Message string
}

// EchoResp is the output type of Echo.
type EchoResp struct {
	Message string
}

// Echo returns the exact same message sent by the caller.
func (b *Basic) Echo(args *EchoArgs, resp *EchoResp) error {
	fmt.Printf("got message %v from client", args.Message)
	resp.Message = args.Message
	return nil
}
