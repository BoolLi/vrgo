package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/BoolLi/vrgo/backup"
	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/oplog"
	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/server"
	"github.com/BoolLi/vrgo/table"

	cache "github.com/patrickmn/go-cache"
)

var mode = flag.String("mode", "", "'server', 'client', or 'backup' mode")

func main() {
	flag.Parse()

	switch *mode {
	case "server":
		// Create server.
		clientTable := table.New(cache.NoExpiration, cache.NoExpiration)
		if err := server.Init(clientTable); err != nil {
			log.Fatalf("failed to initialize server: %v", err)
		}
		server.Register(new(server.VrgoRPC))
		go server.Serve()

		// Create primary.
		if err := primary.Init(oplog.New(), clientTable); err != nil {
			log.Fatalf("failed to initialize primary: %v", err)
		}
		for {
		}
	case "client":
		client.RunClient()
	case "backup":
		backup.RunBackup()
	default:
		fmt.Printf("mode %v can only be 'server' or 'client'\n", *mode)
	}
}
