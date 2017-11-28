package view

import (
	"log"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

// ViewChangeRPC implements the ViewService interface.
type ViewChangeRPC int

type mutexDoViewChangeArgs struct {
	sync.Mutex
	Args []*vrrpc.DoViewChangeArgs
}

var (
	// A buffered channel to signal the monitor thread to change to view change mode.
	// Note: we use a buffered channel here because multiple threads running StartViewChange() could be
	// sending signals to the channel at the same time, but the monitor thread can only consume one of them.
	// Having a buffered channel ensures that while only the first signal is consumed, the rest of the threads do not block.
	StartViewChangeChan chan int = make(chan int, len(globals.AllPorts))

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

	// Send a signal to the monitor to change to the view change mode.
	StartViewChangeChan <- 1

	if args.ViewNum <= globals.ViewNum {
		// If the proposed view num is smaller than the current view num, do nothing.
		log.Printf("exiting because proposed view num %v is no larger than current view num %v", args.ViewNum, globals.ViewNum)
		return nil
	}

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
	log.Printf("new primary %v got DoViewChange message %+v", *flags.Id, args)
	return runDoViewChange(args, resp)
}

// StartView handles the StartView RPC.
func (v *ViewChangeRPC) StartView(args *vrrpc.StartViewArgs, resp *vrrpc.StartViewResp) error {
	return nil
}

func runDoViewChange(args *vrrpc.DoViewChangeArgs, resp *vrrpc.DoViewChangeResp) error {
	doViewChangeArgsReceived.Lock()
	defer doViewChangeArgsReceived.Unlock()

	// Only send a StartView message when enough DoViewChange messages are received.
	if len(doViewChangeArgsReceived.Args) < subquorum {
		doViewChangeArgsReceived.Args = append(doViewChangeArgsReceived.Args, args)
		log.Printf("new primary %v received %+v, but it's only received %v DoViewChange messages so far\n", *flags.Id, args, len(doViewChangeArgsReceived.Args))
		return nil
	}

	// Make sure all the DoViewChange messages have the same view num.
	if !sameViewNums() {
		log.Fatalf("new primary %v received DoViewChange messages with different view nums: %+v\n", *flags.Id, doViewChangeArgsReceived)
	}

	// TODO: Change states and send StartView.
	log.Printf("replica %v becomes the new primary", *flags.Id)
	return nil
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

// TODO: Need to add log too.
func sendDoViewChange(viewNum, currentViewNum, opNum, commitNum, id int) {
	newPrimaryId := viewNum % len(globals.AllPorts)
	log.Printf("sending DoViewChange to new primary %v\n", newPrimaryId)
	newPrimaryPort := globals.AllPorts[newPrimaryId]
	req := vrrpc.DoViewChangeArgs{
		ViewNum:             viewNum,
		Log:                 nil,
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
