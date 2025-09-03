package main

import (
	//"context"

	"flag"
	"log"
	"os"

	"CrossRBC/src/client"
	"CrossRBC/src/logging"
	"CrossRBC/src/utils"
	"fmt"
	"strconv"
	"sync"
)

const (
	defaultMsg      = "abcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcdefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesab" //len of request = 250 in msgpack serialization
	defaultMsg2     = "abcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareA"
	defaultMsg3 = "abcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEV F ARsfwfw ssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfssseAERGAEVFARsfwfsssabcdefghiaerfvgqretveqatvgqareAERGAEVFARsfwfsssabcdefffghiaerfvgqretveqatvgqaresssefAbcddefghiaerfvgqretveqatvgqaresssefAevgqaresssefAeesababcdefghiaerfvg"
	
	helpText_Client = `
    Main function for Client. Start a client and do write or read request. 

	[TypeOfRequest]:
    	0 - write
		1 - write batch

    client [id number] [TypeOfRequest] [batch size/number of requests] [message size]

    Examples:
    	1. send one cross message (default: 250bytes)
			./client 100 0 1 1
    	//start a client with ID = 100, and broadcast [1] request message (epoch is 0, length: [250] bytes) 
		2 send cross message with larger size:1024 bytes
			./client 100 0 1 2
		//start a client with ID = 100, and broadcast [1] request message (epoch is 0, length: [1024] bytes) 
		3. send multiple cross messages
			./client 100 0 3 2
		//start a client with ID = 100, and broadcast 3 request messages (epoch is 0..2, length:[1024] bytes)
    `

	numMessagesPerClient = 3
	numServers           = 4
)

var lockwo sync.Mutex
var freq int

func main() {
	helpPtr := flag.Bool("help", false, helpText_Client)

	flag.Parse()

	//init
	id := "0"
	rtype := 0 //default: write
	numReq := 1
	msg := make([]byte, 250)
	msgSize := 250

	var err error

	if *helpPtr || len(os.Args) < 3 {
		log.Printf(helpText_Client)
		return
	}

	//id
	id = os.Args[1]
	logging.SetID(id)

	//identify type of write and msgSize
	rtype, err = strconv.Atoi(os.Args[2])
	if err != nil {
		log.Printf("Please enter a valid integer (type of request):0,1,2")
		return
	}

	// if os.Args greater than 4, then read it as msgSize
	if len(os.Args) > 3 {
		msgSize, err = strconv.Atoi(os.Args[4])
		if err != nil || msgSize < 1 {
			log.Printf("Please enter a valid integer (message size)")
			return
		}
		switch msgSize {
		case 1:
			msg = utils.StringToBytes(defaultMsg)
		case 2:
			msg = utils.StringToBytes(defaultMsg2)
		case 3:
			msg = utils.StringToBytes(defaultMsg3)
		}

		log.Printf("message size: %v", len(msg))
	}

	//identify number of Requests/batch size
	numReq, err = strconv.Atoi(os.Args[3])
	if err != nil {
		log.Printf("Please enter a valid integer (number of requests or topic number)")
		return
	}
	if numReq < 1 {
		log.Fatalf("Please enter a valid number for numReq")
	}

	p2 := fmt.Sprintf("Client is ready: ID#%v, writeType:%v, numOfReq:%v, messageSize:%v", id, rtype, numReq, len(msg))
	logging.PrintLog(true, logging.NormalLog, p2)

	StartClient(id, numReq, rtype, msg)
}

func StartClient(cid string, numReq int, wtype int, msg []byte) {
	lockwo.Lock()
	client.StartClient(cid, true)
	lockwo.Unlock()

	if wtype == 0 {
		client.SendWriteRequest(msg, numReq)
	} else if wtype == 1 {
		for i := 0; i < freq; i++ {
			client.SendBatchRequest(msg, numReq, i)
			p := fmt.Sprintf("Client#%v, Sending batch #%v request, frequency#%v", cid, numReq, freq)
			logging.PrintLog(true, logging.NormalLog, p)
		}
	}
}
