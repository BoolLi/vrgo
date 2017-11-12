// TODO: Change this into its own package.
package server

import "log"

// ClientRequest represents the in-memory state of a client request in the primary.
type ClientRequest struct {
	Request Request
	done    chan int
}

const incomingReqsSize = 5

var incomingReqs chan ClientRequest

func New() {
	incomingReqs = make(chan ClientRequest, incomingReqsSize)
}

func AddIncomingReq(req *Request) chan int {
	ch := make(chan int)
	r := ClientRequest{
		Request: *req,
		done:    ch,
	}
	incomingReqs <- r
	return ch
}

func DummyConsumeIncomingReq() {
	select {
	case r := <-incomingReqs:
		log.Printf("cosuming request %v", r.Request)
		r.done <- 1
	}
}
