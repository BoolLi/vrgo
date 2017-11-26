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

var (
	startViewChangeReceived mutexInt
	currentProposedViewNum  mutexInt

	subquorum = 1
)

func StartViewChange(args *vrrpc.StartViewChangeArgs, resp *vrrpc.StartViewChangeResp) error {
	if args.ViewNum <= globals.ViewNum {
		// If the proposed view num is smaller than the current view num, do nothing.
		return nil
	}

	// Lock the view change states to prevent race conditions across multiple threads.
	currentProposedViewNum.Lock()
	startViewChangeReceived.Lock()
	defer currentProposedViewNum.Unlock()
	defer startViewChangeReceived.Unlock()

	// If somebody else proposes a view with a larger view num, we should advocate that instead of the old one.
	if args.ViewNum > currentProposedViewNum.Value {
		startViewChangeReceived.Value = 0
		currentProposedViewNum.Value = args.ViewNum

		// Send StartViewChange to all other nodes.
		for _, p := range allOtherPorts() {
			sendStartViewChange(p, args.ViewNum, *flags.Id)
		}
	}
	startViewChangeReceived.Value += 1

	if startViewChangeReceived.Value > subquorum {
		// Sends DoViewChange to new primary.
		// TODO: Should we do this in a separate thread?
	}
	return nil
}

func allOtherPorts() []int {
	var ps []int
	for _, p := range globals.AllPorts {
		if p != *flags.Port {
			ps = append(ps, p)
		}
	}
	return ps
}

func sendStartViewChange(port, viewNum, id int) {
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
	//newPrimaryId := viewNum % len(globals.AllPorts)

}
