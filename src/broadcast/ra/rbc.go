package ra

import (
	// "CrossRBC/src/add"
	"CrossRBC/src/communication/sender"

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
	STATUS_IDLE    RBCStatus = 0 //Not start
	STATUS_CROSS   RBCStatus = 1 //Cross
	STATUS_SEND    RBCStatus = 2 //Propose
	STATUS_ECHO    RBCStatus = 3 //Echo
	STATUS_READY   RBCStatus = 4 //Ready
	STATUS_DELIVER RBCStatus = 5 //Delivered
)

var rstatus utils.IntBoolMap //broadcast status,only has value when  RBC Deliver
// var instancestatus utils.IntIntMap // status for each instance, used in RBC
// var cachestatus utils.IntIntMap    // status for each instance
var received utils.IntSet

// var elock sync.Mutex

// var rlock sync.Mutex
var receivedSig utils.IntByteMap
var receivedReq utils.IntByteMap

var epochStatus sync.Map //track the status for each epoch
// var epochStatusLock sync.Mutex

// cross message
var crossMsgQueue utils.EpochMsgQueue //store the cross message

// var epochCrossNum utils.IntIntMap //track the number of cross messages for each epoch
var epochReadyNum utils.IntIntMap //track the number of ready messages for each epoch

var msgLocal []byte //original message
// var sourceMsg []byte

// var checkedTxNum int
var T_ra_rbc int64

// timestamps:
var tsRAStart int64

func ReadRBCBreakdown() int64 {
	return T_ra_rbc
}

func SendCross(data []byte, numReq int) {
	if tsRAStart == 0 {
		tsRAStart = utils.MakeTimestamp()
		p := fmt.Sprintf("[DEBUG-Cross-RA] Start CROSS startTS:%v", tsRAStart)
		logging.PrintLog(verbose, logging.EvaluationLog, p)
	}

	// elock.Lock()
	p := fmt.Sprintf("[CROSS] Send %v Request to B, startTS:%d", numReq, tsRAStart)
	logging.PrintLog(verbose, logging.NormalLog, p)

	for i := 0; i < numReq; i++ {
		msg := message.ReplicaMessage{
			Instance: i,
			Payload:  data,
			Source:   id,
			Mtype:    message.RBC_ALL,
			Round:    numReq,
			TS:       tsRAStart,
		}
		msgbyte, _ := msg.Serialize()

		sender.MACBroadcast(msgbyte, message.RBC)

	}
}

var GlobalTotalEpoch int
var EpochList utils.IntSet
var receivedStartTS int64

func HandleCross(m message.ReplicaMessage) {
	tsCross := utils.MakeTimestamp()

	if receivedStartTS == 0 {
		receivedStartTS = tsCross
	}

	msgLocal = m.Payload
	sourceID := m.Source
	sourceMsg := m.Payload
	// hashMsg is the hash of sourceMsg
	hashMsg := cryptolib.GenHash(sourceMsg)
	sourceEpoch := int(m.Instance)
	GlobalTotalEpoch = m.Round

	p := fmt.Sprintf("[CROSS-#%v] cross handler, hashMsgSize:%v, totalEpoch:%v, startTS:%d", sourceEpoch, len(hashMsg), GlobalTotalEpoch, receivedStartTS)
	logging.PrintLog(verbose, logging.NormalLog, p)

	//tag = instance + msg hash
	epochByte := []byte(fmt.Sprintf("%v", sourceEpoch))
	quorum_tag := cryptolib.GenInstanceHash(epochByte, sourceMsg)
	quorum.Add(sourceID, string(quorum_tag), sourceMsg, quorum.PP)

	// flag, _ := GetEpochStatus(sourceEpoch)
	if quorum.CheckCrossSmallQuorum(string(quorum_tag), quorum.PP) {
		AddMessageTocrossMsgQueue(hashMsg, sourceEpoch)
		p := fmt.Sprintf("[CROSS-#%v] cross quorum achieved, add into queue(len:%v)", sourceEpoch, crossMsgQueue.Len())
		logging.PrintLog(verbose, logging.NormalLog, p)

		// StartRBC(sourceEpoch, hashMsg)
	}
}

