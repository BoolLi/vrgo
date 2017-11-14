// oplog provides the interface to the in-memory log.
package oplog

import (
	"fmt"
	"log"

	"github.com/BoolLi/vrgo/rpc"
)

// OpRequest represents an operation record in the log.
type OpRequest struct {
	Request rpc.Request
	OpNum   int
}

// OpRequestLog is the in-memory log to store all the records.
type OpRequestLog struct {
	Requests []OpRequest
}

// New creates an OpRequestLog.
func New() *OpRequestLog {
	return &OpRequestLog{}
}

// AppendRequest appends a request along with its opNum to the log.
func (o *OpRequestLog) AppendRequest(request *rpc.Request, opNum int) error {
	log.Printf("oplog adding %v at opNum %v", request, opNum)
	r := OpRequest{Request: *request, OpNum: opNum}
	o.Requests = append(o.Requests, r)
	return nil
}

// ReadLast returns the last request from the log or an error if the log is empty.
func (o *OpRequestLog) ReadLast() (*rpc.Request, int, error) {
	if len(o.Requests) == 0 {
		return nil, 0, fmt.Errorf("OpRequestLog is empty")
	}

	r := o.Requests[len(o.Requests)-1]

	return &r.Request, r.OpNum, nil
}
