package main

import (
	"flag"
	"fmt"

	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/server"
)

var mode = flag.String("mode", "", "'server' mode or 'client' mode")

func main() {
	flag.Parse()
	switch *mode {
	case "server":
		server.Register(new(server.Basic))
		go server.Serve()
		for {
		}
	case "client":
		client.RunClient()
	default:
		fmt.Printf("mode %v can only be 'server' or 'client'\n", *mode)
	}
}