func AddMessageTocrossMsgQueue(msg []byte, epoch int) {
	crossMsgQueue.AddItem(msg, epoch)
	p := fmt.Sprintf("[QUEUE-#%v] cross quorum achieved, add into queue(len:%v)", epoch, crossMsgQueue.Len())
	logging.PrintLog(verbose, logging.NormalLog, p)
}

func SetGlobalTotalEpoch(epoch int) {
	GlobalTotalEpoch = epoch
}

func ReadyStartRBC() {
	for {
		// p := fmt.Sprintf("check check")
		// logging.PrintLog(verbose, logging.ErrorLog, p)

		if crossMsgQueue.Len() > 0 {
			hashmsg, epoch, ok := crossMsgQueue.GetAndRemove()
			if !ok {
				p := fmt.Sprintf("[ERROR-CROSS] cannot get crossMsgQueue")
				logging.PrintLog(verbose, logging.ErrorLog, p)
				break
			}

			p := fmt.Sprintf("[CROSS-#%v get] queue check:%v", epoch, crossMsgQueue.Len())
			logging.PrintLog(verbose, logging.NormalLog, p)

			StartRBC(epoch, hashmsg)
		}
	}
}

var RBCTSStart sync.Map
var RBCTSEnd sync.Map

// Broadcast PROPOSE Message
func StartRBC(instanceid int, input []byte) {
	if verbose {
		_, ok := RBCTSStart.Load(instanceid)
		if !ok {
			RBCTSStart.Store(instanceid, utils.MakeTimestamp())
		}
	}

	if T_ra_rbc == 0 {
		T_ra_rbc = utils.MakeTimestamp()
		p := fmt.Sprintf("[DEBUG-RA-epoch#%v] Start RBC ts:%d, cross breakdown:%v", instanceid, T_ra_rbc, T_ra_rbc-receivedStartTS)
		logging.PrintLog(verbose, logging.EvaluationLog, p)
	}
	p := fmt.Sprintf("[RA-#%v] Start RBC propose send, msgSize:%v", instanceid, len(input))
	logging.PrintLog(verbose, logging.NormalLog, p)

	tEpoch = GlobalTotalEpoch

	// cur_epoch_status, _ := GetEpochStatus(instanceid)
	// localOP.Insert(instanceid, input)

	// identify the current status is send, if yes send the propose message
	// if cur_epoch_status == int(STATUS_CROSS) {
	p1 := fmt.Sprintf("[RA-PROPOSE-epoch#%v] Starting sending PROPOSE msg \n", instanceid)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	// use hash value of input as payload in the message

	// direct print the latency of hash calculation
	// start := utils.MakeTimestamp()
	inputOP = cryptolib.GenHash(input)
	// end := utils.MakeTimestamp()
	// p2 := fmt.Sprintf("[RA-PROPOSE-epoch#%v] Hash calculation latency:%d", instanceid, end-start)
	// logging.PrintLog(, logging.EvaluationLog, p2)

	msg := message.ReplicaMessage{
		Mtype:    message.RBC_SEND,
		Instance: instanceid,     //epoch number
		Source:   id,             //sender id
		Payload:  inputOP,        //msg = op
		Epoch:    rbcEpoch.Get(), //env epoch for rbc
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize RBC message")
	}
	sender.MACBroadcast(msgbyte, message.RBC)
	UpdateEpochStatus(instanceid, STATUS_CROSS)
}

