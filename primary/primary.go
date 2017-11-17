// primary implement logic of a primary in VR.
package primary

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"strings"

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
var clients []*rpc.Client

// Init initializes data structures needed for the primary.
func Init(opLog *oplog.OpRequestLog, t *cache.Cache) error {
	incomingReqs = make(chan ClientRequest, incomingReqsSize)
	opRequestLog = opLog
	clientTable = t

	RegisterRPC(new(VrgoRPC))
	go ServeHTTP()

	// TODO: Connect to multiple backups instead of just one.
	for _, p := range ports {
		var err error
		c, err := rpc.DialHTTP("tcp", fmt.Sprintf("localhost:%v", p))
		if err != nil {
			log.Fatal(fmt.Sprintf("failed to connect to client %v: ", p), err)
		}
		clients = append(clients, c)
	}

	go ProcessIncomingReqs()

	return nil
}

// ProcessIncomingReqs takes requests from incomingReqs queue and processes them.
// Note: This function is going to be the bottleneck because it has to block for each request.
// It cannot delegate waiting for backup replies to other threads, because later requests from the same client
// can reset clientTable while previous ones are still on the fly.
// The best solution is to create a per-client incoming request queue. This ensures linearizability.
func ProcessIncomingReqs() {
	for {
		// 1. Take a request from the incoming request queue.
		var clientReq ClientRequest
		select {
		case clientReq = <-incomingReqs:
			log.Printf("consuming request %v", clientReq.Request)
		}

		// 2. Advance op num.
		opNum += 1

		// 3. Append request to op log.
		if err := opRequestLog.AppendRequest(&clientReq.Request, opNum); err != nil {
			log.Fatalf("could not write %v to op request log: %v", err)
		}

		// 4. Update client table.
		clientTable.Set(strconv.Itoa(clientReq.Request.ClientId),
			vrrpc.Response{
				ViewNum:    viewNum,
				RequestNum: clientReq.Request.RequestNum,
				OpResult:   vrrpc.OperationResult{},
			}, cache.NoExpiration)
		log.Printf("clientTable adding %v at viewNum %v", clientReq.Request, viewNum)

		// 5. Send Prepare messages.
		args := vrrpc.PrepareArgs{
			ViewNum:   viewNum,
			Request:   clientReq.Request,
			OpNum:     opNum,
			CommitNum: 0,
		}

		quorumChan := make(chan bool)
		quorum := len(clients)/2 + 1
		for _, c := range clients {
			go func() {
				var reply vrrpc.PrepareOk
				err := c.Call("BackupReply.Prepare", args, &reply)
				if err != nil {
					log.Printf("got error from client: %v", err)
					return
				}
				log.Printf("got reply from client: %v", reply)
				quorumChan <- true
			}()
		}
		// TODO: If we don't get quorum-1 good replies, we will get stuck.
		for i := 0; i < quorum; i++ {
			<-quorumChan
		}
		log.Printf("got %v replies from clients; marking request as done", quorum)

		clientReq.done <- 1
	}
}

// AddIncomingReq adds a vrrpc.Request to incomingReqs queue.
func AddIncomingReq(req *vrrpc.Request) chan int {
	ch := make(chan int)
	r := ClientRequest{
		Request: *req,
		done:    ch,
	}
	incomingReqs <- r
	return ch
}
