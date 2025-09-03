package consensus

import (
	//"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/vmihailenco/msgpack"

	"CrossRBC/src/config"
	"CrossRBC/src/db"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	"CrossRBC/src/quorum"
	"CrossRBC/src/utils"
)

var verbose bool      //verbose level
var batchOrWrite bool //batch mode

var id int64 //id of server
var iid int  //id in type int, start a RBC using it to instanceid
var errs error

var queue Queue        // cached client requests (CROSS)
var queueReply Queue   // cached client requests (CROSSREPLY)
var queueConfirm Queue // cached client requests (CROSSCONFIRM)

var TxNumber int //count total number of tx

var queueHead QueueHead // hash of the request that is in the fist place of the queue
var sleepTimerValue int // sleeptimer for the while loop that continues to monitor the queue or the request status
var consensus ConsensusType

// var rbcType	RbcType
var n int
var n_groupB int
var members []int
var t1 int64
var t_start int64
var baseinstance int

var batchSize int
var requestSize int

var MsgQueue Queue // record the consensus messages received so far.

func ExitEpoch() {
	t2 := utils.MakeTimestamp()
	if (t2 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	if outputSize.Get() == 0 {
		log.Printf("Finish zero instance!")
		return
	}
	log.Printf("*****epoch %v ends with %v, output size %v, latency %v ms, throughput %d, tps %d", epoch.Get(), output.Len(), outputSize.Get(), t2-t1, int64(outputSize.Get()*batchSize*1000)/(t2-t1), int64(quorum.QuorumSize()*batchSize*1000)/(t2-t1))
	p := fmt.Sprintf("%v %v %v %v %v %v %v", quorum.FSize(), batchSize, int64(outputSize.Get()*batchSize*1000)/(t2-t1), int64(quorum.QuorumSize()*batchSize*1000)/(t2-t1), t2-t1, requestSize, outputSize.Get())
	logging.PrintLog(true, logging.EvaluationLog, p)
}

func CaptureRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****RBC phase ends with %v ms", t3-t1)

}

func CaptureLastRBCLat() {
	t3 := utils.MakeTimestamp()
	if (t3 - t1) == 0 {
		log.Printf("Latancy is zero!")
		return
	}
	log.Printf("*****Final RBC phase ends with %v ms", t3-t1)

}

