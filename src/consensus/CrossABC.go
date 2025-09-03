package consensus

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	// "CrossRBC/src/broadcast/ra"
	"CrossRBC/src/communication"
	"CrossRBC/src/communication/sender"
	"CrossRBC/src/config"
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/db"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/quorum"
	"CrossRBC/src/utils"
)

// version: 0109 13:32

/*all the parameter for hotstuff protocols*/
var Sequence utils.IntValue  //Block Height
var curBlock message.QCBlock //recently voted block
var votedBlocks utils.IntByteMap
var lockedBlock message.QCBlock //locked block
var curHash utils.ByteValue     //current hash

// it seems that awaitingDecision and awaitingDecisionCopy are almost only written and not read.
// awaitingBlocks is read to fill the preHash and prepreHash,
// but will be cleared in view change. (if no view change, it will keep growing).
// So it seems that the first block of a new view will not get its preHash and prepreHash filled.
var awaitingBlocks utils.IntByteMap
var awaitingDecision utils.IntByteMap
var awaitingDecisionCopy utils.IntByteMap

var committedBlocks utils.IntByteMap // record the committed block history.

var vcAwaitingVotes utils.IntIntMap
var vcTime int64

var timer *time.Timer // timer for view changes

var cblock sync.Mutex
var lqcLock sync.RWMutex
var bufferLock sync.Mutex

// A replica will only send out messages of a step if it enters the previous step
var buffer utils.StringIntMap

// evaluation
var curOPS utils.IntValue
var totalOPS utils.IntValue
var beginTime int64

// var lastTime int64
var clock sync.Mutex
var genesisTime int64

var t_abc_start int64

// var t_abc_end int64
var t_abc_abc int64

// var ts_hsCommit int64
// var hsEpoch int //to control not to duplicate call starthotstuff

// design for CrossConsensus
var crossABCStatus utils.IntIntMap

// var crossABCStatusLock sync.RWMutex
var abcEpoch utils.IntValue
var hsHeight utils.IntValue
var deliveredHeight utils.IntValue

var awaitingBlocksLock sync.Mutex

// var crossEpoch int
// var startTag bool
var abcComfirmEpoch int

var crossMsg []byte

// var hotstuffMsg []byte
var hSStarted utils.IntBoolMap

// var hsStartedLock sync.Mutex

// var hsHeightLock sync.Mutex
// var deliveredHeightLock sync.Mutex

type ABCStatus int

// var statusLock sync.Mutex

var awaitingEpochsMapABC map[int]bool
var awaitingEpochsABC []int
var condABC = sync.NewCond(&muABC)
var muABC sync.Mutex
var groupCrossListABC utils.IntByteMap
var hstagABC bool

const (
	STATUS_IDLE      ABCStatus = 0 //Not start
	STATUS_CROSS     ABCStatus = 1 //Cross
	STATUS_ABC       ABCStatus = 2 //ABC
	STATUS_COMMIT    ABCStatus = 3 //Commit, complete ABC
	STATUS_DELIVERED ABCStatus = 4 //Delivered, complete ABC
	STATUS_ENDED     ABCStatus = 5 //End, complete ABC
)

func InitHotStuff(thisid int64) {
	buffer.Init()
	if !config.ParticipationChurn() {
		quorum.StartQuorum(n)
	} else {
		return
	}

	Sequence.Init()
	InitView()
	db.PersistValue("Sequence", &Sequence, db.PersistAll)
	votedBlocks.Init()
	db.PersistValue("votedBlocks", &votedBlocks, db.PersistAll)
	awaitingBlocks.Init()
	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
	awaitingDecision.Init()
	db.PersistValue("awaitingDecision", &awaitingDecision, db.PersistAll)
	awaitingDecisionCopy.Init()
	db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)
	committedBlocks.Init()
	db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
	SetView(0)

	timeoutBuffer.Init(n)
	// crossEpoch = 0
	abcComfirmEpoch = 0
	groupCrossListABC.Init()

	abcEpoch.Init()
	crossABCStatus.Init()
	crossABCStatus.Insert(abcEpoch.Get(), int(STATUS_IDLE))

	hsHeight.Init()
	deliveredHeight.Init()
	hSStarted.Init()
	// startTag = true
	flagABC = false
	hstagABC = true
	InitAwaitingEpochsMapABC()

	// crossQuorumList.Init()

	cryptolib.StartECDSA(thisid)
	// startTag = true

	//if config.EvalMode() > 0 {
	//	curOPS.Init()
	//	totalOPS.Init()
	//}

}

