package clientsender

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"CrossRBC/src/communication"
	"CrossRBC/src/config"
	"CrossRBC/src/cryptolib"
	logging "CrossRBC/src/logging"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/utils"

	"google.golang.org/grpc"
)

var clientTimer int
var verbose bool
var dialOpt []grpc.DialOption // tls dial option
var connections communication.AddrConnMap

var wg sync.WaitGroup
var id int64
var err error

func BuildConnection(ctx context.Context, nid string, address string) bool {
	p := fmt.Sprintf("[Client Sender (clientsender)] building a connection with %v", nid)
	logging.PrintLog(verbose, logging.NormalLog, p)

	conn, err := grpc.DialContext(ctx, address, dialOpt...)
	if err != nil {
		p := fmt.Sprintf("[Client Sender(clientsender)] failed to bulid a connection with %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return false
	}

	p1 := fmt.Sprintf("[Client Sender(clientsender)] successfully built a connection with %v, address:%v", nid, address)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	c := pb.NewSendClient(conn)

	connections.Insert(address, c)

	return true
}

// confirm process op - serialized rep msg
func SendRequest(rtype pb.MessageType, op []byte, address string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(clientTimer)*time.Millisecond)
	defer cancel()

	nid := config.FetchReplicaID(address)
	//v, err := utils.StringToInt64(nid)

	if config.SplitPorts() {
		address = communication.UpdateAddress(address)
	}

	c, built := connections.Get(address)
	if !built || c == nil {
		suc := BuildConnection(ctx, nid, address)
		if !suc {

			p := fmt.Sprintf("[Client Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			clientTimer = clientTimer * 2

			//CatchSendRequestError(v, op, rtype, t1)
			wg.Done()
			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	var r *pb.RawMessage
	r, err = c.SendRequest(ctx, &pb.Request{Type: rtype, Request: op})

	if err != nil {
		p := fmt.Sprintf("[Client Sender Error] could not get reply from node %s, %v", nid, err)
		logging.PrintLog(true, logging.ErrorLog, p)
		//CatchSendRequestError(v, op, rtype, t1)
		connections.Insert(address, nil)
		wg.Done()
		return
	}

	re := string(r.GetMsg())
	p1 := fmt.Sprintf("[CROSS] got a reply: %s", re)
	logging.PrintLog(true, logging.NormalLog, p1)

	wg.Done()
}

// confirm process op - serialized rep msg
// func SendGRequest(rtype pb.MessageType, op []byte, address string) {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(clientTimer)*time.Millisecond)
// 	defer cancel()

// 	nid := configGroup.FetchReplicaID(address)
// 	//v, err := utils.StringToInt64(nid)

// 	if config.SplitPorts() {
// 		address = communication.UpdateAddress(address)
// 	}

// 	c, built := connections.Get(address)
// 	if !built || c == nil {
// 		suc := BuildConnection(ctx, nid, address)
// 		if !suc {

// 			p := fmt.Sprintf("[Client Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
// 			logging.PrintLog(true, logging.ErrorLog, p)
// 			communication.NotLive(nid)
// 			clientTimer = clientTimer * 2

// 			//CatchSendRequestError(v, op, rtype, t1)
// 			wg.Done()
// 			return
// 		} else {
// 			c, _ = connections.Get(address)
// 		}
// 	}

// 	var r *pb.RawMessage
// 	_, err = c.SendSigRequest(ctx, &pb.Request{Type: rtype, Request: op})

// 	if err != nil {
// 		p := fmt.Sprintf("[Client Sender Error] could not get reply from node %s, %v", nid, err)
// 		logging.PrintLog(true, logging.ErrorLog, p)
// 		//CatchSendRequestError(v, op, rtype, t1)
// 		connections.Insert(address, nil)
// 		wg.Done()
// 		return
// 	}

// 	re := string(r.GetMsg())
// 	p1 := fmt.Sprintf("[CROSS] got a reply: %s", re)
// 	logging.PrintLog(true, logging.NormalLog, p1)

// 	wg.Done()
// }

func BroadcastRequest(rtype pb.MessageType, op []byte) {

	nodes := communication.FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]
		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Client Sender] Replica %s is not live, not sending any message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}
		wg.Add(1)
		p := fmt.Sprintf("[Client Sender] Send a %v Request to Replica %v", rtype, nid)
		logging.PrintLog(verbose, logging.NormalLog, p)

		go SendRequest(rtype, op, config.FetchAddress(nid))

	}

	wg.Wait()
}

