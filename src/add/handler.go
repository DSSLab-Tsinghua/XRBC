package add

import (
	"CrossRBC/src/broadcast/ra"
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	"sync"
	"time"

	"fmt"
	"log"
	// "github.com/klauspost/reedsolomon"
)

var id int64
var n int
var verbose bool
var totalEpoch int

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
			// 调用现有的消息处理逻辑
			switch msg.Mtype {
			case message.CrossAdd_Cross:
				p := fmt.Sprintf("[ADD-Cross] Received CROSS message(type:%v) from %v(group B)", msg.Mtype, msg.Source)
				logging.PrintLog(false, logging.NormalLog, p)
				HandleAddCross(msg)
			case message.CrossAdd_Share:
				p := fmt.Sprintf("[ADD-Share] Received SHARE message(type:%v) from %v(group B)", msg.Mtype, msg.Source)
				logging.PrintLog(false, logging.NormalLog, p)
				HandleShare(msg)
			case message.RBC_SEND:
				p := fmt.Sprintf("[RA-SEND] Received PROPOSE message(type:%v) from %v(group B)", msg.Mtype, msg.Source)
				logging.PrintLog(false, logging.NormalLog, p)
				ra.HandleSend(msg)
			case message.RBC_ECHO:
				p := fmt.Sprintf("[RA-ECHO] Received ECHO message(type:%v) from %v(group B)", msg.Mtype, msg.Source)
				logging.PrintLog(false, logging.NormalLog, p)
				ra.HandleEcho(msg)
			case message.RBC_READY:
				p := fmt.Sprintf("[RA-READY] Received READY message(type:%v) from %v(group B)", msg.Mtype, msg.Source)
				logging.PrintLog(false, logging.NormalLog, p)
				ra.HandleReady(msg)
			default:
				log.Printf("[ADD mode]message type #%v not supported", msg.Mtype)
			}
		}
	}
}

func AddMessageToBuffer(msg message.ReplicaMessage) {
	bufferLock.Lock()
	messageBuffer = append(messageBuffer, msg)
	bufferLock.Unlock()
}

func HandleADDMsg(inputMsg []byte) {
	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	// mtype := content.Mtype
	totalEpoch = content.Round

	// p := fmt.Sprintf("[RA-ADD] Received ADD message from %v(group B), type %v(1/46)", content.Source, mtype)
	// logging.PrintLog(verbose, logging.NormalLog, p)

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	AddMessageToBuffer(content)

	//log.Printf("handling message from %v, type %v", source, mtype)
	// switch mtype {
	// case message.CrossAdd_Cross:
	// 	p := fmt.Sprintf("[ADD-Cross] Received CROSS message(type:%v) from %v(group B)", mtype, content.Source)
	// 	logging.PrintLog(false, logging.NormalLog, p)
	// 	HandleAddCross(content)
	// case message.CrossAdd_Share:
	// 	p := fmt.Sprintf("[ADD-Share] Received SHARE message(type:%v) from %v(group B)", mtype, content.Source)
	// 	logging.PrintLog(false, logging.NormalLog, p)
	// 	HandleShare(content)
	// case message.RBC_SEND:
	// 	p := fmt.Sprintf("[RA-SEND] Received PROPOSE message(type:%v) from %v(group B)", mtype, content.Source)
	// 	logging.PrintLog(false, logging.NormalLog, p)
	// 	ra.HandleSend(content)
	// case message.RBC_ECHO:
	// 	p := fmt.Sprintf("[RA-ECHO] Received ECHO message(type:%v) from %v(group B)", mtype, content.Source)
	// 	logging.PrintLog(false, logging.NormalLog, p)
	// 	ra.HandleEcho(content)
	// case message.RBC_READY:
	// 	p := fmt.Sprintf("[RA-READY] Received READY message(type:%v) from %v(group B)", mtype, content.Source)
	// 	logging.PrintLog(false, logging.NormalLog, p)
	// 	ra.HandleReady(content)
	// default:
	// 	log.Printf("[ADD mode]message type #%v not supported", mtype)
	// }

}

func InitADD(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes
	verbose = ver
	tShareList.Init()
	// crossCompleteList.Init()
	tsADDStart = 0
	totalEpoch = 0
	tEpoch = 0
	// // disperseCompleteList.Init()
	// localM.Init()
	go BatchProcessMessages()

	log.Printf("[RA-ADD]RBC-ADD initialized")
}
