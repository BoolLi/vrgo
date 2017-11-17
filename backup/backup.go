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
	"github.com/BoolLi/vrgo/oplog"

	vrrpc "github.com/BoolLi/vrgo/rpc"
	cache "github.com/patrickmn/go-cache"
)

var opRequestLog *oplog.OpRequestLog
var opNum int
var clientTable *cache.Cache
var viewNum int
var incomingPrepareSize = 5
var incomingPrepares chan PrimaryPrepare
var incomingCommit chan int // TODO: change to type CommitRequest when defined

// BackupReply defines the basic RPCs exported by server.
type BackupReply int

// PrimaryPrepare represents the in-memory state of a primary prepare message.
type PrimaryPrepare struct {
	PrepareArgs vrrpc.PrepareArgs
	done        chan vrrpc.PrepareOk
}

// Prepare responds to primary with a PrepareOk message if criteria is met
func (r *BackupReply) Prepare(prepare *vrrpc.PrepareArgs, resp *vrrpc.PrepareOk) error {
	log.Printf("got prepare message from primary: %+v\n", *prepare)

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
			log.Printf("consuming prepare %+v from primary\n", primaryPrepare.PrepareArgs)
		}

		// TODO: Take the commitNum from the request or from the commit message; send it over to commit service.

		// The Request encapsulated in the prepare message.
		prepareRequest := primaryPrepare.PrepareArgs.Request

		_, lastOp, _ := opRequestLog.ReadLast(ctx)

		// Backup should block if it does not have op for all earlier requests in its log.
		if primaryPrepare.PrepareArgs.OpNum > (lastOp + 1) {
			// Channel that listens for update from Commit Service
			incomingCommit = make(chan int)
			_ = <-incomingCommit
			log.Print("received a commit from commit service")
		}
		// 1. Increment op number
		opNum += 1
		// 2. Add request to end of log
		if err := opRequestLog.AppendRequest(ctx, &prepareRequest, opNum); err != nil {
			// TODO: Add logic when appending to log fails.
			log.Fatalf("could not write to op request log: %v", err)
		}

		// 3. Update client table
		clientTable.Set(strconv.Itoa(prepareRequest.ClientId),
			vrrpc.Response{
				ViewNum:    viewNum,
				RequestNum: prepareRequest.RequestNum,
				OpResult:   vrrpc.OperationResult{},
			}, cache.NoExpiration)
		log.Printf("clientTable adding %+v at viewNum %v\n", prepareRequest, viewNum)

		// 4. Send PrepareOk message to channel for primary
		resp := vrrpc.PrepareOk{
			ViewNum: viewNum,
			OpNum:   opNum,
			Id:      *flags.Id,
		}
		log.Printf("backup %v sending PrepareOk %+v to primary\n", *flags.Id, resp)

		primaryPrepare.done <- resp
	}
}

// AddIncomingPrepare adds a vrrpc.PrepareArgs to incomingPrepares queue.
func AddIncomingPrepare(prepare *vrrpc.PrepareArgs) chan vrrpc.PrepareOk {
	ch := make(chan vrrpc.PrepareOk)
	r := PrimaryPrepare{
		PrepareArgs: *prepare,
		done:        ch,
	}
	incomingPrepares <- r
	return ch
}

// Register registers a RPC receiver.
func Register(rcvr vrrpc.BackupService) error {
	return rpc.Register(rcvr)
}

// Serve starts an HTTP server to handle RPC requests.
func Serve() {
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%v", *flags.Port))
	if err != nil {
		log.Fatalf("failed to listen on port %v: %v", *flags.Port, err)
	}
	http.Serve(l, nil)
}

func Init(ctx context.Context, opLog *oplog.OpRequestLog, t *cache.Cache) error {
	opRequestLog = opLog
	clientTable = t
	incomingPrepares = make(chan PrimaryPrepare, incomingPrepareSize)

	go ProcessIncomingPrepares(ctx)

	return nil
}

func DummyCommitService() {
	time.Sleep(10 * time.Second)
	incomingCommit <- 1
	log.Print("committing some dummy request")
}
