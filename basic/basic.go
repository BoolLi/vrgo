// basic defines the basic RPCs the server exports.
package basic

import (
	"fmt"
	"time"
)

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
	fmt.Printf("Echo: got message %v from client\n", args.Message)
	resp.Message = args.Message
	return nil
}

// DelayedEcho is the same as Echo(), but does it asynchronously.
func (b *Basic) DelayedEcho(args *EchoArgs, resp *EchoResp) error {
	fmt.Printf("DelayedEcho: got message %v from client\n", args.Message)
	done := make(chan string)
	go delay(5*time.Second, args.Message, done)

	timer := time.NewTimer(30 * time.Second)
	select {
	case result := <-done:
		timer.Stop()
		resp.Message = result
	case <-timer.C:
		resp.Message = "timeout"
	}
	return nil
}

func delay(d time.Duration, m string, done chan string) {
	time.Sleep(d)
	done <- fmt.Sprintf("delayed echo response %v after %v seconds", m, d)
}
