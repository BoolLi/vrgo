package oplog

import (
	"fmt"
	"log"

	"github.com/BoolLi/vrgo/rpc"
)

type OpRequest struct {
	Request rpc.Request
	OpNum   int
}

type OpRequestLog struct {
	Requests []OpRequest
}

func New() *OpRequestLog {
	return &OpRequestLog{}
}

func (o *OpRequestLog) AppendRequest(request *rpc.Request, opNum int) error {
	log.Printf("oplog adding %v at opNum %v", request, opNum)
	r := OpRequest{Request: *request, OpNum: opNum}
	o.Requests = append(o.Requests, r)
	return nil
}

func (o *OpRequestLog) ReadLast() (*rpc.Request, int, error) {
	if len(o.Requests) == 0 {
		return nil, 0, fmt.Errorf("OpRequestLog is empty")
	}

	r := o.Requests[len(o.Requests)-1]

	return &r.Request, r.OpNum, nil
}
