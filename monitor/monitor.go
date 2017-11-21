package monitor

import (
	"context"
	"log"
	"time"

	"github.com/BoolLi/vrgo/backup"
	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/table"

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
			ctxCancel, _ := context.WithCancel(ctx)
			startPrimary(ctxCancel, opRequestLog, clientTable)
			for {
			}
		case "backup":
			ctxCancel, cancel := context.WithCancel(ctx)
			vt := time.NewTimer(5 * time.Second)
			startBackup(ctxCancel, opRequestLog, clientTable, vt)
			select {
			case <-vt.C:
				// TODO: Think about how to stop backup from handling BackupService.
				log.Printf("view timer expires")
				cancel()
				// Start change.
			}
			for {
			}
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
