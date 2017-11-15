package rpc

type BackupService interface {
  Prepare(args *PrepareArgs, resp *Reply)
}

// Prepare is the input argument type to Echo.
type PrepareArgs struct {
	ViewNum   int
	Request   Request
	OpNum     int
	CommitNum int
}

// Reply is the output type of Echo.
// TODO: Change this later.
type Reply struct {
	ViewNum    int
	RequestNum int
	Result     string
}
