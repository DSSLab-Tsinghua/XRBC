/*GetLenMACBroadcast
Sender functions.
It implements all sending functions for replicas.
*/

package sender

import (
	"CrossRBC/src/communication"
	"CrossRBC/src/config"
	"CrossRBC/src/cryptolib"
	logging "CrossRBC/src/logging"
	"CrossRBC/src/message"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/utils"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var id int64
var idstring string
var err error

// var completed map[string]bool
var verbose bool

var wg sync.WaitGroup

var broadcastTimer int
var sleepTimerValue int
var reply []byte

var dialOpt []grpc.DialOption
var connections communication.AddrConnMap

func BuildConnection(ctx context.Context, nid string, address string) bool {
	p := fmt.Sprintf("building a connection with %v", nid)
	logging.PrintLog(verbose, logging.NormalLog, p)

	// conn, err := grpc.DialContext(ctx, address, dialOpt...)
	conn, err := grpc.NewClient(address, dialOpt...)

	if err != nil {
		p := fmt.Sprintf("[Communication Sender Error] failed to bulid a connection with %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return false
	}

	p1 := fmt.Sprintf("[CommunicationSender(sender)] successfully built a connection with %v, address:%v", nid, address)
	logging.PrintLog(verbose, logging.NormalLog, p1)
	c := pb.NewSendClient(conn)

	connections.Insert(address, c)
	connections.InsertID(address, nid)
	return true
}

// for hotstuff
func RBCByteBroadcast(msg []byte) {

	request, err := message.SerializeWithSignature(id, msg)
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to sign the message")
		return
	}

	nodes := FetchNodesFromConfig()
	p := fmt.Sprintf("[Communication Sender-RBCByteBroadcast-for hotstuff] nodelist: %v", nodes)
	logging.PrintLog(verbose, logging.NormalLog, p)

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		if nid == idstring {
			continue
		}

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}

		addrs := FetchAddressByNid(nid)

		go ByteSend(request, addrs, message.HotStuff_Msg)
	}
}

func FetchAddressByNid(nid string) string {
	cons := config.GConsensus()
	addr := ""
	switch cons {
	case 103:
		addr = config.FetchAddress(nid)
	default:
		addr = config.FetchAddress(nid)
	}
	return addr
}

func ByteSend(msg []byte, address string, msgType message.TypeOfMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(broadcastTimer)*time.Millisecond)
	defer cancel()

	if address == "" {
		p_1 := fmt.Sprintf("[Communication Sender Error-ByteSend] did not find destination id's address")
		logging.PrintLog(true, logging.ErrorLog, p_1)
		return
	}

	if msg == nil {
		p_2 := fmt.Sprintf("[Communication Sender Error-ByteSend] Message is nil, cannot send")
		logging.PrintLog(true, logging.ErrorLog, p_2)
		return
	}

	nid := config.FetchReplicaID(address) //dest id
	c, built := connections.Get(address)  //
	existnid := connections.GetID(address)

	//p:= fmt.Sprintf("Ready to send request to %v in %v(%v)", nid, c, existnid)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	if !built || c == nil || nid != existnid {
		suc := BuildConnection(ctx, nid, address)
		if !suc {
			p := fmt.Sprintf("[Communication Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)

			communication.NotLive(nid)
			broadcastTimer = broadcastTimer * 2

			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	switch msgType {
	case message.RBC_ALL:
		_, err = c.RBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.CrossAdd_Disperse:
		_, err = c.RBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		// p := fmt.Sprintf("[Communication(sender)-share] send SHARE message to group B")
		// logging.PrintLog(true, logging.NormalLog, p)

		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send Cross Add Disperse Msg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}

	case message.ABA_ALL:
		_, err = c.ABASendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.PRF:
		_, err = c.PRFSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.ECRBC_ALL:
		_, err = c.ECRBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.CBC_ALL:
		_, err = c.CBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.EVCBC_ALL:
		_, err = c.EVCBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.MVBA_DISTRIBUTE:
		_, err = c.MVBASendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.RETRIEVE:
		_, err = c.RetrieveSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.SIMPLE_SEND:
		_, err = c.SimpleSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.ECHO_SEND:
		_, err = c.EchoSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.GC_ALL:
		_, err = c.GCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.SIMPLE_PROOF:
		_, err = c.SimpleProofSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.PLAINCBC_ALL:
		_, err = c.PlainCBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.MBA_ALL:
		_, err = c.MBASendByteMsg(ctx, &pb.RawMessage{Msg: msg})

		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}
	case message.HotStuff_Msg:
		_, err = c.HotStuffSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}
	case message.CrossDelta_Vote:
		_, err = c.RBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}

	default:
		log.Fatalf("message type %v not supported", msgType)
	}
}