// Handle PROPOSE and Send ECHO
func HandleSend(m message.ReplicaMessage) {
	// result, exist := rstatus.Get(m.Instance)
	// if exist && result {
	// 	return
	// }

	p := fmt.Sprintf("[RA-#%v] Start RBC propose handler", m.Instance)
	logging.PrintLog(verbose, logging.NormalLog, p)

	cur_epoch_status, exist := GetEpochStatus(m.Instance)
	if exist && cur_epoch_status >= int(STATUS_SEND) {
		return
	}

	if exist && cur_epoch_status == int(STATUS_CROSS) {
		UpdateEpochStatus(m.Instance, STATUS_SEND)
		p1 := fmt.Sprintf("[RBC-epoch#%v] Handling SEND message from node %v", m.Instance, m.Source)
		logging.PrintLog(verbose, logging.NormalLog, p1)
		// instancestatus.Insert(m.Instance, int(STATUS_SEND))

		msg := m
		msg.Source = id
		msg.Mtype = message.RBC_ECHO
		msg.Hash = cryptolib.GenInstanceHash(utils.IntToBytes(m.Instance), m.Payload)

		msgbyte, err := msg.Serialize()
		if err != nil {
			log.Fatalf("failed to serialize echo message")
		}

		p2 := fmt.Sprintf("[RA-#%v] Start RBC ECHO send", m.Instance)
		logging.PrintLog(verbose, logging.NormalLog, p2)

		sender.MACBroadcast(msgbyte, message.RBC)

	}

	// if !received.IsTrue(m.Instance) {
	// 	receivedReq.Insert(m.Instance, m.Payload)
	// 	received.AddItem(m.Instance)
	// 	receivedSig.Insert(m.Instance, m.Sig)
	// }

	// v, exist := GetEpochStatus(m.Instance)
	// if exist && v >= int(STATUS_SEND) {
	// 	SendReady(m)
	// }
	// if exist && v == int(STATUS_READY) {
	// 	Deliver(m.Instance)
	// }
}

// Handle ECHO
func HandleEcho(m message.ReplicaMessage) {
	// result, exist := rstatus.Get(m.Instance)
	// if exist && result {
	// 	return
	// }

	p := fmt.Sprintf("[RA-#%v] ECHO Handle", m.Instance)
	logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.CM)

	// if quorum.CheckQuorum(hash, quorum.CM) {
	// 	UpdateEpochStatus(m.Instance, STATUS_ECHO)
	// 	// if !received.IsTrue(m.Instance) {
	// 	// 	receivedReq.Insert(m.Instance, m.Payload)
	// 	// 	received.AddItem(m.Instance)
	// 	// 	receivedSig.Insert(m.Instance, m.Sig)
	// 	p := fmt.Sprintf("[RA-#%v] Start RBC ECHO handler", m.Instance)
	// 	logging.PrintLog(verbose, logging.NormalLog, p)
	// 	// }
	// 	SendReady(m)
	// }

	echotag, _ := GetEpochStatus(m.Instance)
	if quorum.CheckQuorum(hash, quorum.CM) && echotag == int(STATUS_SEND) {
		UpdateEpochStatus(m.Instance, STATUS_ECHO)

		p1 := fmt.Sprintf("[RA-#%v] meet quorum and Send Ready", m.Instance)
		logging.PrintLog(verbose, logging.NormalLog, p1)

		SendReady(m)
	}
}

func SendReady(m message.ReplicaMessage) {

	// p := fmt.Sprintf("[RA-#%v] Start RBC READY send", m.Instance)
	// logging.PrintLog(verbose, logging.NormalLog, p)

	stat, _ := GetEpochStatus(m.Instance)

	if stat > int(STATUS_ECHO) {
		return
	}

	// p := fmt.Sprintf("[RBC-#%v] Sending READY", m.Instance)
	// logging.PrintLog(verbose, logging.NormalLog, p)

	msg := m
	msg.Source = id
	msg.Mtype = message.RBC_READY
	msgbyte, err := msg.Serialize()
	if err != nil {
		log.Fatalf("failed to serialize ready message")
	}
	sender.MACBroadcast(msgbyte, message.RBC)

	//	else {
	//		// v, exist := cachestatus.Get(m.Instance)
	//		// elock.Unlock()
	//		// if exist && v == int(STATUS_READY) {
	//		epochStatus.Insert(m.Instance, int(STATUS_ECHO))
	//		// p1 := fmt.Sprintf("[RBC-epoch#%v-rbcEpoch#%v] Sending READY", m.Instance, rbcEpoch.Get())
	//		// logging.PrintLog(verbose, logging.NormalLog, p1)
	//		Deliver(m.Instance)
	//		// cachestatus.Insert(m.Instance, int(STATUS_ECHO))
	//		// }
	//	}
}

