package rpc

// ViewService is the RPC to perform a view change.
type ViewService interface {
	// StartViewChange initiates a view change.
	StartViewChange(*StartViewChangeArgs, *StartViewChangeResp) error
	// DoViewChange tells the new primary to start a new view.
	DoViewChange(*DoViewChangeArgs, *DoViewChangeResp) error
	// StartView tells the backups to transition to a new view.
	StartView(*StartViewArgs, *StartViewResp) error
}

// StartViewChangeArgs is the arguments to start a view change.
type StartViewChangeArgs struct {
	ViewNum int
	Id      int
}

// StartViewChangeResp is the response to a StartViewChange message.
type StartViewChangeResp struct {
}

// DoViewChangeArgs is the arguments to tell the new primary to start a new view.
type DoViewChangeArgs struct {
	ViewNum             int
	Log                 []Request
	LatestNormalViewNum int
	OpNum               int
	CommitNum           int
	Id                  int
}

// DoViewChangeResp is the response to a DoViewChange message.
type DoViewChangeResp struct {
}

// StartViewArgs is the arguments for the primary to start a new view.
type StartViewArgs struct {
	ViewNum   int
	Log       []Request
	OpNum     int
	CommitNum int
}

// StartViewResp is the response to a StartView message.
type StartViewResp struct {
}