func ByteGSigSend(msg []byte, address string, msgType message.TypeOfMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(broadcastTimer)*time.Millisecond)
	defer cancel()

	if address == "" {
		p_1 := fmt.Sprintf("[Communication Sender Error-ByteGSigSend] did not find destination id's address")
		logging.PrintLog(true, logging.ErrorLog, p_1)
		return
	}

	nid := config.FetchReplicaID(address) //dest id
	c, built := connections.Get(address)  //
	existnid := connections.GetID(address)

	//p:= fmt.Sprintf("Ready to send request to %v in %v(%v)", nid, c, existnid)
	//logging.PrintLog(verbose, logging.NormalLog, p)

	if !built || c == nil || nid != existnid {
		suc := BuildConnection(ctx, nid, address)
		if !suc {
			p := fmt.Sprintf("[Communication Sender Error] did not connect to node %s, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)

			communication.NotLive(nid)
			broadcastTimer = broadcastTimer * 2

			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	switch msgType {
	case message.RBC_ALL:
		_, err = c.RBCSendByteMsg(ctx, &pb.RawMessage{Msg: msg})
		if err != nil {
			p := fmt.Sprintf("[Communication Sender Error] could not get reply from node %s when send ReplicaMsg, set it to notlive: %v", nid, err)
			logging.PrintLog(true, logging.ErrorLog, p)
			communication.NotLive(nid)
			connections.Insert(address, nil)
			return
		}

	default:
		log.Fatalf("message type %v not supported", msgType)
	}
}

func MACBroadcast(msg []byte, mtype message.ProtocolType) {
	nodes := FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		dest, _ := utils.StringToInt64(nid)
		request, err := message.SerializeWithMAC(id, dest, msg)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
			continue
		}

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}

		switch mtype {
		case message.RBC: //maybe bug here
			go ByteSend(request, config.FetchAddress(nid), message.RBC_ALL)
		case message.ABA:
			go ByteSend(request, config.FetchAddress(nid), message.ABA_ALL)
		case message.CBC:
			go ByteSend(request, config.FetchAddress(nid), message.CBC_ALL)
		case message.EVCBC:
			go ByteSend(request, config.FetchAddress(nid), message.EVCBC_ALL)
		case message.MVBA:
			go ByteSend(request, config.FetchAddress(nid), message.MVBA_DISTRIBUTE)
		case message.SIMPLE:
			go ByteSend(request, config.FetchAddress(nid), message.SIMPLE_SEND)
		case message.ECHO:
			go ByteSend(request, config.FetchAddress(nid), message.ECHO_SEND)
		case message.GC:
			go ByteSend(request, config.FetchAddress(nid), message.GC_ALL)
		case message.SIMPLEPROOF:
			go ByteSend(request, config.FetchAddress(nid), message.SIMPLE_PROOF)
		case message.PLAINCBC:
			go ByteSend(request, config.FetchAddress(nid), message.PLAINCBC_ALL)
		case message.MBA:
			go ByteSend(request, config.FetchAddress(nid), message.MBA_ALL)

		}

	}
}

func TOABroadcast(msg []byte, mtype message.ProtocolType) {

	nodes := config.FetchANodes()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		dest, _ := utils.StringToInt64(nid)
		request, err := message.SerializeWithMAC(id, dest, msg)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
			continue
		}

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}

		switch mtype {
		case message.CrossSigGroup: //maybe bug here
			go ByteGSigSend(request, config.FetchAddress(nid), message.RBC_ALL)
		}

	}
}

