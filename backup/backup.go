// backup implement logic of a backup in VR.
package backup

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

var port = flag.Int("backup_port", 9876, "backup port")

// BackupReply defines the basic RPCs exported by server.
type BackupReply int

// Prepare is the input argument type to Echo.
// TODO: Change this later.
type Prepare struct {
	ViewNum   int
	Request   vrrpc.Request
	OpNum     int
	CommitNum int
}

// Reply is the output type of Echo.
// TODO: Change this later.
type Reply struct {
	ViewNum    int
	RequestNum int
	Result     string
}

// Echo returns the exact same message sent by the caller.
func (r *BackupReply) Echo(args *Prepare, resp *Reply) error {
	log.Printf("got prepare message from primary: %v", *args)
	resp.ViewNum = args.ViewNum
	resp.RequestNum = args.Request.RequestNum
	resp.Result = "result"
	return nil
}

func RunBackup() {
	b := new(BackupReply)
	rpc.Register(b)
	rpc.HandleHTTP()

	p := strconv.Itoa(*port)

	l, e := net.Listen("tcp", ":"+p)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	for {
	}
}
