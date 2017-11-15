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

// PrepareOk is the output type of Echo.
type PrepareOk struct {
	ViewNum    int
	RequestNum int
	Result     string
}