func TOBBroadcast(msg []byte, mtype message.ProtocolType) {

	nodes := config.FetchBNodes()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		dest, _ := utils.StringToInt64(nid)
		request, err := message.SerializeWithMAC(id, dest, msg)
		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
			continue
		}

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}

		switch mtype {
		case message.CrossSigGroup: //maybe bug here
			go ByteGSigSend(request, config.FetchAddress(nid), message.RBC_ALL)
		}

	}
}

func ToAGroupResNode(msg []byte, dest int64, mtype message.ProtocolType) {

	nid := utils.Int64ToString(dest)

	request, err := message.SerializeWithMAC(id, dest, msg)
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
		return
	}

	if communication.IsNotLive(nid) {
		p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
		logging.PrintLog(verbose, logging.NormalLog, p)
		return
	}
	switch mtype {
	case message.CrossSigGroup:
		go ByteGSigSend(request, config.FetchAddress(nid), message.RBC_ALL)
	}

}

func SendToNode(msg []byte, dest int64, mtype message.ProtocolType) {

	nid := utils.Int64ToString(dest)

	request, err := message.SerializeWithMAC(id, dest, msg)
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
		return
	}

	if communication.IsNotLive(nid) {
		p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
		logging.PrintLog(verbose, logging.NormalLog, p)
		return
	}
	switch mtype {
	case message.CrossADD:
		go ByteSend(request, config.FetchAddress(nid), message.CrossAdd_Disperse)
	case message.CrossADDC:
		go ByteSend(request, config.FetchAddress(nid), message.CrossAdd_Reconsruct)
	case message.CBC:
		go ByteSend(request, config.FetchAddress(nid), message.CBC_ALL)
	case message.EVCBC:
		go ByteSend(request, config.FetchAddress(nid), message.EVCBC_ALL)
	case message.ECHO:
		go ByteSend(request, config.FetchAddress(nid), message.ECHO_SEND)
	case message.PLAINCBC:
		go ByteSend(request, config.FetchAddress(nid), message.PLAINCBC_ALL)
	case message.HotStuff:
		request, err := message.SerializeWithSignature(id, msg)
		// p := fmt.Sprintf("(ABC-Hotstuff-Sender) HotStuff send %v message", mtype)
		// logging.PrintLog(verbose, logging.NormalLog, p)

		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to sign the message")
			return
		}
		go ByteSend(request, config.FetchAddress(nid), message.HotStuff_Msg)

	}

}

/*
isDifferentData means whether the message in data is different, in other words, whether the messages sent to
different replicas are different.
*/
func MACBroadcastWithErasureCode(data [][]byte, mtype message.ProtocolType, isDifferentData bool) {

	nodes := FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		dest, _ := utils.StringToInt64(nid)
		var request []byte
		var err error
		if isDifferentData {
			request, err = message.SerializeWithMAC(id, dest, data[i])
		} else {
			request, err = message.SerializeWithMAC(id, dest, data[0])
		}

		if err != nil {
			logging.PrintLog(true, logging.ErrorLog, "[Sender Error] Not able to generate MAC")
			continue
		}

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}

		log.Println("start send: mtype", mtype)

		switch mtype {
		case message.ECRBC:
			go ByteSend(request, config.FetchAddress(nid), message.ECRBC_ALL)
		case message.MVBA:
			go ByteSend(request, config.FetchAddress(nid), message.RETRIEVE)
		case message.CBC:
			go ByteSend(request, config.FetchAddress(nid), message.CBC_ALL)
		case message.GC:
			go ByteSend(request, config.FetchAddress(nid), message.GC_ALL)
		case message.PBFTEC:
			go ByteSend(request, config.FetchAddress(nid), message.PBFTEC_PP)
		default:
			log.Printf("MACEC: no supported message type:", mtype)
		}

	}
}

