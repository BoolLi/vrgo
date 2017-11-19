package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/monitor"
)

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	// TODO: Make a cancellable context.
	switch *flags.Mode {
	case "primary":
		monitor.StartVrgo("primary")
	case "backup":
		monitor.StartVrgo("backup")
	case "client":
		client.RunClient()
	default:
		fmt.Printf("mode %v can only be 'server' or 'client'\n", *flags.Mode)
	}
}
