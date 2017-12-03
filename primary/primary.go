// primary implement logic of a primary in VR.
package primary

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"strings"

	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/view"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

type backupPorts []int

func (cp *backupPorts) String() string {
	var ps []string
	for _, p := range *cp {
		ps = append(ps, fmt.Sprintf("%v", p))
	}
	return strings.Join(ps, ", ")
}

func (cp *backupPorts) Set(value string) error {
	p, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*cp = append(*cp, p)
	return nil
}

var ports backupPorts

func init() {
	flag.Var(&ports, "backup_ports", "backup ports")
}

// ClientRequest represents the in-memory state of a client request in the primary.
type ClientRequest struct {
	Request vrrpc.Request
	done    chan *vrrpc.Response
}

const incomingReqsSize = 5

var (
	incomingReqs chan ClientRequest
	backups      []*rpc.Client
)

// Init initializes data structures needed for the primary.
func Init(ctx context.Context) error {
	incomingReqs = make(chan ClientRequest, incomingReqsSize)

	RegisterVrgo(new(VrgoRPC))
	RegisterView(new(view.ViewChangeRPC))
	//go ServeHTTP()

	for _, p := range ports {
		var err error
		c, err := rpc.DialHTTP("tcp", fmt.Sprintf("localhost:%v", p))
		if err != nil {
			log.Fatal(fmt.Sprintf("failed to connect to backup%v: ", p), err)
		}
		backups = append(backups, c)
	}

	go ProcessIncomingReqs(ctx)

	return nil
}

// ProcessIncomingReqs takes requests from incomingReqs queue and processes them.
// Note: This function is going to be the bottleneck because it has to block for each request.
// It cannot delegate waiting for backup replies to other threads, because later requests from the same client
// can reset globals.ClientTable while previous ones are still on the fly.
// The best solution is to create a per-client incoming request queue. This ensures linearizability.
func ProcessIncomingReqs(ctx context.Context) {
	for {
		// 1. Take a request from the incoming request queue.
		var clientReq ClientRequest
		select {
		case clientReq = <-incomingReqs:
			globals.Log("ProcessIncomingReqs", "taking new request from incoming queue: %+v", clientReq.Request)
		case <-ctx.Done():
			globals.Log("ProcessIncomingReqs", "primary context cancelled when waiting for incoming requests: %+v", ctx.Err())
			return
		}

		// 2. Advance op num.
		globals.OpNum += 1

		// 3. Append request to op log.
		if err := globals.OpLog.AppendRequest(ctx, &clientReq.Request, globals.OpNum); err != nil {
			log.Fatalf("could not write %v to op request log: %v", err)
		}

		// 4. Update client table.
		globals.ClientTable.Set(strconv.Itoa(clientReq.Request.ClientId),
			vrrpc.Response{
				ViewNum:    globals.ViewNum,
				RequestNum: clientReq.Request.RequestNum,
				OpResult:   vrrpc.OperationResult{},
			})
		globals.Log("ProcessIncomingReqs", "clientTable adding %+v at viewNum %v", clientReq.Request, globals.ViewNum)

		// 5. Send Prepare messages.
		args := vrrpc.PrepareArgs{
			ViewNum:   globals.ViewNum,
			Request:   clientReq.Request,
			OpNum:     globals.OpNum,
			CommitNum: globals.CommitNum,
		}

		// 6. Wait for f PrepareOks from backups.
		quorumChan := make(chan bool)
		subquorum := len(backups) / 2
		for _, c := range backups {
			go func(c *rpc.Client) {
				var reply vrrpc.PrepareOk
				err := c.Call("BackupReply.Prepare", args, &reply)
				if err != nil {
					globals.Log("ProcessIncomingReqs", "got error from backup: %v", err)
					return
				}
				globals.Log("ProcessIncomingReqs", "got PrepareOK from backup: %+v", reply)
				quorumChan <- true
			}(c)
		}
		quorumReadyChan := make(chan int)

		// Block when either of the following cases happens first:
		// 1. Primary gets f replies from backups.
		// 2. Primary's context gets cancelled.
		go func() {
			for i := 0; i < subquorum; i++ {
				<-quorumChan
			}
			quorumReadyChan <- 1
		}()
		select {
		case _ = <-quorumReadyChan:
			globals.Log("ProcessIncomingReqs", "got %v replies from backups; marking request as done", subquorum)
		case <-ctx.Done():
			globals.Log("ProcessIncomingReqs", "primary context cancelled when waiting for %v replies from backups: %+v", subquorum, ctx.Err())
			// Undo current operation.
			globals.OpNum -= 1
			globals.OpLog.Undo(ctx)
			globals.ClientTable.Undo(strconv.Itoa(clientReq.Request.ClientId))
			return
		}

		// Now we consider operation commmited.

		// 7. Exeucte the request.
		globals.Log("ProcessIncomingReqs", "executing %v", clientReq.Request.Op.Message)

		// 8. Increment the commit number.
		globals.CommitNum += 1

		// 9. Send reply back to client by pushing the reply to the channel.
		clientReq.done <- &vrrpc.Response{
			ViewNum:    globals.ViewNum,
			RequestNum: clientReq.Request.RequestNum,
			OpResult:   vrrpc.OperationResult{Message: clientReq.Request.Op.Message},
		}
	}
}

// AddIncomingReq adds a vrrpc.Request to incomingReqs queue.
func AddIncomingReq(req *vrrpc.Request) chan *vrrpc.Response {
	ch := make(chan *vrrpc.Response)
	r := ClientRequest{
		Request: *req,
		done:    ch,
	}
	incomingReqs <- r
	return ch
}
