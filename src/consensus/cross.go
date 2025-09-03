package consensus

import (
	// "CrossRBC/src/add"
	"CrossRBC/src/communication/sender"
	pb "CrossRBC/src/proto/proto/communication"

	// "CrossRBC/src/config"
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	"CrossRBC/src/quorum"
	"CrossRBC/src/utils"
	"fmt"
	"log"
	"sync"
)

type RBCStatus int

const (
	// STATUS_CROSSIDLE  RBCStatus = 0 //Not start
	STATUS_CCROSS RBCStatus = 1 //Cross
	STATUS_SEND   RBCStatus = 2 //Propose
	STATUS_ECHO   RBCStatus = 3 //Echo
	STATUS_READY  RBCStatus = 4 //Ready
)

var crosslock sync.Mutex

var rstatus utils.IntBoolMap       //broadcast status,only has value when  RBC Deliver
var instancestatus utils.IntIntMap // status for each instance, used in RBC
var cachestatus utils.IntIntMap    // status for each instance
var received utils.IntSet

var receivedSig utils.IntByteMap
var receivedReq utils.IntByteMap

var epochStatus utils.IntIntMap //track the status for each epoch
// var epochCrossNum utils.IntIntMap //track the number of cross messages for each epoch
var epochReadyNum utils.IntIntMap //track the number of ready messages for each epoch

var msgLocal []byte //original message
var sourceMsg []byte
var checkedTxNum int
var T_ra_rbc int64

// xepoch is the number of epochs, used to check all epoch is delivered
var xepoch int

func XHandleRBCMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype

	p := fmt.Sprintf("[CROSS] Received CROSS message from %v(group A), type %v(=1)", content.Source, mtype)
	logging.PrintLog(verbose, logging.NormalLog, p)

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	//log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.RBC_ALL:
		// raCrossTS = utils.MakeTimestamp()
		XHandleCross(content)
	// case message.RBC_SEND:
	// 	HandleSend(content)
	// case message.RBC_ECHO:
	// 	HandleEcho(content)
	// case message.RBC_READY:
	// 	HandleReady(content)
	default:
		log.Printf("[Cross-Handler] message type #%v not supported", mtype)
	}

}

func XSendCross(data []byte, numReq int) {
	// elock.Lock()
	if t_abc_start == 0 {
		t_abc_start = utils.MakeTimestamp()
		p1 := fmt.Sprintf("[DEBUG-ABC] group A: Send Cross ts: %d", t_abc_start)
		logging.PrintLog(true, logging.EvaluationLog, p1)
	}

	for i := 0; i < numReq; i++ {
		msg := message.ReplicaMessage{
			Instance: i,
			Payload:  data,
			Source:   id,
			Mtype:    message.RBC_ALL,
			Round:    numReq,
			TS:       t_abc_start,
		}
		msgbyte, _ := msg.Serialize()

		sender.MACBroadcast(msgbyte, message.RBC)

		p := fmt.Sprintf("[CROSS-#%v] group A: CROSS send", i)
		logging.PrintLog(verbose, logging.NormalLog, p)
	}
}

// func XQueryStatus(txNum int) bool {
// 	quorum := txNum
// 	if checkedTxNum > quorum {
// 		return false
// 	}

// 	// if txNum == 4 {
// 	// 	quorum = 1
// 	// } else {
// 	// 	quorum = txNum / numOfClient * 9 / 10
// 	// }

// 	p := fmt.Sprintf("[RA-QueryStatus] txNum:%v, checkedTxNum:%v", quorum, checkedTxNum)
// 	logging.PrintLog(verbose, logging.NormalLog, p)

// 	for i := checkedTxNum; i < txNum; i++ {
// 		tag, _ := rstatus.Get(i)
// 		if tag {
// 			// trueCount++
// 			checkedTxNum = i + 1
// 			p := fmt.Sprintf("[RA] r-delivered, quorum:%v, checkedTxNum:%v", quorum, checkedTxNum)
// 			logging.PrintLog(verbose, logging.NormalLog, p)
// 		} else {
// 			time.Sleep(10 * time.Millisecond)
// 		}
// 	}

