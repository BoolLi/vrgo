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

func StartVrgo(entry string) {
	ctx := context.Background()
	switch entry {
	case "primary":
		ctxCancel, _ := context.WithCancel(ctx)
		startPrimary(ctxCancel)
		for {
		}
	case "backup":
		ctxCancel, cancel := context.WithCancel(ctx)
		vt := time.NewTimer(5 * time.Second)
		startBackup(ctxCancel, vt)
		select {
		case <-vt.C:
			log.Printf("view timer expires")
			cancel()
			// Start change.
		}
		for {
		}
	}
}

func startPrimary(ctx context.Context) {
	clientTable := table.New(cache.NoExpiration, cache.NoExpiration)
	if err := primary.Init(ctx, oplog.New(), clientTable); err != nil {
		log.Fatalf("failed to initialize primary: %v", err)
	}
}

func startBackup(ctx context.Context, vt *time.Timer) {
	clientTable := table.New(cache.NoExpiration, cache.NoExpiration)
	if err := backup.Init(ctx, oplog.New(), clientTable, vt); err != nil {
		log.Fatalf("failed to initialize primary: %v", err)
	}
	backup.Register(new(backup.BackupReply))
	go backup.Serve()
}