func InitAwaitingEpochsMapABC() {
	awaitingEpochsMapABC = make(map[int]bool)
	//长度是config.FetchVolume()，初始化为false
	for i := 0; i < config.FetchVolume(); i++ {
		awaitingEpochsMapABC[i] = false
	}
}

func InitGroupSigHotStuff(thisid int64) {
	buffer.Init()
	if !config.ParticipationChurn() {
		quorum.StartQuorum(n)
	} else {
		return
	}

	Sequence.Init()
	InitView()
	db.PersistGroupSigValue("Sequence", &Sequence, db.PersistAll)
	votedBlocks.Init()
	db.PersistGroupSigValue("votedBlocks", &votedBlocks, db.PersistAll)
	awaitingBlocks.Init()
	db.PersistGroupSigValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
	awaitingDecision.Init()
	db.PersistGroupSigValue("awaitingDecision", &awaitingDecision, db.PersistAll)
	awaitingDecisionCopy.Init()
	db.PersistGroupSigValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)
	committedBlocks.Init()
	db.PersistGroupSigValue("committedBlocks", &committedBlocks, db.PersistCritical)
	SetViewSigGroup(0)

	timeoutBuffer.Init(n)
	groupCrossListABC.Init()

	abcEpoch.Init()
	crossABCStatus.Init()
	crossABCStatus.Insert(abcEpoch.Get(), int(STATUS_IDLE))

	hsHeight.Init()
	deliveredHeight.Init()
	hSStarted.Init()

	// crossQuorumList.Init()

	cryptolib.StartECDSA(thisid)
	flagABC = false

	//if config.EvalMode() > 0 {
	//	curOPS.Init()
	//	totalOPS.Init()
	//}
}

// Design for CrossConsensus - ABC mode
func StartCrossABC(inputMsg []byte, txNum int) {
	//hsMsg 是 []pb.RawMessage，其中Msg是inputMsg
	// hsMsg := []pb.RawMessage{
	// 	{Msg: inputMsg},
	// }

	// hsMsg := pb.RawMessage{Msg: inputMsg}
	addVersion := config.FetchAdd()
	switch addVersion {
	case 1:
		ADDSendCross(inputMsg, txNum)
		// go ADDFinishChecker(hsMsg, txNum)
	case 0:
		XSendCross(inputMsg, txNum)
	default:
		p := fmt.Sprintf("[Cross-ABC] CANNOT send CROSS message, add version is %v", addVersion)
		logging.PrintLog(true, logging.ErrorLog, p)
	}

	p := fmt.Sprintf("***Start CrossConsensus - ABC mode***,numOfRequest:%v, length of input Msg: %v", txNum, len(inputMsg))
	logging.PrintLog(true, logging.NormalLog, p)

	// go CrossABCStatusMonitor()
	//to-do: go abc.CrossABCStatusMonitor()

	// HandleABCCrossBatch(inputMsg)
}

var flagABC bool

func ABCEpochDecideChecker() {
	t_abc_end := utils.MakeTimestamp()
	lat_abc_cross := t_abc_abc - receivedStartABCTS
	lat_abc_hs := t_abc_end - t_abc_abc
	f_num := (config.FetchNumBNodes() - 1) / 3
	addV := config.FetchAdd()

	switch addV {
	case 0:
		p := fmt.Sprintf("[ABC-TestResult] f:%v, TxNum:%v: Latency(ms):%d, Breakdown(ms,cross-abc):%d-%d, Throughput(tx/sec):%v", f_num, addtotalEpoch, lat_abc_hs+lat_abc_cross, lat_abc_cross, lat_abc_hs, int64(addtotalEpoch*1000)/int64(lat_abc_hs+lat_abc_cross))
		logging.PrintLog(true, logging.NormalLog, p)

		p1 := fmt.Sprintf("%v, %v, %d: %d-%d, %v", f_num, addtotalEpoch, lat_abc_hs+lat_abc_cross, lat_abc_cross, lat_abc_hs, int64(addtotalEpoch*1000)/int64(lat_abc_hs+lat_abc_cross))
		logging.PrintLog(true, logging.EvaluationLog, p1)
	case 1:
		p := fmt.Sprintf("[ABC-TestResult] f:%v, TxNum:%v: Latency(ms):%d, Latency(ms,add-abc):%d-%d, Throughput(tx/sec):%v", f_num, addtotalEpoch, lat_abc_hs+lat_abc_cross, lat_abc_cross, lat_abc_hs, int64(addtotalEpoch*1000)/int64(lat_abc_hs+lat_abc_cross))
		logging.PrintLog(true, logging.NormalLog, p)

		p1 := fmt.Sprintf("%v, %v, %d: %d-%d, %v", f_num, addtotalEpoch, lat_abc_hs+lat_abc_cross, lat_abc_cross, lat_abc_hs, int64(addtotalEpoch*1000)/int64(lat_abc_hs+lat_abc_cross))
		logging.PrintLog(true, logging.EvaluationLog, p1)
	}

}

