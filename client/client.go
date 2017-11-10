package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"

	"github.com/BoolLi/vrgo/basic"
)

var port = flag.Int("port", 1234, "server port")

func main() {
	p := strconv.Itoa(*port)
	client, err := rpc.DialHTTP("tcp", "localhost:"+p)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter text: \n")
		text, _ := reader.ReadString('\n')

		args := &basic.EchoArgs{Message: text}
		var reply basic.EchoResp
		err = client.Call("Basic.Echo", args, &reply)
		if err != nil {
			log.Fatal("echo error:", err)
		}
		fmt.Printf("Echo response: %v\n", reply.Message)

	}

}
