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
func StartVrgo(mode string) {
	ctx := context.Background()

	clientTable := table.New(cache.NoExpiration, cache.NoExpiration)
	opRequestLog := oplog.New()

	for {
		switch mode {
		case "primary":
			ctxCancel, cancel := context.WithCancel(ctx)
			startPrimary(ctxCancel, opRequestLog, clientTable)
			select {
			case <-view.StartViewChangeChan:
				cancel()
				mode = "vc-recv"
			}
			//for {
			//}
		case "backup":
			ctxCancel, cancel := context.WithCancel(ctx)
			vt := time.NewTimer(5 * time.Second)
			startBackup(ctxCancel, opRequestLog, clientTable, vt)
			select {
			case <-vt.C:
				// TODO: Think about how to stop backup from handling BackupService.
				log.Printf("view timer expires; backup %v starts view change.", *flags.Id)
				cancel()
				mode = "vc-init"
			case <-view.StartViewChangeChan:
				cancel()
				mode = "vc-recv"
			}
			//for {
			//}
		case "vc-init":
			// send SVC to other replicas
			log.Printf("entered vc-init mode")
			view.CurrentProposedViewNum.Lock()
			view.CurrentProposedViewNum.Value = globals.ViewNum + 1
			view.CurrentProposedViewNum.Unlock()
			for _, p := range view.AllOtherPorts() {
				view.SendStartViewChange(p, view.CurrentProposedViewNum.Value, *flags.Id)
			}
			mode = "vc-recv"
		case "vc-recv":
			log.Printf("entered vc-recv mode")
			select {}
			// waits until mode is set to "primary" or "backup"
		}
	}
}

func startPrimary(ctx context.Context, opRequestLog *oplog.OpRequestLog, clientTable *table.ClientTable) {
	if err := primary.Init(ctx, opRequestLog, clientTable); err != nil {
		log.Fatalf("failed to initialize primary: %v", err)
	}
}

func startBackup(ctx context.Context, opRequestLog *oplog.OpRequestLog, clientTable *table.ClientTable, vt *time.Timer) {
	if err := backup.Init(ctx, opRequestLog, clientTable, vt); err != nil {
		log.Fatalf("failed to initialize backup: %v", err)
	}
}
