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
var clientId = flag.Int("client_id", 0, "ID of the client")
var requestNum = flag.Int("request_num", 0, "request number")

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
		req := server.Request{
			Op: server.Operation{
				Message: text,
			},
			ClientId:   *clientId,
			RequestNum: *requestNum,
		}
		log.Printf("\nuser request: %v\n", req)

		var resp server.Response
		call := client.Go("VrgoRPC.Execute", req, &resp, nil)
		go printResp(call)

		// TODO: Need to find a way to increment requestNum but also allow users to send request with same requestNum.
		*requestNum = *requestNum + 1
	}
}

func printResp(call *rpc.Call) {
	resp := <-call.Done
	fmt.Printf("Vrgo response: %v\n", resp)
}
