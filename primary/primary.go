package primary

import (
	"log"

	"github.com/BoolLi/vrgo/rpc"
)

// ClientRequest represents the in-memory state of a client request in the primary.
type ClientRequest struct {
	Request rpc.Request
	done    chan int
}

const incomingReqsSize = 5

var incomingReqs chan ClientRequest

// New initializes data structures needed for the primary.
func New() {
	incomingReqs = make(chan ClientRequest, incomingReqsSize)
}

// AddIncomingReq adds a rpc.Request to the primary to process.
func AddIncomingReq(req *rpc.Request) chan int {
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
