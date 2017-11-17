package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BoolLi/vrgo/backup"
	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/table"

	cache "github.com/patrickmn/go-cache"
)

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	// TODO: Make a cancellable context.
	ctx := context.Background()
	switch *flags.Mode {
	case "server":
		clientTable := table.New(cache.NoExpiration, cache.NoExpiration)
		if err := primary.Init(ctx, oplog.New(), clientTable); err != nil {
			log.Fatalf("failed to initialize primary: %v", err)
		}
		for {
		}
	case "client":
		client.RunClient()
	case "backup":
		clientTable := table.New(cache.NoExpiration, cache.NoExpiration)
		if err := backup.Init(ctx, oplog.New(), clientTable); err != nil {
			log.Fatalf("failed to initialize primary: %v", err)
		}
		backup.Register(new(backup.BackupReply))
		go backup.Serve()
		for {
		}
	default:
		fmt.Printf("mode %v can only be 'server' or 'client'\n", *flags.Mode)
	}
}
