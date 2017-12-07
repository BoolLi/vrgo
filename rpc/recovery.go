package rpc

// RecoveryService is the RPC to perform a recovery.
type RecoveryService interface {
	Recover(request *RecoveryRequest, response *RecoveryResponse) error
}

// RecoveryRequest is the request to start a recovery.
type RecoveryRequest struct {
	Id    int
	Nonce int
}

// RecoveryRequest is the response to a recovery request.
type RecoveryResponse struct {
	ViewNum   int
	Nonce     int
	Log       []OpRequest
	OpNum     int
	CommitNum int
	Id        int
	Mode      string
}
