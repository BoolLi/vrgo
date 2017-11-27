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

type ViewChangeRPC int
type mutexInt struct {
	sync.Mutex
	Value int
}
type mutexDoViewChangeArgs struct {
	sync.Mutex
	Args []*vrrpc.DoViewChangeArgs
}

var (
	startViewChangeReceived mutexInt
	CurrentProposedViewNum  mutexInt

	doViewChangeArgsReceived mutexDoViewChangeArgs

	subquorum           = 2
	StartViewChangeChan chan int
)

func init() {
	StartViewChangeChan = make(chan int)
}

func (v *ViewChangeRPC) StartViewChange(args *vrrpc.StartViewChangeArgs, resp *vrrpc.StartViewChangeResp) error {
	log.Printf("replica %v receives StartViewChange with %v from %v.\n", *flags.Id, args.ViewNum, args.Id)
	StartViewChangeChan <- 1

	if args.ViewNum <= globals.ViewNum {
		// If the proposed view num is smaller than the current view num, do nothing.
		log.Printf("exiting because proposed view num %v is no larger than current view num %v", args.ViewNum, globals.ViewNum)
		return nil
	}

	log.Printf("before lock")
	// Lock the view change states to prevent race conditions across multiple threads.
	CurrentProposedViewNum.Lock()
	log.Printf("after first lock")
	startViewChangeReceived.Lock()
	log.Printf("after 2nd lock")
	defer CurrentProposedViewNum.Unlock()
	defer startViewChangeReceived.Unlock()

	// If somebody else proposes a view with a larger view num, we should advocate that instead of the old one.
	if args.ViewNum > CurrentProposedViewNum.Value {
		log.Printf("the view num %v proposed is larger than the current proposed view num %v\n", args.ViewNum, CurrentProposedViewNum.Value)
		startViewChangeReceived.Value = 0
		CurrentProposedViewNum.Value = args.ViewNum

		// Send StartViewChange to all other nodes.
		for _, p := range AllOtherPorts() {
			SendStartViewChange(p, args.ViewNum, *flags.Id)
		}
	}
	startViewChangeReceived.Value += 1
	log.Printf("replica %v increments startViewChangeReceived to %v\n", *flags.Id, startViewChangeReceived.Value)

	if startViewChangeReceived.Value > subquorum {
		log.Printf("replica %v got more than %v StartViewChange messages\n", *flags.Id, subquorum)
		// Sends DoViewChange to new primary.
		// TODO: Should we do this in a separate thread?
		sendDoViewChange(CurrentProposedViewNum.Value, globals.ViewNum, globals.OpNum, globals.CommitNum, *flags.Id)
		// TODO: Clear startViewChangeReceived, CurrentProposedViewNum, and doViewChangeArgsReceived somewhere.
	}
	log.Printf("replica %v exiting StartViewChange()\n", *flags.Id)
	return nil
}

func (v *ViewChangeRPC) DoViewChange(args *vrrpc.DoViewChangeArgs, resp *vrrpc.DoViewChangeResp) error {
	return runDoViewChange(args, resp)
}

func (v *ViewChangeRPC) StartView(args *vrrpc.StartViewArgs, resp *vrrpc.StartViewResp) error {
	return nil
}

func runDoViewChange(args *vrrpc.DoViewChangeArgs, resp *vrrpc.DoViewChangeResp) error {
	doViewChangeArgsReceived.Lock()
	defer doViewChangeArgsReceived.Unlock()

	if len(doViewChangeArgsReceived.Args) < subquorum {
		doViewChangeArgsReceived.Args = append(doViewChangeArgsReceived.Args, args)
		log.Printf("new primary %v received %+v, but it's only received %v DoViewChange messages so far\n", *flags.Id, args, len(doViewChangeArgsReceived.Args))
		return nil
	}

	if !sameViewNums() {
		log.Fatalf("new primary %v received DoViewChange messages with different view nums: %+v\n", *flags.Id, doViewChangeArgsReceived)
	}

	// TODO: Change states and send StartView.
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

func AllOtherPorts() []int {
	var ps []int
	for _, p := range globals.AllPorts {
		if p != *flags.Port {
			ps = append(ps, p)
		}
	}
	return ps
}

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
