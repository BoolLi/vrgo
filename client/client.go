package client

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"

	"github.com/BoolLi/vrgo/server"
)

var serverPort = flag.Int("server_port", 1234, "server port")

// RunClient runs the client code.
func RunClient() {
	p := strconv.Itoa(*serverPort)
	client, err := rpc.DialHTTP("tcp", "localhost:"+p)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter text: \n")
		text, _ := reader.ReadString('\n')

		args := &server.EchoArgs{Message: text}
		var reply server.EchoResp
		call := client.Go("Basic.DelayedEcho", args, &reply, nil)
		go printReply(call)
	}
}

func printReply(call *rpc.Call) {
	resp := <-call.Done
	fmt.Printf("echo response: %v", resp.Reply.(*server.EchoResp).Message)
}