// func BroadcastGRequest(rtype pb.MessageType, op []byte) {
// 	nodes := []string{"0"}

// 	for i := 0; i < len(nodes); i++ {
// 		nid := nodes[i]
// 		if communication.IsNotLive(nid) {
// 			p := fmt.Sprintf("[Client Sender] Replica %s is not live, not sending any message to it", nid)
// 			logging.PrintLog(verbose, logging.NormalLog, p)
// 			continue
// 		}
// 		wg.Add(1)
// 		p := fmt.Sprintf("[Client Sender] Send a %v Request to Replica %v", rtype, nid)
// 		logging.PrintLog(verbose, logging.NormalLog, p)

// 		go SendGRequest(rtype, op, config.FetchAddress(nid))
// 	}

// 	wg.Wait()
// }

func BroadcastConfirmRequest(rtype pb.MessageType, dataSer []byte) {

	nodes := communication.FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]
		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Sig-Confirm(clientsender)] Replica %s is not live, not sending any message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}
		wg.Add(1)
		p := fmt.Sprintf("[Sig-Confirm(clientsender)] Send a %v Request to Replica %v", rtype, nid)
		logging.PrintLog(verbose, logging.NormalLog, p)

		go SendConfirmRequest(rtype, dataSer, config.FetchAddress(nid))
	}
	wg.Wait()
}

func BroadcastCatchUpRequest(rtype pb.MessageType, dataSer []byte) {

	nodes := communication.FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]
		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Sig-CatchUp(clientsender)] Replica %s is not live, not sending any message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}
		wg.Add(1)
		p := fmt.Sprintf("[Sig-CatchUp(clientsender)] Send a %v Request to Replica %v", rtype, nid)
		logging.PrintLog(verbose, logging.NormalLog, p)

		go SendCatchUpRequest(rtype, dataSer, config.FetchAddress(nid))
	}
	wg.Wait()
}

// Sig: broadcast messages
func BroadcastSigRequest(rtype pb.MessageType, op []byte) {
	p1 := fmt.Sprintf("[Cross-sig(clientsender)] Start Broadcasting %v msg length: %v", rtype, len(op))
	logging.PrintLog(true, logging.NormalLog, p1)
	// fetch server nodes
	nodes := communication.FetchNodesFromConfig()

	p := fmt.Sprintf("[Cross-sig(clientsender)] Receivers: %v", nodes)
	logging.PrintLog(true, logging.NormalLog, p)

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]
		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Cross-sig(clientsender)] Replica %s is not live, not sending any message to it", nid)
			logging.PrintLog(true, logging.ErrorLog, p)
			continue
		}

		wg.Add(1)

		p := fmt.Sprintf("[Cross-sig(clientsender)] client send a %v Request to Replica %v", rtype, nid)
		logging.PrintLog(true, logging.NormalLog, p)

		go SendSigRequest(rtype, op, config.FetchAddress(nid))
	}

	wg.Wait()
}