// 	return checkedTxNum == quorum
// }

// func XQueryStatusCount() int {
// 	return rstatus.GetCount()
// }

// func XQueryReq(instanceid int) []byte {
// 	v, exist := receivedReq.Get(instanceid)
// 	if !exist {
// 		return nil
// 	}
// 	return v
// }

// func XQueryECHOStatus(instanceid int) bool {
// 	stat, _ := instancestatus.Get(instanceid)
// 	v, _ := receivedSig.Get(instanceid)
// 	return stat >= int(STATUS_ECHO) && v != nil
// }

// func XQuerySig(instanceid int) []byte {
// 	v, _ := receivedSig.Get(instanceid)
// 	return v
// }

var crossCCompleteList utils.IntByteMap

// var rbcSendTag bool
var tsCROSSStart int64
var receivedStartABCTS int64
var crossABCMsgQueue utils.EpochMsgQueue

func XHandleCross(m message.ReplicaMessage) {

	// t_abc_start = utils.MakeTimestamp()
	// ts := utils.MakeTimestamp()
	// if tsCROSSStart == 0 {
	// 	tsRAStart = ts
	// 	p := fmt.Sprintf("[DEBUG-Cross-RA] Start CROSS ts:%v", tsRAStart)
	// 	logging.PrintLog(verbose, logging.EvaluationLog, p)
	// }

	msgLocal = m.Payload
	sourceID := m.Source
	sourceMsg = m.Payload
	sourceEpoch := int(m.Instance)

	if addtotalEpoch == 0 {
		addtotalEpoch = m.Round
	}

	if receivedStartABCTS == 0 {
		receivedStartABCTS = m.TS
	}

	epochStatus.Insert(sourceEpoch, int(STATUS_CCROSS))

	//tag = instance + msg hash
	epochByte := []byte(fmt.Sprintf("%v", sourceEpoch))
	quorum_tag := cryptolib.GenInstanceHash(epochByte, sourceMsg)
	quorum.Add(sourceID, string(quorum_tag), sourceMsg, quorum.PP)

	//check quorum size
	p := fmt.Sprintf("[CROSS-#%v(%v)] Handling CROSS message from Client#%v: quorum size %v(2), queueLen:%v", sourceEpoch, addtotalEpoch, sourceID, quorum.CheckCurNum(string(quorum_tag), quorum.PP), crossABCMsgQueue.Len())
	logging.PrintLog(verbose, logging.NormalLog, p)

	// _, exist := crossCCompleteList.Get(sourceEpoch)
	// flag, _ := epochStatus.Get(sourceEpoch)

	if crossABCMsgQueue.Len() == addtotalEpoch {
		return
	}

	if quorum.CheckCrossSmallQuorum(string(quorum_tag), quorum.PP) {

		AddMessageToCrossABCMsgQueue(sourceMsg, sourceEpoch)
		p := fmt.Sprintf("[CROSS-#%v] meet cross quorum", sourceEpoch)
		logging.PrintLog(verbose, logging.NormalLog, p)

		// epochStatus.Insert(sourceEpoch, int(STATUS_SEND))
		// crossCCompleteList.Insert(sourceEpoch, sourceMsg)
	}
}

func AddMessageToCrossABCMsgQueue(msg []byte, epoch int) {
	crossABCMsgQueue.AddItem(msg, epoch)
	p := fmt.Sprintf("[QUEUE-#%v] check queue(len:%v)", epoch, crossABCMsgQueue.Len())
	logging.PrintLog(verbose, logging.NormalLog, p)

}

func ReadyStartHotStuff() {
	for {
		if id != 4 {
			return
		}

		if crossABCMsgQueue.Len() > 0 {
			sMsg, epoch, ok := crossABCMsgQueue.GetAndRemove()
			if !ok {
				p := fmt.Sprintf("[ERROR-CROSS] cannot get crossABCMsgQueue")
				logging.PrintLog(verbose, logging.ErrorLog, p)
				break
			}

			p := fmt.Sprintf("[QUEUE-#%v] Start HotStuff", epoch)
			logging.PrintLog(verbose, logging.NormalLog, p)

			if t_abc_abc == 0 {
				t_abc_abc = utils.MakeTimestamp()
				p1 := fmt.Sprintf("[DEBUG-ABC] group B: Start ABC ts: %d", t_abc_abc)
				logging.PrintLog(verbose, logging.EvaluationLog, p1)
			}

			hsMsg := []pb.RawMessage{{Msg: sMsg}}
			StartHotStuff(hsMsg, epoch)
		}
	}
}

