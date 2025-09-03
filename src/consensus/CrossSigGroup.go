package consensus

import (
	"CrossRBC/src/communication/sender"
	"CrossRBC/src/config"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/utils"

	"CrossRBC/src/cryptolib"
	"CrossRBC/src/quorum"

	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

var timestamp_siggroup_start int64
var timestamp_siggroup_end int64
var timestamp_siggroup_cross int64

var msg_local []byte
var reqNum int
var groupCrossList utils.IntByteMap
var confirmedEpoch int

var awaitingEpochs []int
var awaitingEpochsMap = make(map[int]bool)
var hstag bool

var mu sync.Mutex
var cond = sync.NewCond(&mu)

func ConfirmStatusMonitor() {
	for {
		mu.Lock()
		for len(awaitingEpochs) == 0 || awaitingEpochs[0] != confirmedEpoch {
			if len(awaitingEpochs) > 0 && awaitingEpochs[0] < confirmedEpoch {
				awaitingEpochs = awaitingEpochs[1:]
				continue
			}
			cond.Wait()
		}

		epoch := awaitingEpochs[0]
		awaitingEpochs = awaitingEpochs[1:]
		delete(awaitingEpochsMap, epoch)

		groupCrossList.Insert(epoch, msg_local)
		confirmedEpoch++
		p := fmt.Sprintf("[SigGroup-Confirm-epoch#%v] confirmedEpoch:%v(epoch+1), groupCrossListLen:%v(=%v)", epoch, confirmedEpoch, groupCrossList.GetLen(), reqNum)
		logging.PrintLog(verbose, logging.NormalLog, p)

		if groupCrossList.GetLen() == reqNum && hstag {
			p := fmt.Sprintf("*****[Sig-Group-Confirm-epoch#%v] Sig Group is done.*****", epoch)
			logging.PrintLog(true, logging.NormalLog, p)

			if id == 4 {
				msg := []pb.RawMessage{{Msg: msg_local}}
				timestamp_siggroup_cross = utils.MakeTimestamp()

				log.Printf("*****[SigGroup-Cross] Hotstuff Start ts:%d*****", timestamp_siggroup_start)
				p := fmt.Sprintf("*****[SigGroup-Cross] Hotstuff Start ts:%d*****", timestamp_siggroup_start)
				logging.PrintLog(true, logging.NormalLog, p)
				for i := 1; i < groupCrossList.GetLen()+1; i++ {
					StartHotStuff(msg, i)
				}
				mu.Unlock()
				return
			}
			mu.Unlock()
			return
		}
		mu.Unlock()
	}
}

var flag bool

func GroupSigStatusMonitor() {
	for {
		if CheckHSEnd(reqNum) && flag {
			flag = false
			timestamp_siggroup_end = utils.MakeTimestamp()
			log.Printf("*******[Sig-Group-Commit] Sig Group is x-delivered. endTs:%d*******", timestamp_siggroup_end)
			p := fmt.Sprintf("*******[Sig-Group-Commit] Sig Group is x-delivered. endTs:%d*******", timestamp_siggroup_end)
			logging.PrintLog(true, logging.NormalLog, p)

			lat_cross := timestamp_siggroup_cross - timestamp_siggroup_start
			lat_abc := timestamp_siggroup_end - timestamp_siggroup_cross
			TxNum := reqNum
			f_num := (len(config.FetchBNodes()) - 1) / 3
			p1 := fmt.Sprintf("[SigGroup-TestResult] f:%v, TxNum:%v: Latency(ms,cross-abc):%d-%d, Throughput(tx/sec):%v", f_num, TxNum, lat_cross, lat_abc, int64(TxNum*1000)/(lat_cross+lat_abc))
			logging.PrintLog(true, logging.NormalLog, p1)

			p2 := fmt.Sprintf("%v, %v: %v-%v, %v", f_num, TxNum, lat_cross, lat_abc, int64(TxNum*1000)/(lat_cross+lat_abc))
			logging.PrintLog(true, logging.EvaluationLog, p2)

			return
		} else {
			time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
		}
	}
}

func StartSigCross(data []byte, txNum int) {
	// ops_tmp := message.DeserializeRawOPS(data)
	// batch_tmp := ops_tmp.OPS
	// clientMessage := message.DeserializeClientRequest(batch_tmp[0].Msg)

	// msg := clientMessage.OP

	reqNum = txNum

	p := fmt.Sprintf("[Sig-Group(consensus)] Start Handling Client Request, num:%v", reqNum)
	logging.PrintLog(true, logging.NormalLog, p)

	if id == 0 {
		SendGroupCrossMessage(data, txNum)
	}
}

func SendGroupCrossMessage(msg []byte, numOfRequests int) {
	p := fmt.Sprintf("[SigGroup-SendCross] Start sending GroupCross msg")
	logging.PrintLog(verbose, logging.NormalLog, p)

	//Send cross message to all nodes in Group B
	for i := 0; i < numOfRequests; i++ {
		cross_message := message.SigGroupMessage{
			Type: message.CrossGroup_CROSS,
			ID:   int(id),
			OP:   msg,
			R:    i,
			E:    numOfRequests,
		}

		cross_message_ser, err := cross_message.Serialize()
		if err != nil {
			p := fmt.Sprintf("[SigGroup-SendCross-epoch#%v] Error: fail to serialize the cross message: %v", i, err)
			logging.PrintLog(true, logging.ErrorLog, p)
		}

		sender.TOBBroadcast(cross_message_ser, message.CrossSigGroup)
	}
}

func HandleGroupSigMessage(inputMsg []byte) {
	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeSigGroupMessage(input)
	mtype := content.Type

	switch mtype {
	case message.CrossGroup_CROSS:
		HandleGroupCross(content)
	case message.CrossGroup_REP:
		HandleGroupRep(content)
	case message.CrossGroup_CONFIRM:
		HandleGroupConfirm(content)
	default:
		log.Printf("[Sig-Group(consensus)] Message type %v not supported", mtype)
	}
}

var sTag bool

func HandleGroupCross(content message.SigGroupMessage) {
	p := fmt.Sprintf("[SigGroup-Cross-epoch#%v] Start Handling GroupCross Message, tag:%v", content.R, sTag)
	logging.PrintLog(verbose, logging.NormalLog, p)

	msg_local = content.OP

	reqNum = content.E

	if !sTag {
		sTag = true
		timestamp_siggroup_start = utils.MakeTimestamp()
		log.Printf("*****[SigGroup-Cross] Start ts:%d*****", timestamp_siggroup_start)
		p := fmt.Sprintf("*****[SigGroup-Cross] Start ts:%d*****", timestamp_siggroup_start)
		logging.PrintLog(true, logging.NormalLog, p)

	}

	SendGroupRepMessage(content)
}

var confirmTag utils.IntBoolMap

func HandleGroupRep(content message.SigGroupMessage) {
	p := fmt.Sprintf("[SigGroup-Rep-epoch#%v] Start Handling GroupRep Message", content.R)
	logging.PrintLog(verbose, logging.NormalLog, p)

	epoch := []byte(fmt.Sprintf("%v", content.R))
	quorum_tag := cryptolib.GenInstanceHash(epoch, content.Hash)

	quorum.Add(int64(content.ID), string(quorum_tag), content.Hash, quorum.PP)
	cflag, _ := confirmTag.Get(content.R)

	// quorumCur := quorum.CheckCurNum(string(quorum_tag), quorum.PP)

	// p1 := fmt.Sprintf("[SigGroup-Rep-epoch#%v] cflag %v, quorum:%v", content.R, cflag, quorumCur)
	// logging.PrintLog(verbose, logging.NormalLog, p1)

	if quorum.CheckSigGroupBQuorum(string(quorum_tag), quorum.PP) && !cflag {
		confirmTag.Insert(content.R, true)
		SendGroupConfirmMessage(content)
	}
}

func HandleGroupConfirm(content message.SigGroupMessage) {
	p := fmt.Sprintf("[SigGroup-Confirm-epoch#%v] Start Handling GroupConfirm Message", content.R)
	logging.PrintLog(verbose, logging.NormalLog, p)

	hash_op := cryptolib.GenHash(msg_local)

	if string(hash_op) == string(content.Hash) {
		mu.Lock()
		if !awaitingEpochsMap[int(content.R)] {
			awaitingEpochs = append(awaitingEpochs, int(content.R))
			awaitingEpochsMap[int(content.R)] = true
			sort.Ints(awaitingEpochs)
			cond.Signal()
		}
		mu.Unlock()
		p := fmt.Sprintf("[SigGroup-Confirm-epoch#%v] awaitingEpoch:%v", content.R, awaitingEpochs)
		logging.PrintLog(verbose, logging.NormalLog, p)
	}
}

func SendGroupRepMessage(content message.SigGroupMessage) {
	p := fmt.Sprintf("[SigGroup-SendRep-epoch#%v] Start sending GroupRep msg", content.R)
	logging.PrintLog(verbose, logging.NormalLog, p)
	hash_op := cryptolib.GenHash(content.OP)

	rep_message := message.SigGroupMessage{
		Type: message.CrossGroup_REP,
		ID:   int(id),
		Hash: hash_op,
		R:    content.R,
	}

	rep_message_ser, err := rep_message.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Sig-Group(consensus)] Error: fail to serialize the rep message: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
	}

	sender.ToAGroupResNode(rep_message_ser, int64(content.ID), message.CrossSigGroup)
}

