package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/BoolLi/vrgo/client"
	"github.com/BoolLi/vrgo/globals"
	"github.com/BoolLi/vrgo/monitor"
)

func main() {
	log.SetOutput(os.Stdout)
	rand.Seed(time.Now().Unix())

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
