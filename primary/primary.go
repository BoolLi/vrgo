package primary

import (
	"log"
	"strconv"

	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/rpc"

	cache "github.com/patrickmn/go-cache"
)

// ClientRequest represents the in-memory state of a client request in the primary.
type ClientRequest struct {
	Request rpc.Request
	done    chan int
}

const incomingReqsSize = 5

var incomingReqs chan ClientRequest
var opRequestLog *oplog.OpRequestLog
var opNum int
var clientTable *cache.Cache
var viewNum int

// Init initializes data structures needed for the primary.
func Init(opLog *oplog.OpRequestLog, t *cache.Cache) error {
	incomingReqs = make(chan ClientRequest, incomingReqsSize)
	opRequestLog = opLog
	clientTable = t
	return nil
}

func ProcessIncomingReq(req *rpc.Request) chan int {
	// 1. Add request to incoming queue.
	ch := AddIncomingReq(req)
	// 2. Advance op num.
	opNum += 1

	// 3. Append request to op log.
	if err := opRequestLog.AppendRequest(req, opNum); err != nil {
		// TODO: Add logic when appending to log fails.
		log.Fatalf("could not write to op request log: %v", err)
	}

	// 4. Update client table.
	clientTable.Set(strconv.Itoa(req.ClientId),
		rpc.Response{
			ViewNum:    viewNum,
			RequestNum: req.RequestNum,
			OpResult:   rpc.OperationResult{},
		}, cache.NoExpiration)
	log.Printf("clientTable adding %v at viewNum %v", req, viewNum)

	// 5. Send Prepare messages.

	return ch
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