func SendGroupConfirmMessage(content message.SigGroupMessage) {
	p := fmt.Sprintf("[SigGroup-SendConfirm-epoch#%v] Start sending GroupConfirm msg", content.R)
	logging.PrintLog(verbose, logging.NormalLog, p)

	confirm_message := message.SigGroupMessage{
		Type: message.CrossGroup_CONFIRM,
		ID:   int(id),
		R:    content.R,
		Hash: content.Hash,
	}

	confirm_message_ser, err := confirm_message.Serialize()
	if err != nil {
		p := fmt.Sprintf("[SigGroup-epoch#%v] Error: fail to serialize the confirm message: %v", content.R, err)
		logging.PrintLog(true, logging.ErrorLog, p)
	}

	sender.TOBBroadcast(confirm_message_ser, message.CrossSigGroup)
	sender.TOABroadcast(confirm_message_ser, message.CrossSigGroup)
}

func InitGroupSig(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes
	verbose = ver
	quorum.StartQuorum(n)
	cryptolib.StartECDSA(id)
	groupCrossList.Init()
	InitStatus(n)
	flag = true
	hstag = true

	confirmTag.Init()
	// InitConfirmTag()

	confirmedEpoch = 0
	reqNum = 0
	sTag = false

	sleepTimerValue = config.FetchSleepTimer()

	// 在初始化时启动 ConfirmStatusMonitor 和 GroupSigStatusMonitor
	go ConfirmStatusMonitor()
	go GroupSigStatusMonitor()
}

// func InitConfirmTag() {
// 	confirmTag.Init()
// 	// len := config.FetchVolume()
// 	for i := 0; i < len; i++ {
// 		confirmTag.Insert(i, true)
// 	}
// }
