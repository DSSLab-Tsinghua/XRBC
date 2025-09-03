package ra

import (
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	"CrossRBC/src/quorum"
	"CrossRBC/src/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

var id int64
var n int
var verbose bool
var epoch utils.IntValue
var rbcEpoch utils.IntValue

// var wrbc bool
var inputOP []byte
var raCrossTS int64

// var localOP utils.IntByteMap

func UpdateEpochStatus(instanceid int, status RBCStatus) {
	epochStatus.Store(instanceid, int(status))
	p := fmt.Sprintf("[RBC-epoch#%v] Updated status to %v", instanceid, status)
	logging.PrintLog(false, logging.NormalLog, p)
}

func GetEpochStatus(instance int) (int, bool) {
	status, ok := epochStatus.Load(instance)
	if !ok {
		return 0, false
	}
	return status.(int), true
}

func SetEpochStatus(instance int, status int) {
	epochStatus.Store(instance, status)
}

func HandleRBCMsg(inputMsg []byte) {

	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	// mtype := content.Mtype

	// p := fmt.Sprintf("[RA-CROSS] Received CROSS message from %v(group A), type %v(=1)", content.Source, mtype)
	// logging.PrintLog(verbose, logging.NormalLog, p)

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	AddMessageToBuffer(content)
}

var messageBuffer []message.ReplicaMessage
var bufferLock sync.Mutex

func BatchProcessMessages() {
	for {
		bufferLock.Lock()
		if len(messageBuffer) == 0 {
			bufferLock.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}

		batch := messageBuffer
		messageBuffer = nil
		bufferLock.Unlock()

		for _, msg := range batch {
			switch msg.Mtype {
			case message.RBC_ALL:
				HandleCross(msg)
			case message.RBC_SEND:
				HandleSend(msg)
			case message.RBC_ECHO:
				HandleEcho(msg)
			case message.RBC_READY:
				HandleReady(msg)
			default:
				log.Printf("[BatchProcess] Unsupported message type: %v", msg.Mtype)
			}
		}
	}
}

func AddMessageToBuffer(msg message.ReplicaMessage) {
	bufferLock.Lock()
	messageBuffer = append(messageBuffer, msg)
	bufferLock.Unlock()
}

func ReadRACrossStart() int64 {
	return receivedStartTS
}

func InitRBC(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	//log.Printf("ini rstatus %v",rstatus.GetAll())
	rstatus.Init()
	rstatus.Insert(epoch.Get(), false)
	epoch.Init()
	rbcEpoch.Init()
	// crossCompleteList.Init()
	// rbcSendTag = false
	rbcDelivered.Init()
	xDelivered = false

	// instancestatus.Init()
	// cachestatus.Init()
	receivedReq.Init()
	received.Init()
	receivedSig.Init()
	epochReadyNum.Init()
	tsRAStart = 0
	receivedStartTS = 0
	// checkedTxNum = 0
	tEpoch = 0
	T_ra_rbc = 0
	GlobalTotalEpoch = 0
	crossMsgQueue = *utils.NewEpochMsgQueue()
	EpochList.Init()

	// deliveredEpochList = []int{}

	// localOP.Init()
	go BatchProcessMessages()
	go func() {
		// log.Printf("[RA] ReadyStartRBC Goroutine started")
		ReadyStartRBC()
	}()
	// log.Printf("[RA]RBC initialized")

}
