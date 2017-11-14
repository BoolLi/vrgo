package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/primary"
	"github.com/BoolLi/vrgo/server"
	"github.com/BoolLi/vrgo/table"

	cache "github.com/patrickmn/go-cache"
)

var mode = flag.String("mode", "", "'server' mode or 'client' mode")

func main() {
	flag.Parse()

	switch *mode {
	case "server":
		server.Init(table.New(cache.NoExpiration, cache.NoExpiration))
		server.Register(new(server.Basic))
		server.Register(new(server.VrgoRPC))
		go server.Serve()
		primary.New()

		time.Sleep(10 * time.Second)
		primary.DummyConsumeIncomingReq()

		for {
		}
	case "client":
		client.RunClient()
	default:
		fmt.Printf("mode %v can only be 'server' or 'client'\n", *mode)
	}
}