// This func is invoked by the leader to broadcast a new proposal,
// so it may be invoked for many times by one node.
func StartHotStuff(batch []pb.RawMessage, seq int) {
	crossABCStatus.Insert(seq, int(STATUS_CROSS))

	p := fmt.Sprintf("[ABC-HotStuff] **Start HotStuff serverID#%v, height %d(total:%v)**", id, seq, addtotalEpoch-1)
	logging.PrintLog(verbose, logging.NormalLog, p)

	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_QC,
		Seq:    seq,
		Source: id,
		View:   LocalView(),
		OPS:    batch,
		TS:     utils.MakeTimestamp(),
		Num:    quorum.NSize(),
	}

	msg.QC = FetchBlockInfo(seq)
	tmphash := msg.GetMsgHash()
	msg.Hash = ObtainCurHash(tmphash, seq)
	curHash.Set(msg.Hash)
	db.PersistValue("curHash", &curHash, db.PersistAll)

	awaitingBlocks.Insert(seq, msg.Hash)
	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)

	p_d3 := fmt.Sprintf("[HotStuff-checkhash-seq#%v] awaiting blocks: %v", seq, msg.Hash)
	logging.PrintLog(verbose, logging.NormalLog, p_d3)

	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
	awaitingDecisionCopy.Insert(seq, msg.Hash)
	db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)

	log.Printf("proposing block with height %d, awaiting %d", seq, awaitingDecisionCopy.GetLen())

	msgbyte, _ := msg.Serialize()
	request, _ := message.SerializeWithSignature(id, msgbyte)
	go HandleQCByteMsg(request) // this message is first ``received'' by the node itself.
	sender.RBCByteBroadcast(msgbyte)
}

func CheckIfDecided(seq int) bool {
	// 检查当前 seq 是否已经达成共识的逻辑
	// 通过检查 committedBlocks 变量
	_, exist := committedBlocks.Get(seq)
	return exist
}

// Fetch the current block (can be used as the parent block).
// The seq inputted is only used to check whether this is the first block.
func FetchBlockInfo(seq int) []byte {
	if seq == 1 || curBlock.Height == 0 {
		return nil //return nil, representing initial block
	}
	var msg []byte
	var err error
	cblock.Lock()
	msg, err = curBlock.Serialize()
	cblock.Unlock()
	if err != nil {
		log.Printf("fail to serialize curblock")
		return []byte("")
	}

	return msg
}

func GetSeq() int {
	return Sequence.Get()
}

func Increment() int {
	Sequence.Increment()
	db.PersistValue("Sequence", &Sequence, db.PersistAll)
	return Sequence.Get()
}

func UpdateSeq(seq int) {
	if seq > GetSeq() {
		Sequence.Set(seq)
		db.PersistValue("Sequence", &Sequence, db.PersistAll)
		// log.Printf("update sequence to %v", Sequence.Get())
	}
}

func GenHashOfTwoVal(input1 []byte, input2 []byte) []byte {
	tmp := make([][]byte, 2, 2)
	tmp[0] = input1
	tmp[1] = input2
	b := bytes.Join(tmp, []byte(""))
	return cryptolib.GenHash(b)
}

func ObtainCurHash(input []byte, seq int) []byte {
	var result []byte
	bh := GenHashOfTwoVal(utils.IntToBytes(seq), input)
	ch := curHash.Get()
	result = GenHashOfTwoVal(ch, bh)
	return result
}

/*
Get data from buffer and cache. Used for consensus status
*/
func GetBufferContent(key string, btype TypeOfBuffer) (ConsensusStatus, bool) {
	switch btype {
	case BUFFER:
		v, exist := buffer.Get(key)
		return ConsensusStatus(v), exist
	}
	return 0, false
}

/*
Update buffer and cache. Used for consensus status
*/
func UpdateBufferContent(key string, value ConsensusStatus, btype TypeOfBuffer) {
	switch btype {
	case BUFFER:
		buffer.Insert(key, int(value))
	}
}

/*
Delete buffer and cache. Used for consensus status
*/
func DeleteBuffer(key string, btype TypeOfBuffer) {
	switch btype {
	case BUFFER:
		buffer.Delete(key)
	}
}

