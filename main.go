package main

import (
	"flag"
	"fmt"

	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/server"
	"github.com/BoolLi/vrgo/table"

	cache "github.com/patrickmn/go-cache"
)

var mode = flag.String("mode", "", "'server' mode or 'client' mode")

func main() {
	flag.Parse()

	server.Init(table.New(cache.NoExpiration, cache.NoExpiration))

	switch *mode {
	case "server":
		server.Register(new(server.Basic))
		server.Register(new(server.VrgoRPC))
		go server.Serve()
		for {
		}
	case "client":
		client.RunClient()
	default:
		fmt.Printf("mode %v can only be 'server' or 'client'\n", *mode)
	}
}
