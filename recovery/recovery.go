package recovery

import (
	"context"
	"log"
	"math/rand"
	"net/rpc"
	"strconv"

	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

type RecoveryRPC int

// RegisterRecovery registers a Recovery RPC receiver.
func RegisterRecovery(rcvr vrrpc.RecoveryService) error {
	return rpc.Register(rcvr)
}

func (r *RecoveryRPC) Recover(request *vrrpc.RecoveryRequest, response *vrrpc.RecoveryResponse) error {
	*response = vrrpc.RecoveryResponse{
		ViewNum: globals.ViewNum,
		Nonce:   request.Nonce,
		Id:      *flags.Id,
		Mode:    globals.Mode,
	}
	if globals.Mode == "primary" {
		response.Log = globals.OpLog.Requests
		response.OpNum = globals.OpNum
		response.CommitNum = globals.CommitNum
	}
	return nil
}

func PerformRecovery(ctx context.Context) bool {
	recoveryPrimaryChan := make(chan *vrrpc.RecoveryResponse)
	recoveryBackupChan := make(chan *vrrpc.RecoveryResponse)
	var responses []*vrrpc.RecoveryResponse
	subquorum := len(globals.AllOtherPorts()) / 2
	nonce := rand.Int()

	for _, port := range globals.AllOtherPorts() {
		globals.Log("PerformRecovery", "sending Recovery request to replica with port %v", port)
		p := strconv.Itoa(port)
		client, err := globals.GetOrCreateClient("localhost:" + p)
		if err != nil {
			log.Fatal("dialing:", err)
		}

		req := &vrrpc.RecoveryRequest{
			Id:    *flags.Id,
			Nonce: nonce,
		}

		go func(c *rpc.Client) {
			var resp vrrpc.RecoveryResponse
			err := c.Call("RecoveryRPC.Recover", req, &resp)
			if err != nil {
				globals.Log("PerformRecovery", "got error from replica: %v", err)
				return
			}
			globals.Log("PerformRecovery", "got RecoveryResponse from replica: %+v", resp)
			if resp.Mode == "primary" {
				recoveryPrimaryChan <- &resp
			} else if resp.Mode == "backup" {
				recoveryBackupChan <- &resp
			} else {
				// Other replicas are under view change. Abort this one and restart recovery.
				// Write to viewchangeChan.
				globals.Log("PerformRecovery", "other nodes are under view change: %v", resp.Mode)
			}
		}(client)
	}

	recoveryReadyChan := make(chan int)

	// Block when either of the following cases happens first:
	// 1. Gets f+1 replies from replicas, one of which is from primary.
	// 2. Context gets cancelled.
	// 3. Timer expires.
	// 4. Other replicas are under view change.
	go func() {
		res := <-recoveryPrimaryChan
		responses = append(responses, res)
		for i := 0; i < subquorum; i++ {
			res = <-recoveryBackupChan
			responses = append(responses, res)
		}
		recoveryReadyChan <- 1
	}()

	select {
	case _ = <-recoveryReadyChan:
		globals.Log("PerformRecovery", "got recovery responses: %+v", responses)
		return applyRecoveryResps(responses)
	case <-ctx.Done():
		globals.Log("PerformRecovery", "recovery context cancelled when waiting for %v replies from backups: %+v", subquorum, ctx.Err())
		return false
		// 1. case timerChan
		// 2. case viewchangeChan
	}

}

func applyRecoveryResps(responses []*vrrpc.RecoveryResponse) bool {
	// 1. Check if all nonces are the same.
	nonce := responses[0].Nonce
	for _, r := range responses {
		if r.Nonce != nonce {
			globals.Log("applyRecoveryResps", "got different nonces from different replies")
			return false
		}
	}

	// 2. Update replica state to primary's state.
	var primaryResp *vrrpc.RecoveryResponse
	for _, r := range responses {
		if r.Mode == "primary" {
			primaryResp = r
		}
	}
	if primaryResp == nil {
		globals.Log("applyRecoveryResps", "no primary response found")
		return false
	}

	globals.ViewNum = primaryResp.ViewNum
	globals.OpLog.Requests = primaryResp.Log
	globals.OpNum = primaryResp.OpNum
	globals.CommitNum = primaryResp.CommitNum
	globals.Log("applyRecoveryResps", "finished recovery; view num: %v; op num: %v; commit num: %v", globals.ViewNum, globals.OpNum, globals.CommitNum)
	return true
}
