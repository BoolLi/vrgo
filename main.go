package main

import (
	"log"
	"os"

	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/monitor"
)

func main() {
	log.SetOutput(os.Stdout)

	// TODO: Make a cancellable context.
	switch globals.Mode {
	case "primary":
		monitor.StartVrgo()
	case "backup":
		monitor.StartVrgo()
	default:
		client.RunClient()
	}
}
