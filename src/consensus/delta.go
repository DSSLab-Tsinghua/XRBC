package consensus

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

type DeltaStatus int

const (
	STATUS_PROPOSE DeltaStatus = 1
	STATUS_VOTE    DeltaStatus = 2
	Status_BA      DeltaStatus = 3
)

var deltaTotalEpoch int

var t_delta_start int64
var t_delta_end int64

var deltaProposeList sync.Map

func HandleDeltaMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype

	p := fmt.Sprintf("[Delta] Received message from %v(group A), type %v", content.Source, mtype)
	logging.PrintLog(verbose, logging.NormalLog, p)

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	//log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.RBC_ALL:
		HandleDeltaPropose(content)
	case message.CrossDelta_Vote:
		HandleDeltaVote(content)
	default:
		log.Printf("[Delta-Handler] message type #%v not supported", mtype)
	}

}

var proposeLock sync.Mutex

func SendDeltaPropose(data []byte, numReq int) {
	proposeLock.Lock()
	defer proposeLock.Unlock()

	if t_delta_start == 0 {
		t_delta_start = utils.MakeTimestamp()
		p1 := fmt.Sprintf("[DEBUG-Delta] group A: Send Propose ts: %d", t_delta_start)
		logging.PrintLog(verbose, logging.EvaluationLog, p1)
	}

	deltaTotalEpoch = numReq

	for i := 0; i < numReq; i++ {
		msg := message.ReplicaMessage{
			Instance: i,
			Payload:  data,
			Source:   id,
			Mtype:    message.RBC_ALL,
			Round:    numReq,
		}
		msgbyte, _ := msg.Serialize()

		sender.MACBroadcast(msgbyte, message.RBC)

		p := fmt.Sprintf("[Delta-#%v] group A: PROPOSE send to group A+B", i)
		logging.PrintLog(verbose, logging.NormalLog, p)
	}
}

func HandleDeltaPropose(m message.ReplicaMessage) {
	proposeLock.Lock()
	defer proposeLock.Unlock()

	p1 := fmt.Sprintf("[Delta-#%v] Vote Handle", m.Instance)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	_, exist := deltaProposeList.Load(m.Instance)
	if exist {
		p := fmt.Sprintf("[Delta-#%v] PROPOSE already handled", m.Instance)
		logging.PrintLog(verbose, logging.NormalLog, p)
		return
	}

	deltaProposeList.Store(m.Instance, true)

	// msgLocal = m.Payload
	sourceID := m.Source
	sourceMsg = m.Payload
	sourceEpoch := int(m.Instance)

	if deltaTotalEpoch == 0 {
		deltaTotalEpoch = m.Round
	}

	p := fmt.Sprintf("[Delta-#%v(%v)] PROPOSE handle from Client#%v", sourceEpoch, deltaTotalEpoch, sourceID)
	logging.PrintLog(verbose, logging.NormalLog, p)

	SendDeltaVote(sourceMsg, sourceEpoch)
}

var deltaVoteLock sync.Mutex
var deltaVoteList utils.IntBoolMap

func SendDeltaVote(data []byte, epoch int) {
	deltaVoteLock.Lock()
	defer deltaVoteLock.Unlock()

	msg := message.ReplicaMessage{
		Instance: epoch,
		Payload:  data,
		Source:   id,
		Mtype:    message.CrossDelta_Vote,
	}
	msgbyte, _ := msg.Serialize()

	sender.MACBroadcast(msgbyte, message.RBC)

	p := fmt.Sprintf("[Delta-#%v] VOTE send", epoch)
	logging.PrintLog(verbose, logging.NormalLog, p)
}

func HandleDeltaVote(m message.ReplicaMessage) {
	deltaVoteLock.Lock()
	defer deltaVoteLock.Unlock()

	suc, exist := deltaVoteList.Get(m.Instance)
	if suc && exist {
		p := fmt.Sprintf("[Delta-#%v] VOTE already handled", m.Instance)
		logging.PrintLog(verbose, logging.NormalLog, p)
		return
	}

	// msgLocal = m.Payload
	sourceID := m.Source
	sourceMsg = m.Payload
	sourceEpoch := int(m.Instance)

	p := fmt.Sprintf("[Delta-#%v(%v)] VOTE handle from Client#%v", sourceEpoch, deltaTotalEpoch, sourceID)
	logging.PrintLog(verbose, logging.NormalLog, p)

	epochByte := []byte(fmt.Sprintf("%v", sourceEpoch))
	quorum_tag := cryptolib.GenInstanceHash(epochByte, sourceMsg)
	quorum.Add(sourceID, string(quorum_tag), sourceMsg, quorum.PP)

	if quorum.CheckDeltaSmallQuorum(string(quorum_tag), quorum.PP) {
		deltaVoteList.Insert(m.Instance, true)
		StartBA(sourceEpoch)
	}
}

func StartBA(epoch int) {
	p1 := fmt.Sprintf("[Delta-#%v] StartBA", epoch)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	if deltaVoteList.GetLen() == deltaTotalEpoch {
		t_delta_end = utils.MakeTimestamp()
		p3 := fmt.Sprintf("[DEBUG-Delta] group A: delta ts: %d", t_delta_end)
		logging.PrintLog(verbose, logging.EvaluationLog, p3)

		f := (n - 1) / 3

		p1 := fmt.Sprintf("[Delta-TestResult] f:%v, Epoch:%v, Latency(ms):%d", f, deltaTotalEpoch, t_delta_end-t_delta_start)
		logging.PrintLog(true, logging.NormalLog, p1)

		p2 := fmt.Sprintf("%v, %v, %d", f, deltaTotalEpoch, t_delta_end-t_delta_start)
		logging.PrintLog(true, logging.EvaluationLog, p2)

	}

}

func InitDelta(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes
	verbose = ver
	quorum.StartDeltaQuorum(n)

	t_delta_start = 0
	t_delta_end = 0
	deltaTotalEpoch = 0

	deltaVoteList.Init()

	log.Printf("[Delta]Delta initialized. numOfNode:%v", n)

}
