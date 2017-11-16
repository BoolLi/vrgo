// backup implement logic of a backup in VR.
package backup

import (
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
var incomingCommit chan int // TODO: change to type CommitRequest when defined

// BackupReply defines the basic RPCs exported by server.
type BackupReply int

// Echo returns the exact same message sent by the caller.
func (r *BackupReply) Prepare(args *vrrpc.PrepareArgs, resp *vrrpc.PrepareOk) error {
	log.Printf("got prepare message from primary: %v", *args)

	_, lastOp, _ := opRequestLog.ReadLast()

	// Backup should block if it does not have op for all earlier requests in its log.
	if args.OpNum > (lastOp + 1) {
		// Channel that listens for update from Commit Service
		incomingCommit = make(chan int)
		_ = <- incomingCommit
		log.Print("received a commit from commit service")
	}
	// 1. Increment op number
	opNum += 1
	// 2. Add request to end of log
	if err := opRequestLog.AppendRequest(&args.Request, opNum); err != nil {
		// TODO: Add logic when appending to log fails.
		log.Fatalf("could not write to op request log: %v", err)
	}

	// 3. Update client table
	clientTable.Set(strconv.Itoa(args.Request.ClientId),
		vrrpc.Response{
			ViewNum:    viewNum,
			RequestNum: args.Request.RequestNum,
			OpResult:   vrrpc.OperationResult{},
		}, cache.NoExpiration)
	log.Printf("clientTable adding %v at viewNum %v", args.Request, viewNum)

	// 4. Send PrepareOk message to primary
	*resp = vrrpc.PrepareOk{
		ViewNum:	viewNum,
		OpNum:		opNum,
		Id:				333, // TODO: fetch Id
	}

	return nil
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

func Init(opLog *oplog.OpRequestLog, t *cache.Cache) error {
	opRequestLog = opLog
	clientTable = t
	return nil
}

func DummyCommitService(){
	time.Sleep(10 * time.Second)
	incomingCommit <- 1
	log.Print("committing some dummy request")
}
