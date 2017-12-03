package view

import (
	"log"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/oplog"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

// ViewChangeRPC implements the ViewService interface.
type ViewChangeRPC int

type mutexDoViewChangeArgs struct {
	sync.Mutex
	Args []*vrrpc.DoViewChangeArgs
}

// Locked locks the value.
func (m *mutexDoViewChangeArgs) Locked(f func()) {
	m.Lock()
	defer m.Unlock()
	f()
}

var (
	// A buffered channel to signal the monitor thread to change to view change mode.
	// Note: we use a buffered channel here because multiple threads running StartViewChange() could be
	// sending signals to the channel at the same time, but the monitor thread can only consume one of them.
	// Having a buffered channel ensures that while only the first signal is consumed, the rest of the threads do not block.
	StartViewChangeChan chan int = make(chan int, len(globals.AllPorts))

	// A channel to notify the monitor that view change is done and what mode should the replica switch to.
	ViewChangeDone chan string = make(chan string)

	startViewChangeReceived  globals.MutexInt
	currentProposedViewNum   globals.MutexInt
	doViewChangeArgsReceived mutexDoViewChangeArgs
	sendDoViewChangeExecuted globals.MutexBool
	subquorum                = len(globals.AllPorts) / 2
)

// StartViewChange handles the StartViewChange RPC.
// This function is triggered whenever the node receives a StartViewChange message. When triggered, the node will stop what it is
// doing right now and start the view change protocol. It implements step 1 and 2 in section 4.2 in the paper.
// This function is thread-safe, so multiple nodes can call this RPC on the same node concurrently.
func (v *ViewChangeRPC) StartViewChange(args *vrrpc.StartViewChangeArgs, resp *vrrpc.StartViewChangeResp) error {
	log.Printf("replica %v receives StartViewChange with %v from %v.\n", *flags.Id, args.ViewNum, args.Id)

	if args.ViewNum <= globals.ViewNum {
		// If the proposed view num is smaller than the current view num, do nothing.
		log.Printf("exiting because proposed view num %v is no larger than current view num %v", args.ViewNum, globals.ViewNum)
		return nil
	}

	// Send a signal to the monitor to change to the view change mode.
	StartViewChangeChan <- 1

	// Lock the view change states to prevent race conditions across multiple threads.
	currentProposedViewNum.Lock()
	startViewChangeReceived.Lock()
	defer currentProposedViewNum.Unlock()
	defer startViewChangeReceived.Unlock()

	// If somebody else proposes a view with a larger view num, we should advocate that instead of the old one.
	if args.ViewNum > currentProposedViewNum.V {
		log.Printf("the view num %v proposed is larger than the current proposed view num %v\n", args.ViewNum, currentProposedViewNum.V)
		startViewChangeReceived.V = 0
		currentProposedViewNum.V = args.ViewNum

		// Send StartViewChange to all other nodes.
		for _, p := range AllOtherPorts() {
			SendStartViewChange(p, args.ViewNum, *flags.Id)
		}
	}
	startViewChangeReceived.V += 1
	log.Printf("replica %v increments startViewChangeReceived to %v\n", *flags.Id, startViewChangeReceived.V)

	// Only send DoViewChange when enough StartViewChange messages have been received and DoViewChange hasn't been sent before.
	sendDoViewChangeExecuted.Locked(func() {
		if startViewChangeReceived.V > subquorum && !sendDoViewChangeExecuted.V {
			log.Printf("replica %v got more than %v StartViewChange messages\n", *flags.Id, subquorum)
			// TODO: Should we do this in a separate thread?
			sendDoViewChange(currentProposedViewNum.V, globals.ViewNum, globals.OpNum, globals.CommitNum, *flags.Id)
			sendDoViewChangeExecuted.V = true
			// TODO: Clear startViewChangeReceived, currentProposedViewNum, doViewChangeArgsReceived, and sendDoViewChangeExecuted somewhere.
		}
	})
	return nil
}

// DoViewChange handles the DoViewChange RPC.
// This function is triggered when the new primary receives a DoViewChange message. It only starts a new view when enough DoViewChange
// messages are received. It is thread-safe so multiple nodes can send DoViewChange messages to the new primary concurrently.
func (v *ViewChangeRPC) DoViewChange(args *vrrpc.DoViewChangeArgs, resp *vrrpc.DoViewChangeResp) error {
	return runDoViewChange(args, resp)
}

// StartView handles the StartView RPC.
func (v *ViewChangeRPC) StartView(args *vrrpc.StartViewArgs, resp *vrrpc.StartViewResp) error {
	log.Printf("replica %v got StartView from new primary: %+v", *flags.Id, args)
	ViewChangeDone <- "backup"
	return nil
}

// ClearViewChangeStates clears the intermediate states of the current view change.
// This function is atomic and thread-safe.
func ClearViewChangeStates() {
	startViewChangeReceived.Lock()
	defer startViewChangeReceived.Unlock()
	currentProposedViewNum.Lock()
	defer currentProposedViewNum.Unlock()
	doViewChangeArgsReceived.Lock()
	defer doViewChangeArgsReceived.Unlock()
	sendDoViewChangeExecuted.Lock()
	defer sendDoViewChangeExecuted.Unlock()

	for len(StartViewChangeChan) > 0 {
		<-StartViewChangeChan
	}

	startViewChangeReceived.V = 0
	currentProposedViewNum.V = 0
	doViewChangeArgsReceived.Args = nil
	sendDoViewChangeExecuted.V = false
}

func runDoViewChange(args *vrrpc.DoViewChangeArgs, resp *vrrpc.DoViewChangeResp) error {
	if args.ViewNum <= globals.ViewNum {
		// If the proposed view num is smaller than the current view num, do nothing.
		log.Printf("received do view change view num %v <= current view num %v", args.ViewNum, globals.ViewNum)
		return nil
	}

	doViewChangeArgsReceived.Lock()
	defer doViewChangeArgsReceived.Unlock()

	doViewChangeArgsReceived.Args = append(doViewChangeArgsReceived.Args, args)
	if len(doViewChangeArgsReceived.Args) != subquorum {
		log.Printf("replica %v received %v DoViewChanges != subquorum %v", *flags.Id, len(doViewChangeArgsReceived.Args), subquorum)
		return nil
	}

	// Make sure all the DoViewChange messages have the same view num.
	if !sameViewNums() {
		log.Fatalf("replica %v received DoViewChange messages with different view nums: %+v\n", *flags.Id, doViewChangeArgsReceived)
	}

	log.Printf("replica %v becomes the new primary", *flags.Id)

	// 1. Set new view num.
	log.Printf("previous view num: %v; new view num: %v", globals.ViewNum, args.ViewNum)
	globals.ViewNum = args.ViewNum

	// 2. Update op log to be the one with the largest latest normal view num.
	refreshLog()
	log.Printf("oplog: %+v", globals.OpLog)

	// 3. Update the op num to that of the topmost entry in the new log.
	_, opNum, err := globals.OpLog.ReadLast(globals.CtxCancel)
	if err != nil {
		log.Fatalf("failed to read the last entry in the new log: %v", err)
	}
	log.Printf("previous op num: %v; new op num: %v", globals.OpNum, opNum)
	globals.OpNum = opNum

	// 4. Set commit num to the largest such number it received in the DoViewChange messages.
	refreshCommitNum()

	// 5. Send StartView to all other replicas.
	for _, p := range AllOtherPorts() {
		sendStartView(p)
	}

	// 6. Notify monitor to switch to primary mode.
	ViewChangeDone <- "primary"
	return nil
}

func refreshLog() {
	maxNormalViewNum := -1
	var l *[]vrrpc.OpRequest
	for _, args := range doViewChangeArgsReceived.Args {
		if args.LatestNormalViewNum > maxNormalViewNum {
			l = &args.Log
			maxNormalViewNum = args.LatestNormalViewNum
		}
	}
	log.Printf("changing oplog to the log in the message with latest normal view num %v", maxNormalViewNum)
	globals.OpLog = &oplog.OpRequestLog{Requests: *l}
	// TODO: If several messages have the same v', selects the one among them with the largest op num.
}

func refreshCommitNum() {
	maxCommitNum := 0
	for _, args := range doViewChangeArgsReceived.Args {
		if args.CommitNum > maxCommitNum {
			maxCommitNum = args.CommitNum
		}
	}
	log.Printf("changing commit num from %v to %v", globals.CommitNum, maxCommitNum)
	globals.CommitNum = maxCommitNum
}

func sameViewNums() bool {
	vn := doViewChangeArgsReceived.Args[0].ViewNum
	for _, arg := range doViewChangeArgsReceived.Args {
		if arg.ViewNum != vn {
			return false
		}
	}
	return true
}

// AllOtherPorts returns all the other replica ports except for that of the current node.
// TODO: Move this function to a proper place.
func AllOtherPorts() []int {
	var ps []int
	for _, p := range globals.AllPorts {
		if p != *flags.Port {
			ps = append(ps, p)
		}
	}
	return ps
}

// InitiateStartViewChange initiates a view change protocol by sending StartViewChange messages to all other replicas.
func InitiateStartViewChange() {
	currentProposedViewNum.Locked(func() {
		currentProposedViewNum.V += 1
		for _, p := range AllOtherPorts() {
			SendStartViewChange(p, currentProposedViewNum.V, *flags.Id)
		}
	})
}

// SendStartViewChange sends a StartViewChange message with a proposed viewNum and the current node id to a replica at port.
func SendStartViewChange(port, viewNum, id int) {
	log.Printf("replica %v sends StartViewChange %v to replica with port %v\n", *flags.Id, viewNum, port)
	p := strconv.Itoa(port)
	// TODO: Make it more efficient by caching client for each port.
	client, err := rpc.DialHTTP("tcp", "localhost:"+p)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	req := vrrpc.StartViewChangeArgs{
		ViewNum: viewNum,
		Id:      id,
	}
	var resp vrrpc.StartViewChangeResp
	_ = client.Go("ViewChangeRPC.StartViewChange", req, &resp, nil)
}

func sendDoViewChange(viewNum, currentViewNum, opNum, commitNum, id int) {
	newPrimaryId := viewNum % len(globals.AllPorts)
	log.Printf("sending DoViewChange to new primary %v\n", newPrimaryId)
	newPrimaryPort := globals.AllPorts[newPrimaryId]
	req := vrrpc.DoViewChangeArgs{
		ViewNum:             viewNum,
		Log:                 globals.OpLog.Requests,
		LatestNormalViewNum: currentViewNum,
		OpNum:               opNum,
		CommitNum:           commitNum,
		Id:                  *flags.Id,
	}
	var resp vrrpc.DoViewChangeResp

	if newPrimaryId == *flags.Id {
		log.Printf("new primary %v is the current node.\n", newPrimaryId)
		// call runDoViewChange() directly.
		// TODO: maybe as a Go routine?
		runDoViewChange(&req, &resp)
		return
	}
	// call DoViewChange() RPC.
	p := strconv.Itoa(newPrimaryPort)
	client, err := rpc.DialHTTP("tcp", "localhost:"+p)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	_ = client.Go("ViewChangeRPC.DoViewChange", req, &resp, nil)
}

func sendStartView(port int) {
	log.Printf("new primary %v sending StartView to replica at %v", *flags.Id, port)
	req := vrrpc.StartViewArgs{
		ViewNum:   globals.ViewNum,
		Log:       globals.OpLog.Requests,
		OpNum:     globals.OpNum,
		CommitNum: globals.CommitNum,
	}
	var resp vrrpc.StartViewResp
	p := strconv.Itoa(port)
	client, err := rpc.DialHTTP("tcp", "localhost:"+p)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	_ = client.Go("ViewChangeRPC.StartView", req, &resp, nil)
}
