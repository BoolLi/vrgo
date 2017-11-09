package main

import (
	"basic"
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

var port = flag.Int("port", 1234, "server port")

func main() {
	b := new(basic.Basic)
	rpc.Register(b)
	rpc.HandleHTTP()

	p := strconv.Itoa(*port)

	l, e := net.Listen("tcp", ":"+p)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	for {
	}
}