func RequestMonitor(v int) {
	// if consensus == CrossABC {
	// 	// wcx: Now we only test the view change of hotstuff.
	// 	for queue.IsEmpty() && LocalView() == 0 {
	// 		// wait until the first client sends its first request.
	// 		// then we start the rotatingTimer.
	// 		// In other words, we view the time the first request is received
	// 		//as the beginning of the system.
	// 		// log.Printf("wait for new requests in queue.")
	// 		continue
	// 	}
	// 	if config.IsViewChangeMode() {
	// 		StartRotatingTimer(v)
	// 	}
	// }

	for {
		switch consensus {
		case CrossRA:
			if !queue.IsEmpty() {
				// time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
				// batch := queue.GrabWtihMaxLenAndClear() //list of pb.RawMessage{Msg: request}
				clientReqRaw := queue.GrabWtihMaxLenAndClear()[0].Msg //list of pb.RawMessage{Msg: request}
				clientReq := message.DeserializeClientRequest(clientReqRaw)
				// batchSize = len(batch)                  //numbers of RawMessage in the "batch"
				// log.Printf("Handling [%v] batch requests\n", batchSize)

				txNum := int(clientReq.R)
				data := clientReq.OP

				StartProcessing(data, txNum)
			} else {
				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			}
		case CrossABC:
			// if id != 0 {
			// 	return
			// }

			if !queue.IsEmpty() {
				clientReqRaw := queue.GrabWtihMaxLenAndClear()[0].Msg //list of pb.RawMessage{Msg: request}
				clientReq := message.DeserializeClientRequest(clientReqRaw)

				txNum := int(clientReq.R)
				data := clientReq.OP

				StartProcessing(data, txNum)
			} else {
				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			}

		case CrossSig:
			// if curStatus is ready or repStatus is ready or confirmStatus is ready, and the queue is not empty, then we start processing.
			// if (curStatus.Get() == READY) && !queue.IsEmpty() {
			if !queue.IsEmpty() {

				// p10 := fmt.Sprintf("[Sig-(consensus)] consensus epoch:%v, checking queues: Cross %v, Reply %v, Confirm %v", epoch.Get(), curStatus.Get(), repStatus.Get(), confirmStatus.Get())
				// logging.PrintLog(true, logging.NormalLog, p10)

				// curStatus.Set(PROCESSING)

				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)

				batch := queue.GrabWithMaxLenAndClear() //rawmessage{msg: request}
				batchSize = len(batch)

				p := fmt.Sprintf("[Sig-(consensus)] received %v messages", batchSize)
				logging.PrintLog(verbose, logging.NormalLog, p)

				txNumber := config.FetchVolume()

				for i := 0; i < len(batch); i++ {
					p1 := fmt.Sprintf("[Sig-(consensus)] queue's %v req, LengthOfQueue:%v, TxNumber:%v", i, batchSize, txNumber)
					logging.PrintLog(verbose, logging.NormalLog, p1)

					StartProcessing(batch[i].Msg, txNumber)
				}
			} else {
				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			}

		case CrossSigGroup:
			if !queue.IsEmpty() {
				clientReqRaw := queue.GrabWtihMaxLenAndClear()

				// print clientReqRaw
				p1 := fmt.Sprintf("[SigGroup-(consensus)] clientReqRaw:%v", clientReqRaw)
				logging.PrintLog(verbose, logging.NormalLog, p1)

				clientReqRawFirst := clientReqRaw[0]
				clientReqRawFirstMsg := clientReqRawFirst.Msg //list of pb.RawMessage{Msg: request}
				clientReq := message.DeserializeClientRequest(clientReqRawFirstMsg)
				// batchSize = len(batch)                  //numbers of RawMessage in the "batch"
				// log.Printf("Handling [%v] batch requests\n", batchSize)

				txNum := int(clientReq.R)
				data := clientReq.OP
				p := fmt.Sprintf("[SigGroup-(consensus)] received %v messages", txNum)
				logging.PrintLog(verbose, logging.NormalLog, p)

				StartProcessing(data, txNum)
			} else {
				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			}
		case DeltaBA:
			if !queue.IsEmpty() {
				clientReqRaw := queue.GrabWtihMaxLenAndClear()[0].Msg //list of pb.RawMessage{Msg: request}
				clientReq := message.DeserializeClientRequest(clientReqRaw)

				txNum := int(clientReq.R)
				data := clientReq.OP

				StartProcessing(data, txNum)
			} else {
				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			}

		}
	}
}

// to-do: fetch request monitor
func SigClientRequestMonitor(v int) {
	for {
		switch consensus {
		case CrossSig:
			if (repStatus.Get() == READY || fetchStatus.Get() == READY) && !queue.IsEmpty() {
				p10 := fmt.Sprintf("[Sig(consensus)] epoch:%v, id:%v, checking queues: Reply %v, Fetch%v", epoch.Get(), id, repStatus.Get(), fetchStatus.Get())
				logging.PrintLog(true, logging.NormalLog, p10)

				if repStatus.Get() == READY {
					repStatus.Set(PROCESSING)
					p2 := fmt.Sprintf("[Sig-Reply(consensus)] epoch:%v, Set current status as processing", epoch.Get())
					logging.PrintLog(true, logging.NormalLog, p2)
				} else if fetchStatus.Get() == READY {
					fetchStatus.Set(PROCESSING)
					p4 := fmt.Sprintf("[Sig-Fetch(consensus)] epoch:%v, Set current status as processing", epoch.Get())
					logging.PrintLog(true, logging.NormalLog, p4)
				}

				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)

				batch := queue.GrabWithMaxLenAndClear() //rawmessage{msg: request}
				batchSize = len(batch)

				//handle reply and fetch message
				for i := 0; i < len(batch); i++ {
					StartCrossSigClient(batch[i].Msg)
				}
				return
			} else {
				time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
			}
		}
	}
}