func HandleReady(m message.ReplicaMessage) {

	p := fmt.Sprintf("[RA-#%v] READY handle", m.Instance)
	logging.PrintLog(verbose, logging.NormalLog, p)

	stat, _ := GetEpochStatus(m.Instance)

	if stat > int(STATUS_ECHO) {
		return
	}
	// p := fmt.Sprintf("[RBC-READY-epoch#%v-rbcEpoch#%v] Handling ready message from node %v", m.Instance, rbcEpoch.Get(), m.Source)
	// logging.PrintLog(verbose, logging.NormalLog, p)

	hash := utils.BytesToString(m.Hash)
	quorum.Add(m.Source, hash, nil, quorum.FW)

	// p1 := fmt.Sprintf("[RBC-READY-epoch#%v-rbcEpoch#%v] Quorum size %v", m.Instance, rbcEpoch.Get(), quorum.CheckCurNum(hash, quorum.CM))
	// logging.PrintLog(verbose, logging.NormalLog, p1)

	tag, _ := GetEpochStatus(m.Instance)

	if quorum.CheckSmallQuorum(hash, quorum.FW) && tag == int(STATUS_SEND) {
		// if !received.IsTrue(m.Instance) {
		// 	receivedReq.Insert(m.Instance, m.Payload)
		// 	received.AddItem(m.Instance)
		// }

		p := fmt.Sprintf("[RA-#%v] handle ready but not send ready, send ready", m.Instance)
		logging.PrintLog(verbose, logging.NormalLog, p)
		SendReady(m)
	}

	if quorum.CheckQuorum(hash, quorum.CM) && tag == int(STATUS_ECHO) {
		// stat, _ := instancestatus.Get(m.Instance)
		// p2 := fmt.Sprintf("[RBC-READY-epoch#%v-rbcEpoch#%v] Meet Quorum to deliver, instancestatus=%v", m.Instance, rbcEpoch.Get(), stat)
		// logging.PrintLog(verbose, logging.NormalLog, p2)
		// instancestatus.Insert(m.Instance, int(STATUS_READY))
		UpdateEpochStatus(m.Instance, STATUS_READY)
		Deliver(m.Instance)
	}
}

var rbcDelivered utils.IntBoolMap

var deliveredEpochMap = make(map[int]bool)
var deliveredEpochLock sync.RWMutex

var tEpoch int
var xDelivered bool

func Deliver(instance int) {
	p1 := fmt.Sprintf("[RA-#%v] Start RBC Deliver", instance)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	sta, _ := GetEpochStatus(instance)

	if sta == int(STATUS_DELIVER) {
		return
	}

	UpdateEpochStatus(instance, STATUS_DELIVER)

	//检查deliveredEpochList是否有重复的epoch，如果没有，就添加
	if !ExistsInDeliveredEpochMap(instance) {
		AddToDeliveredEpochMap(instance)
		// p := fmt.Sprintf("[RBC-epoch#%v] Deliver ts:%v", instance, utils.MakeTimestamp())
		// logging.PrintLog(verbose, logging.EvaluationLog, p)
		if verbose {
			_, ok := RBCTSEnd.Load(instance)
			if !ok {
				RBCTSEnd.Store(instance, utils.MakeTimestamp())
			}

			ts_start_rbc, _ := RBCTSStart.Load(instance)
			ts_end_rbc, _ := RBCTSEnd.Load(instance)

			start, ok1 := ts_start_rbc.(int64)
			end, ok2 := ts_end_rbc.(int64)

			if ok1 && ok2 {
				duration := end - start

				p1 = fmt.Sprintf("[RA-#%v] RA duration:%d", instance, duration)
				logging.PrintLog(verbose, logging.EvaluationLog, p1)
			}
		}
	}

	if len(deliveredEpochMap) == tEpoch {
		xDelivered = true
	}

}

func IsXDelivered() bool {
	return xDelivered
}

func ReadTotalEpoch() int {
	return tEpoch
}

func ExistsInDeliveredEpochMap(epoch int) bool {
	deliveredEpochLock.RLock()
	defer deliveredEpochLock.RUnlock()
	_, exists := deliveredEpochMap[epoch]
	return exists
}

func AddToDeliveredEpochMap(epoch int) {
	deliveredEpochLock.Lock()
	defer deliveredEpochLock.Unlock()
	deliveredEpochMap[epoch] = true
}
