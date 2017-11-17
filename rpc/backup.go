package rpc

type BackupService interface {
  Prepare(args *PrepareArgs, resp *PrepareOk) error
}

// Prepare is the input argument type to Echo.
type PrepareArgs struct {
	ViewNum   int
	Request   Request
	OpNum     int
	CommitNum int
}

// PrepareOk is the output type of Prepare.
type PrepareOk struct {
	ViewNum    int
	OpNum      int
	Id         int
}

// Commit is sent by primary if no new Prepare message is being sent
type Commit struct {
  ViewNum   int
  CommitNum int
}