func XReadRBCBreakdown() int64 {
	return T_ra_rbc
}

// func XHandleSend(m message.ReplicaMessage) {
// 	result, exist := rstatus.Get(m.Instance)
// 	if exist && result {
// 		return
// 	}

// 	p := fmt.Sprintf("[RBC-epoch#%v-rbcEpoch#%v] Handling SEND message from node %v", m.Instance, rbcEpoch.Get(), m.Source)
// 	logging.PrintLog(verbose, logging.NormalLog, p)
// 	instancestatus.Insert(m.Instance, int(STATUS_SEND))
// 	epochStatus.Insert(m.Instance, int(STATUS_ECHO))

// 	msg := m
// 	msg.Source = id
// 	msg.Mtype = message.RBC_ECHO
// 	msg.Hash = cryptolib.GenInstanceHash(utils.IntToBytes(m.Instance), m.Payload)

// 	if !received.IsTrue(m.Instance) {
// 		receivedReq.Insert(m.Instance, m.Payload)
// 		received.AddItem(m.Instance)
// 		receivedSig.Insert(m.Instance, m.Sig)
// 	}

// 	msgbyte, err := msg.Serialize()
// 	if err != nil {
// 		log.Fatalf("failed to serialize echo message")
// 	}

// 	sender.MACBroadcast(msgbyte, message.RBC)

// 	v, exist := cachestatus.Get(m.Instance)
// 	if exist && v >= int(STATUS_ECHO) {
// 		SendReady(m)
// 	}
// 	if exist && v == int(STATUS_READY) {
// 		Deliver(m.Instance)
// 	}
// }

// func XHandleEcho(m message.ReplicaMessage) {
// 	result, exist := rstatus.Get(m.Instance)
// 	if exist && result {
// 		return
// 	}

// 	p := fmt.Sprintf("[RBC-epoch#%v-rbcEpoch#%v] Handling echo message from node %v", m.Instance, rbcEpoch.Get(), m.Source)
// 	logging.PrintLog(verbose, logging.NormalLog, p)

// 	hash := utils.BytesToString(m.Hash)
// 	quorum.Add(m.Source, hash, nil, quorum.FW)

// 	if quorum.CheckQuorum(hash, quorum.FW) {
// 		if !received.IsTrue(m.Instance) && !wrbc {
// 			receivedReq.Insert(m.Instance, m.Payload)
// 			received.AddItem(m.Instance)
// 		}
// 		epochStatus.Insert(m.Instance, int(STATUS_READY))

// 		p1 := fmt.Sprintf("[RBC-ECHO-epoch#%v-rbcEpoch#%v] Meet Quorum -to send READY", m.Instance, rbcEpoch.Get())
// 		logging.PrintLog(verbose, logging.NormalLog, p1)

// 		SendReady(m)
// 	}
// }

// func XSendReady(m message.ReplicaMessage) {
// 	crosslock.Lock()
// 	stat, _ := instancestatus.Get(m.Instance)

// 	if stat == int(STATUS_SEND) {
// 		instancestatus.Insert(m.Instance, int(STATUS_ECHO))
// 		crosslock.Unlock()
// 		p := fmt.Sprintf("[RBC-epoch#%v-rbcEpoch#%v] Sending READY", m.Instance, rbcEpoch.Get())
// 		logging.PrintLog(verbose, logging.NormalLog, p)