func HandleQCByteMsg(inputMsg []byte) {
	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	// TODO: tmp.Sig should be verified.

	content := message.DeserializeHotStuffMessage(input)

	mtype := content.Mtype
	source := content.Source
	communication.SetLive(utils.Int64ToString(source))

	p := fmt.Sprintf("[QC] received a message from %v, type %v, view %v, seq %v", source, mtype, content.View, content.Seq)
	logging.PrintLog(verbose, logging.NormalLog, p)

	switch mtype {
	case pb.MessageType_QC:
		log.Println("Handle QCMsg")
		HandleNormalMsg(content)
	case pb.MessageType_QCREP:
		log.Println("Handle QCREPMsg")
		HandleNormalRepMsg(content)
	case pb.MessageType_TIMEOUT:
		HandleTimeoutMsg(content, tmp)
	case pb.MessageType_TQC:
		HandleTQCMsg(content)
	case pb.MessageType_VIEWCHANGE:
		HandleQCVCMessage(content, tmp)
	case pb.MessageType_NEWVIEW:
		HandleQCNewView(tmp.Msg)
	}

	// 插入到 awaitingBlocks
	awaitingBlocksLock.Lock()
	awaitingBlocks.Insert(content.Seq, content.Hash)
	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
	awaitingBlocksLock.Unlock()

	p1 := fmt.Sprintf("[QC-Handle] Inserted block with seq #%d into awaitingBlocks", content.Seq)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	// 检查是否可以按顺序 deliver
	for seq := deliveredHeight.Get() + 1; ; seq++ {
		awaitingBlocksLock.Lock()
		blockHash, exist := awaitingBlocks.Get(seq)
		if !exist || blockHash == nil {
			awaitingBlocksLock.Unlock()
			p := fmt.Sprintf("[QC-Deliver-Stop] No block found for seq #%d in awaitingBlocks", seq)
			logging.PrintLog(true, logging.ErrorLog, p)
			break
		}

		// 检查是否严格按顺序
		if seq != deliveredHeight.Get()+1 {
			awaitingBlocksLock.Unlock()
			p := fmt.Sprintf("[QC-Deliver-Error] Out-of-order block detected at seq #%d, expected #%d", seq, deliveredHeight.Get()+1)
			logging.PrintLog(true, logging.ErrorLog, p)
			break
		}

		// Deliver 当前 seq 的区块
		deliverBlock(seq, blockHash)
		awaitingBlocks.Delete(seq) // 删除已处理的区块
		db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
		awaitingBlocksLock.Unlock()
	}
}

func rank(block message.QCBlock, blocktwo message.QCBlock) int {
	//log.Printf("blockview %v, two view %v, height %v, %v", block.View, blocktwo.View, block.Height, blocktwo.Height)
	if block.View < blocktwo.View {
		return -1
	}
	if block.Height < blocktwo.Height {
		return -1
	} else if block.Height == blocktwo.Height {
		return 0
	}
	return 1
}

// wcx: TODO: blockinfo should be verified that its height value is correct. (from genisis to it)
func VerifyBlock(curSeq int, blockinfo message.QCBlock) bool {
	if curSeq == 1 {
		return true
	}

	if blockinfo.Height <= lockedBlock.Height {
		p := fmt.Sprintf("[ABC-VerifyBlock] Block height %d is less than or equal to locked block height %d", blockinfo.Height, lockedBlock.Height)
		logging.PrintLog(true, logging.ErrorLog, p)
		//Warning: is this wrong? should it be < and return false?
		return true
	}
	cons := config.Consensus()
	switch cons {
	case 103:
		if len(blockinfo.QC) < quorum.QuorumGroupBSize() { // blockinfo.View != LocalView()||
			return false
		}
	default:
		if len(blockinfo.QC) < quorum.QuorumSize() { // blockinfo.View != LocalView()||
			p := fmt.Sprintf("[VerifyBlock] Block height %d has insufficient QC signatures", blockinfo.Height)
			logging.PrintLog(true, logging.ErrorLog, p)
			return false
		}
	}

	if !VerifyQC(blockinfo) {
		p := fmt.Sprintf("[VerifyBlock] block signature %d not verified", blockinfo.Height)
		logging.PrintLog(true, logging.ErrorLog, p)
		return false
	}
	return true

}

