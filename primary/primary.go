// primary implement logic of a primary in VR.
package primary

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"strings"
	"time"

	"github.com/BoolLi/vrgo/oplog"

	vrrpc "github.com/BoolLi/vrgo/rpc"
	cache "github.com/patrickmn/go-cache"
)

type clientPorts []int

func (cp *clientPorts) String() string {
	var ps []string
	for _, p := range *cp {
		ps = append(ps, fmt.Sprintf("%v", p))
	}
	return strings.Join(ps, ", ")
}

func (cp *clientPorts) Set(value string) error {
	p, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*cp = append(*cp, p)
	return nil
}

var ports clientPorts

func init() {
	flag.Var(&ports, "backup_ports", "backup ports")
}

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

	RegisterRPC(new(VrgoRPC))
	go ServeHTTP()

	// TODO: Connect to multiple backups instead of just one.
	p := ports[0]
	var err error
	client, err = rpc.DialHTTP("tcp", fmt.Sprintf("localhost:%v", p))
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to connect to client %v: ", p), err)
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
	// TODO: primary should send to all clients and wait for f replies.
	err := client.Call("BackupReply.Prepare", args, &reply)
	if err != nil {
		log.Fatal("backup reply error:", err)
	}
	log.Printf("got reply from client: %v", reply)
	// TODO: Intead of calling this, we should wait for f replies on a separate thread.
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
