package rpc

type RecoveryService interface {
  Recover(request *RecoveryRequest, response *RecoveryResponse) error
}

type RecoveryRequest struct {
	Id      int
  Nonce   int
}

type RecoveryResponse struct {
	ViewNum    int
  Nonce      int
  Log        []OpRequest
  OpNum      int
  CommitNum  int
	Id         int
}
