// client defines the client side logic.
package client

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/BoolLi/vrgo/flags"
	"github.com/BoolLi/vrgo/globals"

	vrrpc "github.com/BoolLi/vrgo/rpc"
)

var (
	requestNum = flag.Int("request_num", 0, "request number")
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

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		req := vrrpc.Request{
			Op: vrrpc.Operation{
				Message: text,
			},
			ClientId:   *flags.Id,
			RequestNum: *requestNum,
		}

		p := strconv.Itoa(globals.Port)
		rpcClient, err := globals.GetOrCreateClient("localhost:" + p)
		if err != nil {
			log.Fatal("dialing:", err)
		}
		curId, err := currentPrimaryId()
		if err != nil {
			log.Fatalf("failed to get current primary id")
		}
		log.Printf("sending to replica %v", curId)

		var resp vrrpc.Response

		ch := make(chan error)

		go func() {
			ch <- rpcClient.Call("VrgoRPC.Execute", req, &resp)
		}()
		select {
		case err := <-ch:
			if err != nil {
				log.Printf("failed to call VrgoRPC: %v", err)
			}
			processResp(&resp)
			*requestNum = *requestNum + 1
		case <-time.After(5 * time.Second):
			log.Printf("timed out trying to connect to replica %v", curId)
		}
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

func processResp(resp *vrrpc.Response) {
	curId, err := currentPrimaryId()
	if err != nil {
		log.Fatalf("Failed to look up primary ID: %v", err)
	}
	log.Printf("current view num: %v", resp.ViewNum)

	if errMsg := resp.Err; errMsg != "" {
		if errMsg == "not primary" {
			newId := resp.ViewNum % len(globals.AllPorts)
			log.Printf("Primary %v => %v", curId, newId)
			globals.Port = globals.AllPorts[newId]
		} else if errMsg == "view change" {
			log.Printf("currently under view change")
		} else {
			log.Printf("got error message but it was not rognized: %v", errMsg)
		}
		return
	}

	fmt.Printf("Vrgo response: %v\n", resp.OpResult.Message)
}
