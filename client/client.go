// client defines the client side logic.
package client

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"sync"

	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

var (
	requestNum = flag.Int("request_num", 0, "request number")

	client    *rpc.Client
	clientMux sync.Mutex
)

// RunClient runs the client code.
func RunClient() {
	csvFile, err := os.Open(*flags.ConfigPath)
	if err != nil {
		log.Fatalf("failed to open csv file: %v", err)
	}
	r := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read line from config %v: %v", csvFile, err)
		}

		port, err := strconv.Atoi(line[2])
		if err != nil {
			log.Fatalf("failed to convert port to int: %v", err)
		}

		if line[0] == "primary" {
			globals.Port = port
			break
		}
	}

	p := strconv.Itoa(globals.Port)
	c, err := globals.GetOrCreateClient("localhost:" + p)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	client = c

	// If we are in automated mode (indicated by negative client id), send
	// predefined inputs automatically.
	if *flags.Id < 0 {
		for _, element := range ClientInput {
			req := vrrpc.Request{
				Op: vrrpc.Operation{
					Message: element,
				},
				ClientId:   *flags.Id,
				RequestNum: *requestNum,
			}
			var resp vrrpc.Response
			_ = client.Go("VrgoRPC.Execute", req, &resp, nil)
			*requestNum += 1
		}
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
		clientMux.Lock()
		call := client.Go("VrgoRPC.Execute", req, &resp, nil)
		clientMux.Unlock()

		go printResp(call)

		// TODO: Need to find a way to increment requestNum but also allow users to send request with same requestNum.
		*requestNum = *requestNum + 1
	}
}

func currentPrimaryId() (int, error) {
	for id, p := range globals.AllPorts {
		if p == globals.Port {
			return id, nil
		}
	}
	return 0, fmt.Errorf("cannot find id corresponding to port %v", globals.Port)
}

func printResp(call *rpc.Call) {
	resp := <-call.Done
	curId, err := currentPrimaryId()
	if err != nil {
		log.Fatalf("Failed to look up primary ID: %v", err)
	}
	newId := resp.Reply.(*vrrpc.Response).ViewNum % len(globals.AllPorts)

	log.Printf("current view num: %v", resp.Reply.(*vrrpc.Response).ViewNum)
	if curId == newId {
		fmt.Printf("Vrgo response: %v\n", resp.Reply.(*vrrpc.Response).OpResult.Message)
		return
	}

	log.Printf("Primary %v => %v", curId, newId)
	clientMux.Lock()
	defer clientMux.Unlock()
	client.Close()

	globals.Port = globals.AllPorts[newId]
	p := strconv.Itoa(globals.Port)
	c, err := globals.GetOrCreateClient("localhost:" + p)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	client = c
}
