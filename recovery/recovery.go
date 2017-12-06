package recovery

import (
	vrrpc "github.com/BoolLi/vrgo/rpc"
)

type RecoveryRPC int

func (r *RecoveryRPC) Recover(request *vrrpc.RecoveryRequest, response *vrrpc.RecoveryResponse) error {
	return nil
}

func SendRecoveryRequest() {
}