func CoinBroadcast(nodeid int64, instanceid int, roundnum int, mtype message.TypeOfMessage) {
	nodes := FetchNodesFromConfig()

	for i := 0; i < len(nodes); i++ {
		nid := nodes[i]

		shareinput := utils.IntToBytes(instanceid + roundnum)
		request := cryptolib.GenPRFShare(shareinput)

		m := message.ReplicaMessage{
			Source:   nodeid,
			Instance: instanceid,
			Round:    roundnum,
			Payload:  request,
			Hash:     shareinput, //Used as C for coin messages
		}

		msgbyte, _ := m.Serialize()

		if communication.IsNotLive(nid) {
			p := fmt.Sprintf("[Communication Sender] Replica %v is not live, don't send message to it", nid)
			logging.PrintLog(verbose, logging.NormalLog, p)
			continue
		}

		go ByteSend(msgbyte, config.FetchAddress(nid), mtype)

	}
}

/*
Used for membership protocol to fetch list of nodes
Output

	[]string: a list of nodes (in the string type)
*/
func FetchNodesFromConfig() []string {
	cons := config.Consensus()
	nodelist := make([]string, 0)
	switch cons {
	case 100, 101:
		nodelist = config.FetchBNodes()
	case 102:
		nodelist = config.FetchNodes()
	case 103:
		nodelist = config.FetchBNodes()
	default:
		nodelist = config.FetchNodes()
	}
	return nodelist
}

func StartSender(rid string) {
	log.Printf("Starting sender %v", rid)
	// config.LoadConfig()
	verbose = config.FetchVerbose()
	idstring = rid

	id, err = utils.StringToInt64(rid) // string to int64
	if err != nil {
		p := fmt.Sprintf("[Communication Sender Error] Replica id %v is not valid. Double check the configuration file", id)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	// Set up a connection to the server.

	dialOpt = []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		//grpc.WithKeepaliveParams(kacp),
	}

	connections.Init()

	verbose = config.FetchVerbose()
	communication.StartConnectionManager()
	broadcastTimer = config.FetchBroadcastTimer()
	sleepTimerValue = config.FetchSleepTimer()
}

func SetId(newnid int64) {
	id = newnid
}

// Send to client by client id
func SendtoClientWithID(dataSer []byte, cid string, mtype pb.MessageType) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(broadcastTimer)*time.Millisecond)
	defer cancel()

	address := config.FetchClientAddress(cid)
	if address == "" {
		p := fmt.Sprintf("[Cross-%v(sender):Communication Sender Error] did not find destination client id's address", mtype)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	p := fmt.Sprintf("[Cross-%v(sender)] ready to send request to client node %v in %v", mtype, cid, address)
	logging.PrintLog(verbose, logging.NormalLog, p)

	c, built := connections.Get(address)
	existnid := connections.GetID(address)

	if !built || c == nil || cid != existnid {
		suc := BuildConnection(ctx, cid, address)
		if !suc {
			p := fmt.Sprintf("[Communication Sender Error(sender)] did not connect to client node %s, set it to notlive: %v", cid, err)
			logging.PrintLog(true, logging.ErrorLog, p)

			communication.NotLive(cid)
			broadcastTimer = broadcastTimer * 2

			return
		} else {
			c, _ = connections.Get(address)
		}
	}

	switch mtype {
	case 17:
		p := fmt.Sprintf("[Cross-Reply(sender)] servers send reply to client node %v", cid)
		logging.PrintLog(verbose, logging.NormalLog, p)
		_, err = c.SendReplyRequest(ctx, &pb.Request{Type: pb.MessageType_CROSSREPLY, Request: dataSer})
	case 18:
		_, err = c.SendReplyRequest(ctx, &pb.Request{Type: pb.MessageType_CROSSCONFIRM, Request: dataSer})
		p := fmt.Sprintf("[Cross-Confirm(sender)] servers send confirm to server node %v", cid)
		logging.PrintLog(verbose, logging.NormalLog, p)
	case 19:
		_, err = c.SendReplyRequest(ctx, &pb.Request{Type: pb.MessageType_CROSSFETCH, Request: dataSer})
		p := fmt.Sprintf("[Cross-Fetch(sender)] servers send fetch to client node %v", cid)
		logging.PrintLog(verbose, logging.NormalLog, p)
	}

}
