package monitor

import (
	"context"
	"log"
	"time"

	"github.com/BoolLi/vrgo/backup"
	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/table"
	"github.com/BoolLi/vrgo/view"

	cache "github.com/patrickmn/go-cache"
)

// Start a VR process as <mode>. Mode can be "primary", "backup", and "viewchange".
// Depending on different conditions, a node can switch between different modes, which is managed by this function.
func StartVrgo(mode string) {
	ctx := context.Background()

	globals.ClientTable = table.New(cache.NoExpiration, cache.NoExpiration)
	globals.OpLog = oplog.New()

	for {
		switch mode {
		case "primary":
			// TODO: It's probably not enough to just clear the states at the start of primary and backup.
			view.ClearViewChangeStates()
			ctxCancel, cancel := context.WithCancel(ctx)
			startPrimary(ctxCancel)

			select {
			case <-view.StartViewChangeChan:
				cancel()
				mode = "viewchange"
			}
		case "backup":
			view.ClearViewChangeStates()
			ctxCancel, cancel := context.WithCancel(ctx)
			vt := time.NewTimer(5 * time.Second)
			startBackup(ctxCancel, vt)

			select {
			case <-vt.C:
				// TODO: Think about how to stop backup from handling BackupService.
				log.Printf("view timer expires; backup %v starts view change.", *flags.Id)
				cancel()
				mode = "viewchange-init"
			case <-view.StartViewChangeChan:
				cancel()
				mode = "viewchange"
			}
		case "viewchange-init":
			view.InitiateStartViewChange()
			mode = "viewchange"
		case "viewchange":
			log.Printf("entered viewchange mode")
			select {}
			// waits until mode is set to "primary" or "backup"
		}
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
