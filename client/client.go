// client defines the client side logic.
package client

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"

	"github.com/BoolLi/vrgo/flags"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

var requestNum = flag.Int("request_num", 0, "request number")

// RunClient runs the client code.
func RunClient() {
	p := strconv.Itoa(*flags.Port)
	client, err := rpc.DialHTTP("tcp", "localhost:"+p)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter text: \n")
		text, _ := reader.ReadString('\n')
		req := vrrpc.Request{
			Op: vrrpc.Operation{
				Message: text,
			},
			ClientId:   *flags.Id,
			RequestNum: *requestNum,
		}

		var resp vrrpc.Response
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
