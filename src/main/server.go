package main

import (
	"CrossRBC/src/communication/receiver"
	"flag"
	"log"
	"os"
)

const (
	helpTextServer = `
Main function for servers. Start a replica. 
server [ReplicaID]
`
)

func main() {
	helpPtr := flag.Bool("help", false, helpTextServer)
	flag.Parse()

	if *helpPtr || len(os.Args) < 2 {
		log.Printf(helpTextServer)
		return
	}

	id := "0"
	if len(os.Args) > 1 {
		id = os.Args[1]
	}

	log.Printf("**Starting replica %s**\n", id)
	receiver.StartReceiver(id, true)
}
