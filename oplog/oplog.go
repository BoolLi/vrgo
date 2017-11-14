package oplog

import (
	"fmt"

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

func (o *OpRequestLog) AppendRequest(request *rpc.Request, opNum int) {
	r := OpRequest{Request: *request, OpNum: opNum}
	o.Requests = append(o.Requests, r)
}

func (o *OpRequestLog) ReadLast() (*rpc.Request, int, error) {
	if len(o.Requests) == 0 {
		return nil, -1, fmt.Errorf("OpRequestLog is empty")
	}

	r := o.Requests[len(o.Requests)-1]

	return &r.Request, r.OpNum, nil
}