func HandleNormalMsg(content message.HotStuffMessage) { //For replica to process proposals from the leader
	viewMux.RLock()
	defer viewMux.RUnlock()
	if content.View < LocalView() || curStatus.Get() == VIEWCHANGE {
		return
	}

	hash := ""
	source := content.Source

	hash = utils.BytesToString(content.Hash)
	// if config.EvalMode() > 0 {
	// 	evaluation(len(content.OPS), content.Seq)
	// }

	if vcTime > 0 {
		// this seems not to be ture in view 0,
		//since vcTime is not assigned a value in view 0.
		vcdTime := utils.MakeTimestamp()
		log.Printf("processing block height %v, %v ms", content.Seq, vcdTime-vcTime)
	}

	if content.OPS != nil {
		awaitingDecision.Insert(content.Seq, content.Hash)
		db.PersistValue("awaitingDecision", &awaitingDecision, db.PersistAll)
		//dTime := utils.MakeTimestamp()
		//diff,_ := utils.Int64ToInt(dTime - cTime)
		//log.Printf("[%v] ++latency-1 for QCM %v ms", content.Seq, diff)
		if !Leader() {
			awaitingDecisionCopy.Insert(content.Seq, content.Hash)
			db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)
		}
	}
	blockinfo := message.DeserializeQCBlock(content.QC)
	if !VerifyBlock(content.Seq, blockinfo) {
		log.Printf("[QC] HotStuff Block %d not verified", blockinfo.Height)
		p := fmt.Sprintf("[QC] HotStuff Block %d not verified", blockinfo.Height)
		logging.PrintLog(true, logging.ErrorLog, p)
		// return
	}
	/*if content.OPS != nil{
		dTime := utils.MakeTimestamp()
		diff,_ := utils.Int64ToInt(dTime - cTime)
		log.Printf("[%v] ++latency-2 for QCM %v ms", content.Seq, diff)
	}*/

	p := fmt.Sprintf("[QC] HotStuff processing QC proposed at height %d", content.Seq)
	logging.PrintLog(verbose, logging.NormalLog, p)

	go ProcessQCInfo(hash, blockinfo, content)

	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_QCREP,
		Source: id,
		View:   LocalView(),
		Hash:   content.Hash,
		Seq:    content.Seq,
	}

	log.Printf("[d-crossabc] received a proposal from %v, height %v", source, content.Seq)

	sig := cryptolib.GenSig(id, content.Hash)

	msg.Sig = sig // a signature for the voted block, not for the entire message

	if !cryptolib.VerifySig(id, content.Hash, msg.Sig) {
		p := fmt.Sprintf("Node %d can not verify its newly generated sig!", id)
		logging.PrintLog(true, logging.ErrorLog, p)
		// return
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[QCMessage Error] Not able to serialize the message")
		return
	}
	msgwithsig, _ := message.SerializeWithSignature(id, msgbyte)
	if Leader() {
		// the message is first received by the leader itself.
		go HandleQCByteMsg(msgwithsig)
	}
	go sender.SendToNode(msgbyte, source, message.HotStuff)
}

// it seems that this func is useless, since queueHead is not set to a value in another place.
func HandleQueue(ch string, ops []pb.RawMessage) {
	if Leader() {
		return
	}
	if queueHead.Get() != "" {
		//log.Printf("stop queue head %s", queueHead)
		queueHead.Set("")
		timer.Stop()
	}
}

func ProcessQCInfo(hash string, blockinfo message.QCBlock, content message.HotStuffMessage) {
	stat, _ := crossABCStatus.Get(content.Seq)
	p := fmt.Sprintf("[QCCommit-h#%v] status:%v(<3), check prehash:%v, preprehash:%v", content.Seq, stat, blockinfo.PreHash != nil, blockinfo.PrePreHash != nil)
	logging.PrintLog(verbose, logging.NormalLog, p)

	// 确保按顺序 deliver
	for seq := deliveredHeight.Get() + 1; ; seq++ {
		awaitingBlocksLock.Lock()
		blockHash, exist := awaitingBlocks.Get(seq)
		if !exist || blockHash == nil {
			awaitingBlocksLock.Unlock()
			p := fmt.Sprintf("[QC-Deliver-Stop] No block found for seq #%d in awaitingBlocks", seq)
			logging.PrintLog(true, logging.ErrorLog, p)
			break
		}

		// 检查是否严格按顺序
		if seq != deliveredHeight.Get()+1 {
			awaitingBlocksLock.Unlock()
			p := fmt.Sprintf("[QC-Deliver-Error] Out-of-order block detected at seq #%d, expected #%d", seq, deliveredHeight.Get()+1)
			logging.PrintLog(true, logging.ErrorLog, p)
			break
		}

		// Deliver 当前 seq 的区块
		deliverBlock(seq, blockHash)
		awaitingBlocks.Delete(seq) // 删除已处理的区块
		db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)
		awaitingBlocksLock.Unlock()
	}

	// 更新当前区块状态
	if blockinfo.PreHash != nil {
		if blockinfo.PrePreHash != nil {
			// deliver/commit block
			lqcLock.RLock()
			blockser, _ := lockedBlock.Serialize()
			committedBlocks.Insert(lockedBlock.Height, blockser)
			lqcLock.RUnlock()
			db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
			ts_tmp := utils.MakeTimestamp()
			p := fmt.Sprintf("[QC-Commit(ProcessQCInfo)] HotStuff commit block at height #%d,ts:%v", lockedBlock.Height, ts_tmp)
			logging.PrintLog(verbose, logging.NormalLog, p)

			// p1 := fmt.Sprintf("[DEBUG-ABC] group B: ABC Height:%v ts:%v", lockedBlock.Height, ts_tmp)
			// logging.PrintLog(verbose, logging.EvaluationLog, p1)

			crossABCStatus.Insert(lockedBlock.Height, 3)
		}
		lqcLock.Lock()
		lockedBlock = curBlock
		db.PersistValue("lockedBlock", &lockedBlock, db.PersistCritical)
		lqcLock.Unlock()
		votedBlocks.Delete(curBlock.Height)
		db.PersistValue("votedBlocks", &votedBlocks, db.PersistAll)
	}
	if !Leader() && blockinfo.Height > curBlock.Height {
		cblock.Lock()
		curBlock = blockinfo
		db.PersistValue("curBlock", &curBlock, db.PersistAll)
		cblock.Unlock()
		curHash.Set(curBlock.Hash)
		db.PersistValue("curHash", &curHash, db.PersistAll)
	}

	UpdateSeq(content.Seq)
}

