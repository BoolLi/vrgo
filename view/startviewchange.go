package view

import (
	"github.com/BoolLi/vrgo/globals"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

func StartViewChange(args *vrrpc.StartViewChangeArgs, resp *vrrpc.StartViewChangeResp) error {
	if args.ViewNum > globals.ViewNum {
		// Send StartViewChange to all other nodes.
	}
}