// 		msg := m
// 		msg.Source = id
// 		msg.Mtype = message.RBC_READY
// 		msgbyte, err := msg.Serialize()
// 		if err != nil {
// 			log.Fatalf("failed to serialize ready message")
// 		}
// 		sender.MACBroadcast(msgbyte, message.RBC)
// 	} else {
// 		v, exist := cachestatus.Get(m.Instance)
// 		crosslock.Unlock()
// 		if exist && v == int(STATUS_READY) {
// 			instancestatus.Insert(m.Instance, int(STATUS_ECHO))
// 			p1 := fmt.Sprintf("[RBC-epoch#%v-rbcEpoch#%v] Sending READY", m.Instance, rbcEpoch.Get())
// 			logging.PrintLog(verbose, logging.NormalLog, p1)
// 			Deliver(m.Instance)
// 			cachestatus.Insert(m.Instance, int(STATUS_ECHO))
// 		}
// 	}
// }

// func HandleReady(m message.ReplicaMessage) {
// 	result, exist := rstatus.Get(m.Instance)
// 	if exist && result {
// 		return
// 	}

// 	p := fmt.Sprintf("[RBC-READY-epoch#%v-rbcEpoch#%v] Handling ready message from node %v", m.Instance, rbcEpoch.Get(), m.Source)
// 	logging.PrintLog(verbose, logging.NormalLog, p)

// 	hash := utils.BytesToString(m.Hash)
// 	quorum.Add(m.Source, hash, nil, quorum.CM)

// 	p1 := fmt.Sprintf("[RBC-READY-epoch#%v-rbcEpoch#%v] Quorum size %v", m.Instance, rbcEpoch.Get(), quorum.CheckCurNum(hash, quorum.CM))
// 	logging.PrintLog(verbose, logging.NormalLog, p1)

// 	if quorum.CheckEqualSmallQuorum(hash) {
// 		if !received.IsTrue(m.Instance) {
// 			receivedReq.Insert(m.Instance, m.Payload)
// 			received.AddItem(m.Instance)
// 		}
// 		SendReady(m)
// 	}

// 	if quorum.CheckQuorum(hash, quorum.CM) {
// 		stat, _ := instancestatus.Get(m.Instance)
// 		p2 := fmt.Sprintf("[RBC-READY-epoch#%v-rbcEpoch#%v] Meet Quorum to deliver, instancestatus=%v", m.Instance, rbcEpoch.Get(), stat)
// 		logging.PrintLog(verbose, logging.NormalLog, p2)
// 		instancestatus.Insert(m.Instance, int(STATUS_READY))

// 		Deliver(m.Instance)
// 	}
// }

var rbcDelivered utils.IntBoolMap
var xDelivered bool
var crosstEpoch int

// func Deliver(instance int) {
// 	crosslock.Lock()
// 	defer crosslock.Unlock()

// 	checkDeliverStatus, _ := rbcDelivered.Get(instance)
// 	if !checkDeliverStatus {
// 		rbcDelivered.Insert(instance, true)
// 		p := fmt.Sprintf("[RBC] Delivered Epochs:%v (total epoch:%v)", rbcDelivered.GetCount(), crosstEpoch)
// 		logging.PrintLog(verbose, logging.NormalLog, p)
// 	}

// 	len := rbcDelivered.GetCount()
// 	if len == crosstEpoch {
// 		xDelivered = true
// 	}
// }

// func IsXDelivered() bool {
// 	return xDelivered
// }

// func ReadTotalEpoch() int {
// 	return crosstEpoch
// }

// // var id int64
// // var n int
// // var verbose bool
// // var epoch utils.IntValue
// var rbcEpoch utils.IntValue
// var wrbc bool
// var inputOP []byte
// var localOP utils.IntByteMap

// func SetStatus(instanceid int, status int) {
// 	epochStatus.Insert(instanceid, status)
// 	p := fmt.Sprintf("[RA-epoch#%v] Set epoch status %v", instanceid, status)
// 	logging.PrintLog(verbose, logging.NormalLog, p)
// }

// func StartRBC(instanceid int, input []byte, totalEpoch int) {
// 	ts := utils.MakeTimestamp()

