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

// Echo returns the exact same message sent by the caller.
func (r *BackupReply) Prepare(args *vrrpc.PrepareArgs, resp *vrrpc.PrepareOk) error {
	log.Printf("got prepare message from primary: %v", *args)
	resp.ViewNum = args.ViewNum
	resp.RequestNum = args.Request.RequestNum
	resp.Result = "result"
	return nil
}

// Register registers a RPC receiver.
func Register(rcvr vrrpc.BackupService) error {
	return rpc.Register(rcvr)
}

func RunBackup() {
	b := new(BackupReply)
	Register(b)
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
