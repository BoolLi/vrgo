// primary implement logic of a primary in VR.
package primary

import (
	"log"
	"net/rpc"
	"strconv"
	"time"

	"github.com/BoolLi/vrgo/oplog"

	vrrpc "github.com/BoolLi/vrgo/rpc"
	cache "github.com/patrickmn/go-cache"
)

// ClientRequest represents the in-memory state of a client request in the primary.
type ClientRequest struct {
	Request vrrpc.Request
	done    chan int
}

const incomingReqsSize = 5

var incomingReqs chan ClientRequest
var opRequestLog *oplog.OpRequestLog
var opNum int
var clientTable *cache.Cache
var viewNum int

// TODO: Keep track of the clients dynamically.
var client *rpc.Client

// Init initializes data structures needed for the primary.
func Init(opLog *oplog.OpRequestLog, t *cache.Cache) error {
	incomingReqs = make(chan ClientRequest, incomingReqsSize)
	opRequestLog = opLog
	clientTable = t

	var err error
	client, err = rpc.DialHTTP("tcp", "localhost:9876")
	if err != nil {
		log.Fatal("failed to connect to client 9876: ", err)
	}

	return nil
}

func ProcessIncomingReq(req *vrrpc.Request) chan int {
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
		vrrpc.Response{
			ViewNum:    viewNum,
			RequestNum: req.RequestNum,
			OpResult:   vrrpc.OperationResult{},
		}, cache.NoExpiration)
	log.Printf("clientTable adding %v at viewNum %v", req, viewNum)

	// 5. Send Prepare messages.
	args := vrrpc.PrepareArgs{
		ViewNum:   viewNum,
		Request:   *req,
		OpNum:     opNum,
		CommitNum: 0,
	}
	var reply vrrpc.PrepareOk
	// TODO: primary should send to all backups and wait for f replies.
	err := client.Call("BackupReply.Prepare", args, &reply)
	if err != nil {
		log.Fatal("backup reply error:", err)
	}
	log.Printf("got reply from client: %v", reply)
	
	// TODO: Intead of calling this, we should wait for f replies on a separate thread.
	// Wait for f PrepareOk messages before
	// 1. Make sure all earlier operations are executed
	// 2. Execute current operation by making up call to service code
	// 3. Increment commit number
	// 4. Respond to client
	// 5. Update client's entry in client table to contain result
	go DummyConsumeIncomingReq()

	return ch
}

// AddIncomingReq adds a vrrpc.Request to the primary to process.
func AddIncomingReq(req *vrrpc.Request) chan int {
	ch := make(chan int)
	r := ClientRequest{
		Request: *req,
		done:    ch,
	}
	incomingReqs <- r
	return ch
}

func DummyConsumeIncomingReq() {
	time.Sleep(10 * time.Second)
	select {
	case r := <-incomingReqs:
		log.Printf("cosuming request %v", r.Request)
		r.done <- 1
	}
}