func deliverBlock(seq int, blockHash []byte) {
	ts_tmp := utils.MakeTimestamp()
	p := fmt.Sprintf("[DEBUG-ABC] delivered height #%d, ts:%d", seq, ts_tmp)
	logging.PrintLog(true, logging.NormalLog, p)

	if seq <= deliveredHeight.Get() && addtotalEpoch != 1 {
		p := fmt.Sprintf("[DEBUG-ABC] Skipping already delivered height #%d", seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}

	crossABCStatus.Insert(seq, int(STATUS_DELIVERED))
	deliveredHeight.Set(seq)
	db.PersistValue("deliveredHeight", &deliveredHeight, db.PersistAll)

	p1 := fmt.Sprintf("[ABC-#%v] deliver totalepoch:%v", seq, addtotalEpoch)
	logging.PrintLog(true, logging.NormalLog, p1)

	if seq == addtotalEpoch-1 {
		ABCEpochDecideChecker()
	}
}

func ProcessQCInfoCheck(blockinfo message.QCBlock, content message.HotStuffMessage) {
	p := fmt.Sprintf("[QCCommitChecker-h#%v(total:%v)] check prehash:%v, preprehash: %v", content.Seq, addtotalEpoch, blockinfo.PreHash != nil, blockinfo.PrePreHash != nil)
	logging.PrintLog(verbose, logging.NormalLog, p)
	epoch_cur := blockinfo.Height - 1
	//如果是Height=1，2, QCRep达到共识时就会commit
	switch epoch_cur {
	case 1:
		lqcLock.RLock()
		blockser, _ := lockedBlock.Serialize()
		committedBlocks.Insert(blockinfo.Height, blockser)
		lqcLock.RUnlock()
		db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
		ts_tmp := utils.MakeTimestamp()
		p := fmt.Sprintf("[QC-Commit-auto1] HotStuff commit block at height %d, ts:%d", blockinfo.Height, ts_tmp)
		logging.PrintLog(verbose, logging.NormalLog, p)

		p1 := fmt.Sprintf("[DEBUG-ABC] group B: ABC Height:%v ts:%v", blockinfo.Height, ts_tmp)
		logging.PrintLog(true, logging.EvaluationLog, p1)

		// crossABCStatus.Insert(blockinfo.Height, 4)
		// s, _ := crossABCStatus.Get(blockinfo.Height)
		p2 := fmt.Sprintf("[QC-Commit-auto-h#%v] Epoch %v, crossABCStatus %v(=3)", blockinfo.Height, blockinfo.Height)
		logging.PrintLog(verbose, logging.NormalLog, p2)

		deliverBlock(blockinfo.Height, blockinfo.Hash)
	case 2:
		if blockinfo.PreHash != nil {
			// deliver/commit block
			lqcLock.RLock()
			blockser, _ := lockedBlock.Serialize()
			committedBlocks.Insert(blockinfo.Height, blockser)
			lqcLock.RUnlock()
			db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
			ts_tmp := utils.MakeTimestamp()
			p := fmt.Sprintf("[QC-Commit-auto2] HotStuff commit block at height %d,ts:%v", blockinfo.Height, ts_tmp)
			logging.PrintLog(verbose, logging.NormalLog, p)

			p1 := fmt.Sprintf("[DEBUG-ABC] group B: ABC Height:%v ts:%v", blockinfo.Height, ts_tmp)
			logging.PrintLog(true, logging.EvaluationLog, p1)

			crossABCStatus.Insert(epoch_cur, 4)
			s, _ := crossABCStatus.Get(epoch_cur)
			p2 := fmt.Sprintf("[QC-Commit-rep-h#%v] abcEpoch %v, crossABCStatus %v(=3)", blockinfo.Height, epoch_cur, s)
			logging.PrintLog(verbose, logging.NormalLog, p2)
		}
		lqcLock.Lock()
		lockedBlock = curBlock // why curBlock is exactly blockinfo's parent?
		db.PersistValue("lockedBlock", &lockedBlock, db.PersistCritical)
		lqcLock.Unlock()
		votedBlocks.Delete(curBlock.Height)
		db.PersistValue("votedBlocks", &votedBlocks, db.PersistAll)
	default:
		if blockinfo.PreHash != nil {
			if blockinfo.PrePreHash != nil {
				// deliver/commit block
				lqcLock.RLock()
				blockser, _ := lockedBlock.Serialize()
				committedBlocks.Insert(blockinfo.Height, blockser)
				lqcLock.RUnlock()
				db.PersistValue("committedBlocks", &committedBlocks, db.PersistCritical)
				// ts_tmp := utils.MakeTimestamp()
				p := fmt.Sprintf("[QC-Commit-rep] HotStuff commit block at height %d", blockinfo.Height)
				logging.PrintLog(verbose, logging.NormalLog, p)
				// p1 := fmt.Sprintf("[DEBUG-ABC] group B: ABC Height:%v ts:%v", blockinfo.Height, ts_tmp)
				// logging.PrintLog(verbose, logging.EvaluationLog, p1)

				// if blockinfo.Height == config.FetchVolume() {
				// 	ts_hsCommit = utils.MakeTimestamp()
				// }

				// crossABCStatus.Insert(blockinfo.Height, int(STATUS_DELIVERED))
				s, _ := crossABCStatus.Get(blockinfo.Height)
				p2 := fmt.Sprintf("[QC-Commit-rep-h#%v] Status Refreshed curStatus %v, crossABCStatus %v(=3)", blockinfo.Height, curStatus.Get(), s)
				logging.PrintLog(verbose, logging.NormalLog, p2)
			}
			lqcLock.Lock()
			lockedBlock = curBlock // why curBlock is exactly blockinfo's parent?
			db.PersistValue("lockedBlock", &lockedBlock, db.PersistCritical)
			lqcLock.Unlock()
			votedBlocks.Delete(curBlock.Height)
			db.PersistValue("votedBlocks", &votedBlocks, db.PersistAll)
		}
	}
}

func HandleNormalRepMsg(content message.HotStuffMessage) {
	viewMux.RLock()
	defer viewMux.RUnlock()
	if content.View < LocalView() || curStatus.Get() == VIEWCHANGE {
		return
	}

	h, exist := awaitingBlocks.Get(content.Seq)
	// the block of Seq must be the one 'I' proposed
	if exist && bytes.Compare(h, content.Hash) != 0 {
		p := fmt.Sprintf("[QCRep] hash not matching, height#%v, hash:%v", content.Seq, content.Hash)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}
	if !cryptolib.VerifySig(content.Source, content.Hash, content.Sig) {
		p := fmt.Sprintf("[QCRep] signature for QCRep with height %v not verified", content.Seq)
		logging.PrintLog(true, logging.ErrorLog, p)
		// return
	}
	hash := utils.BytesToString(content.Hash)

	bufferLock.Lock()
	defer bufferLock.Unlock()
	// check that if there has existed a prepare qc for the block of hash.
	v, _ := GetBufferContent("BLOCK"+hash, BUFFER)
	if v == PREPARED {
		// log.Printf("There has been a prepare QC for content.Seq: %v", content.Seq)
		return
	}

	quorum.Add(content.Source, hash, content.Sig, quorum.CM)

	curStatusCheck_tmp := curStatus.Get()
	p := fmt.Sprintf("[QCRep] received a QCRep from %v, height %v, quorum size %v, curStatus %v", content.Source, content.Seq, quorum.CheckCurNum(hash, quorum.CM), curStatusCheck_tmp)
	logging.PrintLog(verbose, logging.NormalLog, p)

	cons := config.GConsensus()
	tag := false

	switch cons {
	case 103:
		tag = quorum.CheckSigGroupBFullQuorum(hash, quorum.CM)
	default:
		tag = quorum.CheckQuorum(hash, quorum.CM)
	}

	if tag {
		p := fmt.Sprintf("[QCRep-h%v] quorum size %v checked, curStatus %v", content.Seq, quorum.CheckCurNum(hash, quorum.CM), curStatus.Get())
		logging.PrintLog(verbose, logging.NormalLog, p)

		UpdateBufferContent("BLOCK"+hash, PREPARED, BUFFER)

		cer_byte := quorum.FetchCer(hash)
		if cer_byte == nil {
			p := fmt.Sprintf("[QC] cannnot obtain certificate from cache for block %v", content.Seq)
			logging.PrintLog(verbose, logging.ErrorLog, p)
		}
		_, sigs, ids := message.DeserializeSignatures(cer_byte)

		qcblock := message.QCBlock{
			View:   LocalView(),
			Height: content.Seq,
			QC:     sigs,
			IDs:    ids,
			Hash:   content.Hash,
		}

		ph, any := awaitingBlocks.Get(content.Seq - 1)
		if any && ph != nil {
			qcblock.PreHash = ph
		}
		ph, any = awaitingBlocks.Get(content.Seq - 2)
		if any && ph != nil {
			qcblock.PrePreHash = ph
		}

		p_d2 := fmt.Sprintf("[QCRep-h#%v] block prehash%v, preprehash%v", content.Seq, qcblock.PreHash != nil, qcblock.PrePreHash != nil)
		logging.PrintLog(verbose, logging.NormalLog, p_d2)

		if content.Seq > curBlock.Height {
			cblock.Lock()
			curBlock = qcblock
			db.PersistValue("curBlock", &curBlock, db.PersistAll)
			cblock.Unlock()
		} else {
			log.Printf("New block's QC has Seq: %v <= curBlock.Height: %v", content.Seq, curBlock.Height)
		}
		curStatus.Set(READY)

		p_d1 := fmt.Sprintf("[QCRepCheck-h#%v] block prehash%v, preprehash%v", content.Seq, qcblock.PreHash != nil, qcblock.PrePreHash != nil)
		logging.PrintLog(verbose, logging.NormalLog, p_d1)

		go ProcessQCInfoCheck(qcblock, content)
	}

}

// Get the throughput of the system
// Get the throughput of the system
// func evaluation(lenOPS int, seq int) {
// 	if lenOPS > 1 {
// 		//log.Printf("[Replica] evaluation mode with %d ops", lenOPS)
// 		var p = fmt.Sprintf("[Replica] evaluation mode with %d ops", lenOPS)
// 		logging.PrintLog(verbose, logging.EvaluationLog, p)
// 	}

// 	clock.Lock()
// 	defer clock.Unlock()
// 	val := curOPS.Get()
// 	if seq == 1 {
// 		beginTime = utils.MakeTimestamp()
// 		lastTime = beginTime
// 		genesisTime = utils.MakeTimestamp()
// 	}

// 	tval := totalOPS.Get()
// 	lenOPS = lenOPS //* 5
// 	tval = tval + lenOPS
// 	totalOPS.Set(tval)

// 	// the logic is wrong.
// 	// Here, the throughput is coumputed by the number of operations in the new block
// 	//deviding the latency of the last block.
// 	// It can only be correct when the number of operations of every block is the same.
// 	// Besides, the 'latency' is not the commit latency.
// 	// The commit latency of block i should be latency(i+3)+latency(i+2)+latency(i+1)+
// 	//the time of the leader packing and sending the block i, where latency(i) is the evaluated value.
// 	if val+lenOPS >= config.MaxBatchSize() {
// 		curOPS.Set(0)
// 		var endTime = utils.MakeTimestamp()
// 		var throughput int
// 		lat, _ := utils.Int64ToInt(endTime - beginTime)
// 		if lat > 0 {
// 			throughput = 1000 * (val + lenOPS) / lat // tx/s
// 		}

// 		clockTime, _ := utils.Int64ToInt(utils.MakeTimestamp() - genesisTime)
// 		log.Printf("[Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. Throughput %d tx/s. ", tval, lenOPS, clockTime, seq, lat, throughput)
// 		var p = fmt.Sprintf("[Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. Throughput %d tx/s", tval, lenOPS, clockTime, seq, lat, throughput)
// 		logging.PrintLog(true, logging.EvaluationLog, p)
// 		beginTime = endTime
// 		lastTime = beginTime
// 	} else {
// 		curOPS.Set(val + lenOPS)
// 	}
// }

func GetHSHeight() int {
	return hsHeight.Get()
}

func CrossABCStatus(epoch int) int {
	v, _ := crossABCStatus.Get(epoch)
	return v
}

func CheckHSEnd(height int) bool {
	_, exist := committedBlocks.Get(height - 1)
	return exist
}