func HandleRequest(request []byte, hash string) {
	batchOrWrite = false
	batchSize = 1
	queue.Append(request)
	TxNumber++
	db.PersistValue("queue", &queue, db.PersistAll)

	p := fmt.Sprintf("[Consensus-events] add tx to queue, txnumber:%v", TxNumber)
	logging.PrintLog(verbose, logging.NormalLog, p)
}

// CrossConsensus - sig mode: handle the client's request
// request = serialized whole cross message, hash is the hash of request
// func HandleSigRequest(request []byte, hash string) {
// 	//m ClientSigRequest
// 	m := message.DeserializeClientSigRequest(request)
// 	// if m.Type == pb.MessageType_CROSS {
// 	// 	TxNumber++
// 	// }
// 	batchOrWrite = false
// 	batchSize = 1
// 	requestSize = len(request)

// 	// go RequestMonitor(int(m.R))

// 	queue.Append(request)
// 	db.PersistValue("queue", &queue, db.PersistAll)

// 	p := fmt.Sprintf("[Sig(consensus)] add %v req to queue(size:%v), txNumber:%v", m.Type, queue.Length(), TxNumber)
// 	logging.PrintLog(true, logging.NormalLog, p)
// }

// func HandleSigReply(request []byte, hash string) {
// 	// m: request format
// 	m := message.DeserializeClientSigRequest(request)

// 	p := fmt.Sprintf("[Sig-%v(consensus)] nodeid%v, epoch:%v(from msg), Handling request %v", id, m.Type, m.R, m.OP)
// 	logging.PrintLog(verbose, logging.NormalLog, p)

// 	/*if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
// 		log.Printf("[Authentication Error] The signature of client request has not been verified.")
// 		return
// 	}*/

// 	batchSize = 1
// 	requestSize = len(request)

// 	//to-do:recheck view
// 	//SetView(int(m.R))

// 	queueReply.Append(request)
// 	// db.PersistValue("queue", &queue, db.PersistAll)
// }

func HandleBatchRequest(requests []byte) {
	batchOrWrite = true
	requestArr := DeserializeRequests(requests)
	//var hashes []string
	Len := len(requestArr)
	TxNumber = Len * 4
	p := fmt.Sprintf("[CROSS(batch mode)-Consensus]Server %v received %v msg", id, Len)
	logging.PrintLog(true, logging.NormalLog, p)

	//log.Printf("Handling batch requests with len %v\n",Len)
	//for i:=0;i<Len;i++{
	//	hashes = append(hashes,string(cryptolib.GenHash(requestArr[i])))
	//}
	//for i:=0;i<Len;i++{
	//	HandleRequest(requestArr[i],hashes[i])
	//}
	/*for i:=0;i<Len;i++{
		rawMessage := message.DeserializeMessageWithSignature(requestArr[i])
		m := message.DeserializeClientRequest(rawMessage.Msg)

		if !cryptolib.VerifySig(m.ID, rawMessage.Msg, rawMessage.Sig) {
			log.Printf("[Authentication Error] The signature of client logout request has not been verified.")
			return
		}
	}*/
	batchSize = Len                  //batch size
	requestSize = len(requestArr[0]) //1 msg size

	queue.AppendBatch(requestArr) //append serialized client request to queue (from the batch)
	db.PersistValue("queue", &queue, db.PersistAll)
}

func DeserializeRequests(input []byte) [][]byte {
	var requestArr [][]byte
	msgpack.Unmarshal(input, &requestArr)
	return requestArr
}
