package monitor

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"github.com/BoolLi/vrgo/backup"
	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/recovery"
	"github.com/BoolLi/vrgo/table"
	"github.com/BoolLi/vrgo/view"

	cache "github.com/patrickmn/go-cache"
)

var (
	backupTimeout     = 5 * time.Second
	viewchangeTimeout = 10 * time.Second
)

// Start a VR process.
// Depending on different conditions, a node can switch between different modes, which is managed by this function.
func StartVrgo() {
	ctx := context.Background()

	recovery.RegisterRecovery(new(recovery.RecoveryRPC))

	crashSig := fmt.Sprintf("./crash-%v", *flags.Id)
	if crashed(crashSig) {
		globals.Log("StartVrgo", "crashed before; entering recovery mode")
		globals.Mode = "recovery"
	} else {
		globals.Log("StartVrgo", "hasn't crashed before")
	}

	writeCrashSignal(crashSig)

	globals.ClientTable = table.New(cache.NoExpiration, cache.NoExpiration)
	globals.OpLog = oplog.New()

	// Serve starts an HTTP server to handle RPC requests.
	go func() {
		// Serve starts an HTTP server to handle RPC requests.
		rpc.HandleHTTP()
		l, err := net.Listen("tcp", fmt.Sprintf(":%v", globals.Port))
		if err != nil {
			log.Fatalf("failed to listen on port %v: %v", globals.Port, err)
		}
		http.Serve(l, nil)
	}()

	for {
		switch globals.Mode {
		case "primary":
			globals.Log("StartVrgo", "entered primary mode")
			// TODO: It's probably not enough to just clear the states at the start of primary and backup.
			view.ClearViewChangeStates(true)
			ctxCancel, cancel := context.WithCancel(ctx)
			globals.CtxCancel = ctxCancel
			startPrimary(ctxCancel)

			select {
			case <-view.StartViewChangeChan:
				cancel()
				globals.Mode = "viewchange"
			}
		case "backup":
			globals.Log("StartVrgo", "entered backup mode")
			view.ClearViewChangeStates(true)
			ctxCancel, cancel := context.WithCancel(ctx)
			vt := time.NewTimer(backupTimeout)
			startBackup(ctxCancel, vt)

			select {
			case <-vt.C:
				// TODO: Think about how to stop backup from handling BackupService.
				globals.Log("StartVrgo", "view timer expires")
				cancel()
				globals.Mode = "viewchange-init"
			case <-view.StartViewChangeChan:
				cancel()
				globals.Mode = "viewchange"
			}
		case "viewchange-init":
			globals.Log("StartVrgo", "entered viewchange-init mode")
			view.InitiateStartViewChange()
			globals.Mode = "viewchange"
		case "viewchange":
			globals.Log("StartVrgo", "entered viewchange mode")
			vt := time.NewTimer(viewchangeTimeout)
			select {
			case newMode := <-view.ViewChangeDone:
				globals.Log("StartVrgo", "switched from %v to %v", globals.Mode, newMode)
				globals.Mode = newMode
			case <-vt.C:
				view.ClearViewChangeStates(false)
				globals.Mode = "viewchange-init"
			}
			// waits until mode is set to "primary" or "backup"
		case "recovery":

		}
	}
	// TODO: Delete crashSig before exiting.
}

func crashed(crashSig string) bool {
	_, err := ioutil.ReadFile(crashSig)
	if err != nil {
		return false
	}
	return true
}

// writeCrashSignal creates a file on disk to indicate crashing.
// The file is created when the replica starts up and deleted when the replica teminates
// normally. The file remains on disk if the replica crashed and it will check the
// existense of this file when it starts up.
func writeCrashSignal(crashSig string) {
	if err := ioutil.WriteFile(crashSig, []byte{}, 0644); err != nil {
		log.Fatalf("failed to write crash signal: %v", err)
	}
}

func startPrimary(ctx context.Context) {
	if err := primary.Init(ctx); err != nil {
		log.Fatalf("failed to initialize primary: %v", err)
	}
}

func startBackup(ctx context.Context, vt *time.Timer) {
	if err := backup.Init(ctx, vt); err != nil {
		log.Fatalf("failed to initialize backup: %v", err)
	}
}