// Sig grpc: type, request, version
func SendSigRequest(rtype pb.MessageType, op []byte, address string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(clientTimer)*time.Millisecond)
	defer cancel()

	nid := config.FetchReplicaID(address)

	p1 := fmt.Sprintf("[Sig-Cross] fetch replica ID#%v by address#%v", nid, address)
	logging.PrintLog(true, logging.NormalLog, p1)

	if config.SplitPorts() {
		address = communication.UpdateAddress(address)
	}

	c, built := connections.Get(address)
	if !built || c == nil {
		suc := BuildConnection(ctx, nid, address)
		if !suc {

			p := fmt.Sprintf("[Client Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			clientTimer = clientTimer * 2

			//CatchSendRequestError(v, op, rtype, t1)
			wg.Done()
			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	_, err = c.SendSigRequest(ctx, &pb.Request{Type: rtype, Request: op})
	if err != nil {
		p := fmt.Sprintf("[Client Sender Error] could not get reply from node %s, %v", nid, err)
		logging.PrintLog(true, logging.ErrorLog, p)
		//CatchSendRequestError(v, op, rtype, t1)
		connections.Insert(address, nil)
		wg.Done()
		return
	}

	p := fmt.Sprintf("[Client Sender] Successfully sent a %v request to %v", rtype, nid)
	logging.PrintLog(true, logging.NormalLog, p)

	wg.Done()
}

// Sig grpc: type, request, version
func SendConfirmRequest(rtype pb.MessageType, op []byte, address string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(clientTimer)*time.Millisecond)
	defer cancel()

	nid := config.FetchReplicaID(address)

	if config.SplitPorts() {
		address = communication.UpdateAddress(address)
	}

	c, built := connections.Get(address)
	if !built || c == nil {
		suc := BuildConnection(ctx, nid, address)
		if !suc {

			p := fmt.Sprintf("[Sig-Confirm(clientsender) Client Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			clientTimer = clientTimer * 2

			//CatchSendRequestError(v, op, rtype, t1)
			wg.Done()
			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	_, err = c.SendConfirmRequest(ctx, &pb.Request{Type: rtype, Request: op})
	if err != nil {
		p := fmt.Sprintf("[Sig-Confirm(clientsender) Client Sender Error] could not get reply from node %s, %v", nid, err)
		logging.PrintLog(true, logging.ErrorLog, p)
		//CatchSendRequestError(v, op, rtype, t1)
		connections.Insert(address, nil)
		wg.Done()
		return
	}

	wg.Done()
}

func SendCatchUpRequest(rtype pb.MessageType, op []byte, address string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(clientTimer)*time.Millisecond)
	defer cancel()

	nid := config.FetchReplicaID(address)

	if config.SplitPorts() {
		address = communication.UpdateAddress(address)
	}

	c, built := connections.Get(address)
	if !built || c == nil {
		suc := BuildConnection(ctx, nid, address)
		if !suc {

			p := fmt.Sprintf("[Sig-CatchUp(clientsender) Client Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			clientTimer = clientTimer * 2

			//CatchSendRequestError(v, op, rtype, t1)
			wg.Done()
			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	_, err = c.SendConfirmRequest(ctx, &pb.Request{Type: rtype, Request: op})
	if err != nil {
		p := fmt.Sprintf("[Sig-CatchUp(clientsender) Client Sender Error] could not get reply from node %s, %v", nid, err)
		logging.PrintLog(true, logging.ErrorLog, p)
		//CatchSendRequestError(v, op, rtype, t1)
		connections.Insert(address, nil)
		wg.Done()
		return
	}

	wg.Done()
}

func StartClientSender(cid string, loadkey bool) {

	// config.LoadConfig()
	verbose = config.FetchVerbose()

	id, err = utils.StringToInt64(cid)
	if err != nil {
		fmt.Println("[Client Sender Error] Client id %v is not valid. Double check the configuration file", id)
		return
	}

	cryptolib.StartCrypto(id, config.CryptoOption())

	communication.StartConnectionManager()

	connections.Init()

	clientTimer = config.FetchClientTimer()

	dialOpt = []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	log.Printf("[Client Sender] Start Client Sender id#%s ok", cid)

}
