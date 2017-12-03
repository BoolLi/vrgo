// backup implement logic of a backup in VR.
package backup

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"

	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/view"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

var (
	incomingPrepareSize = 5
	incomingPrepares    chan PrimaryPrepare
	incomingCommit      chan int // TODO: change to type CommitRequest when defined
	viewTimer           *time.Timer
)

// BackupReply defines the basic RPCs exported by server.
type BackupReply int

// PrimaryPrepare represents the in-memory state of a primary prepare message.
type PrimaryPrepare struct {
	PrepareArgs vrrpc.PrepareArgs
	done        chan vrrpc.PrepareOk
}

// Prepare responds to primary with a PrepareOk message if criteria is met
func (r *BackupReply) Prepare(prepare *vrrpc.PrepareArgs, resp *vrrpc.PrepareOk) error {
	globals.Log("Prepare", "got prepare message from primary: %+v", *prepare)

	ch := AddIncomingPrepare(prepare)
	select {
	case r := <-ch:
		log.Println("backup done processing prepare")
		*resp = r
	}

	return nil
}

func ProcessIncomingPrepares(ctx context.Context) {
	for {
		var primaryPrepare PrimaryPrepare
		select {
		case primaryPrepare = <-incomingPrepares:
			globals.Log("ProcessIncomingPrepares", "consuming prepare %+v from primary", primaryPrepare.PrepareArgs)
		case <-ctx.Done():
			globals.Log("ProcessIncomingPrepares", "backup context cancelled when waiting for incoming prepares: %+v", ctx.Err())
			return
		}

		// TODO: Take the commitNum from the request or from the commit message; send it over to commit service.

		// The Request encapsulated in the prepare message.
		prepareRequest := primaryPrepare.PrepareArgs.Request

		_, lastOp, _ := globals.OpLog.ReadLast(ctx)

		// Backup should block if it does not have op for all earlier requests in its log.
		if primaryPrepare.PrepareArgs.OpNum > (lastOp + 1) {
			// Channel that listens for update from Commit Service
			incomingCommit = make(chan int)
			_ = <-incomingCommit
			log.Print("received a commit from commit service")
		}
		// 1. Increment op number
		globals.OpNum += 1
		// 2. Add request to end of log
		if err := globals.OpLog.AppendRequest(ctx, &prepareRequest, globals.OpNum); err != nil {
			// TODO: Add logic when appending to log fails.
			log.Fatalf("could not write to op request log: %v", err)
		}

		// 3. Update client table
		globals.ClientTable.Set(strconv.Itoa(prepareRequest.ClientId),
			vrrpc.Response{
				ViewNum:    globals.ViewNum,
				RequestNum: prepareRequest.RequestNum,
				OpResult:   vrrpc.OperationResult{},
			})
		globals.Log("ProcessIncomingPrepares", "client table adding %+v at viewNum %v", prepareRequest, globals.ViewNum)

		// 4. Send PrepareOk message to channel for primary
		resp := vrrpc.PrepareOk{
			ViewNum: globals.ViewNum,
			OpNum:   globals.OpNum,
			Id:      *flags.Id,
		}
		globals.Log("ProcessIncomingPrepares", "backup %v sending PrepareOk %+v to primary", *flags.Id, resp)

		primaryPrepare.done <- resp
	}
}

// AddIncomingPrepare adds a vrrpc.PrepareArgs to incomingPrepares queue.
func AddIncomingPrepare(prepare *vrrpc.PrepareArgs) chan vrrpc.PrepareOk {
	// Reset viewTimer.
	viewTimer.Reset(5 * time.Second)
	ch := make(chan vrrpc.PrepareOk)
	r := PrimaryPrepare{
		PrepareArgs: *prepare,
		done:        ch,
	}
	incomingPrepares <- r
	return ch
}

// Register registers a RPC receiver.
func Register(rcvr interface{}) error {
	return rpc.Register(rcvr)
}

// Serve starts an HTTP server to handle RPC requests.
func ServeHTTP() {
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", *flags.Port))
	if err != nil {
		log.Fatalf("failed to listen on port %v: %v", *flags.Port, err)
	}
	http.Serve(l, nil)
}

func Init(ctx context.Context, vt *time.Timer) error {
	incomingPrepares = make(chan PrimaryPrepare, incomingPrepareSize)
	viewTimer = vt

	Register(new(BackupReply))
	Register(new(view.ViewChangeRPC))
	//go ServeHTTP()

	go ProcessIncomingPrepares(ctx)

	return nil
}

func DummyCommitService() {
	time.Sleep(10 * time.Second)
	incomingCommit <- 1
	log.Print("committing some dummy request")
}