// 	if T_ra_rbc == 0 {
// 		T_ra_rbc = ts
// 		p := fmt.Sprintf("[DEBUG-RA-epoch#%v] Start RBC ts:%v", instanceid, ts)
// 		logging.PrintLog(verbose, logging.EvaluationLog, p)
// 	}

// 	tEpoch = totalEpoch

// 	cur_epoch_status, _ := epochStatus.Get(instanceid)
// 	localOP.Insert(instanceid, input)

// 	// identify the current status is send, if yes send the propose message
// 	if cur_epoch_status == int(STATUS_SEND) {
// 		p := fmt.Sprintf("[RA-PROPOSE-epoch#%v-id#%v-rbcEpoch#%v] Starting sending PROPOSE msg \n", instanceid, id, rbcEpoch.Get())
// 		logging.PrintLog(verbose, logging.NormalLog, p)

// 		msg := message.ReplicaMessage{
// 			Mtype:    message.RBC_SEND,
// 			Instance: instanceid,     //epoch number
// 			Source:   id,             //sender id
// 			Payload:  inputOP,        //msg = op
// 			Epoch:    rbcEpoch.Get(), //env epoch for rbc
// 		}

// 		msgbyte, err := msg.Serialize()
// 		if err != nil {
// 			log.Fatalf("failed to serialize RBC message")
// 		}
// 		sender.MACBroadcast(msgbyte, message.RBC)
// 	} else {
// 		// if
// 	}

// }

// var raCrossTS int64

// func HandleRBCMsg(inputMsg []byte) {

// 	tmp := message.DeserializeMessageWithSignature(inputMsg)
// 	input := tmp.Msg
// 	content := message.DeserializeReplicaMessage(input)
// 	mtype := content.Mtype

// 	p := fmt.Sprintf("[RA-CROSS] Received CROSS message from %v(group A), type %v(=1)", content.Source, mtype)
// 	logging.PrintLog(verbose, logging.NormalLog, p)

// 	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
// 		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
// 		return
// 	}

// 	//log.Printf("handling message from %v, type %v", source, mtype)
// 	switch mtype {
// 	case message.RBC_ALL:
// 		raCrossTS = utils.MakeTimestamp()
// 		HandleCross(content)
// 	case message.RBC_SEND:
// 		HandleSend(content)
// 	case message.RBC_ECHO:
// 		HandleEcho(content)
// 	case message.RBC_READY:
// 		HandleReady(content)
// 	default:
// 		log.Printf("[RA-Handler] message type #%v not supported", mtype)
// 	}

// }

// func ReadRACrossStart() int64 {
// 	return tsRAStart
// }

// func SetHash() { // use hash for the echo and ready phase
// 	wrbc = true
// }

// func SetEpoch(e int) {
// 	epoch.Set(e)
// }

// func SetrbcEpoch(e int) {
// 	rbcEpoch.Set(e)
// }

// func EpochStatusInit(epoch int) {
// 	epochStatus.Init()
// 	epochStatus.Insert(epoch, int(STATUS_IDLE))
// }

func InitCROSS(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	//log.Printf("ini rstatus %v",rstatus.GetAll())
	rstatus.Init()
	rstatus.Insert(epoch.Get(), false)
	epoch.Init()
	crossCCompleteList.Init()
	// rbcSendTag = false
	rbcDelivered.Init()
	xDelivered = false
	instancestatus.Init()
	cachestatus.Init()
	epochStatus.Init()
	receivedReq.Init()
	received.Init()
	receivedSig.Init()
	epochReadyNum.Init()
	crossABCMsgQueue = *utils.NewEpochMsgQueue()
	tsRAStart = 0
	checkedTxNum = 0
	tEpoch = 0
	T_ra_rbc = 0
	xepoch = 0
	receivedStartABCTS = 0

	go func() {
		log.Printf("[ABC] ReadyStartABC Goroutine started")
		ReadyStartHotStuff()
	}()

	log.Printf("[RA]RBC initialized")

}

// func ClearRBCStatus(instanceid int) {
// 	rstatus.Delete(instanceid)
// 	instancestatus.Delete(instanceid)
// 	cachestatus.Delete(instanceid)
// 	receivedReq.Delete(instanceid)
// }
