package consensus

import (
	"CrossRBC/src/communication/sender"
	"CrossRBC/src/config"
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/db"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/quorum"
	"CrossRBC/src/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

var view int
var viewMux sync.RWMutex
var leader bool
var endorser bool

// var rotatingTimer *time.Timer  // wcx: no use.
var timeoutBuffer quorum.INTBUFFER

const UintSize = 32 << (^uint(0) >> 32 & 1)

// ID of the leader according to the view number
func LeaderID(view int) int {
	return LeaderDyMem(view)
}

func LeaderDyMem(view int) int {
	if config.FetchNumReplicas() == 0 {
		time.Sleep(time.Duration(5) * time.Millisecond)
	}
	if config.FetchNumReplicas() == 0 {
		// logging.PrintLog(true, logging.ErrorLog, "Failed to get a correct leader according to the view number")
		return 0
	}
	return view % config.FetchNumReplicas()
}

func Endorser() bool {
	return endorser
}

// Check whether this node is the leader
func Leader() bool {
	return leader
}

// Set up leader status, returns true if this node is the leader
func SetLeader(input bool) {
	leader = input
}

// Return view number
func LocalView() int {
	return view
}

func InitView() {
	view = 0
}

// Set view number
func SetView(v int) {
	view = v
	viewInt := utils.IntValue{}
	viewInt.Set(view)

	db.PersistValue("view", &viewInt, db.PersistCritical)

	tmp, _ := utils.Int64ToInt(id)
	if LeaderID(v) == tmp {
		leader = true
	} else {
		leader = false
	}
}

func SetViewSigGroup(v int) {
	view = v
	viewInt := utils.IntValue{}
	viewInt.Set(view)

	db.PersistGroupSigValue("view", &viewInt, db.PersistCritical)

	tmp, _ := utils.Int64ToInt(id)
	if LeaderID(v) == tmp {
		leader = true
	} else {
		leader = false
	}
}

func StartRotatingTimer(v int) {
	rt := config.FetchRotatingTime()
	time.AfterFunc(time.Duration(rt)*time.Second, func() {
		StartViewChange(v) // no need for timeout message
		// TimeoutHandler(v) // start a timeout QC mechanism
	})
}

// For hotstuff
func TimeoutHandler(v int) {
	viewMux.Lock()
	defer viewMux.Unlock()
	log.Printf("hotstuff handler rotating timer expires in view %v", v)
	if config.PersistLevel() != int(db.NoPersist) {
		// if view number is persisted, timeout messages are not needed.
		StartViewChange(v)
		return
	}
	if v < LocalView() {
		return
	}

	// log.Printf("curStatus: %v", curStatus.Get())
	// log.Printf("v:%v, LocalView():%v", v, LocalView())
	curStatus.Set(VIEWCHANGE)

	// if no persist, we need to send and collect timeout msgs and qcs.
	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_TIMEOUT,
		Source: id,
		View:   v,
		TS:     utils.MakeTimestamp(), // no use
		Num:    quorum.NSize(),        // no use
	}
	msgbyte, err := msg.Serialize()
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[QCVCMessage Error] Not able to serialize the message")
		return
	}
	p := fmt.Sprintf("sending a timout message of view %d", v)
	logging.PrintLog(verbose, logging.NormalLog, p)
	request, _ := message.SerializeWithSignature(id, msgbyte)
	go HandleQCByteMsg(request)      // this message is first ``received'' by the node itself.
	sender.RBCByteBroadcast(msgbyte) // This func only casts hotstuff message
}

func HandleTimeoutMsg(content message.HotStuffMessage, vcm message.MessageWithSignature) { //For new leader to collect vc messages. Todo: double check VC rules @QC
	log.Printf("receive a timeout msg from replica %v for view %v", content.Source, content.View)
	viewMux.Lock()
	if content.View < LocalView() {
		viewMux.Unlock()
		return
	}
	viewMux.Unlock()

	bufferLock.Lock()
	hash := utils.BytesToString(cryptolib.GenHash(utils.IntToBytes(content.View)))
	out, _ := GetBufferContent("TQC"+hash, BUFFER)
	if out == PREPARED {
		bufferLock.Unlock()
		return
	}
	timeoutBuffer.InsertValue(content.View, content.Source, vcm)
	if timeoutBuffer.GetLen(content.View) >= quorum.QuorumSize() {
		UpdateBufferContent("TQC"+hash, PREPARED, BUFFER)
		bufferLock.Unlock()
		msg := message.HotStuffMessage{
			Mtype:  pb.MessageType_TQC,
			View:   content.View,
			Source: id,
			V:      timeoutBuffer.GetV(content.View),
		}

		msgbyte, err := msg.Serialize()
		if err != nil {
			p := fmt.Sprintf("[View Change Error] Not able to serialize TQC message: %v", err)
			logging.PrintLog(true, logging.ErrorLog, p)
			return
		}
		request, _ := message.SerializeWithSignature(id, msgbyte)
		go HandleQCByteMsg(request) // this message is first ``received'' by the node itself.
		sender.RBCByteBroadcast(msgbyte)
	} else {
		bufferLock.Unlock()
	}

}

func VerifyTQC(v int, tqc []message.MessageWithSignature) bool {
	if v < 0 {
		log.Printf("The TQC is for view -1.")
		return true
	}
	if len(tqc) < quorum.QuorumSize() {
		log.Printf("len(tqc):%v < quorum.QuorumSize():%v", len(tqc), quorum.QuorumSize())
		return false
	}
	ids := utils.NewSet()
	for i := 0; i < len(tqc); i++ {
		content := message.DeserializeHotStuffMessage(tqc[i].Msg)
		if ids.HasItem(content.Source) || content.View != v || content.Mtype != pb.MessageType_TIMEOUT {
			log.Printf("ids.HasItem(content.Source):%v, content.View != v:%v, content.Mtype != pb.MessageType_TIMEOUT:%v", ids.HasItem(content.Source), content.View != v, content.Mtype != pb.MessageType_TIMEOUT)
			return false
		}
		ids.AddItem(content.Source)
		if !cryptolib.VerifySig(content.Source, tqc[i].Msg, tqc[i].Sig) {
			p := fmt.Sprintf("signature not verified for timeout msg from %v", content.Source)
			logging.PrintLog(true, logging.ErrorLog, p)
			return false
		}
	}
	return true
}

// A replica change its view only at the moment when it receives a TQC.
func HandleTQCMsg(content message.HotStuffMessage) {
	log.Printf("receive a TQC msg from replica %v for view %v", content.Source, content.View)
	//Verify the received TQC
	if !VerifyTQC(content.View, content.V) {
		log.Printf("TQC from replica %v is not verified.", content.Source)
		return
	}

	viewMux.Lock()
	defer viewMux.Unlock()
	if content.View < LocalView() {
		return
	}

	StartViewChange(content.View)

	bufferLock.Lock()
	defer bufferLock.Unlock()
	hash := utils.BytesToString(cryptolib.GenHash(utils.IntToBytes(content.View)))
	out, _ := GetBufferContent("TQC"+hash, BUFFER)
	if out == PREPARED {
		return
	}

	timeoutBuffer.InsertV(content.View, content.V)
	UpdateBufferContent("TQC"+hash, PREPARED, BUFFER)
	msgbyte, err := content.Serialize()
	if err != nil {
		p := fmt.Sprintf("[View Change Error] Not able to serialize TQC message: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}
	// forward the TQC msg from another replica, so the Source of this msg is not 'me'.
	sender.RBCByteBroadcast(msgbyte)
}

// Start view change by sending a VIEWCHANGE message
// wcx: I modify this function to input the view which should be changed.
// I think a view change operation should be associated with a view (or a leader).
// So if this view has been changed, there is no need to do this view change operation.
func StartViewChange(v int) {
	if v < LocalView() {
		// v is at most equal to LocalView(), can not be larger.
		return
	}
	//if curStatus.Get() == VIEWCHANGE {
	//	return
	//}

	curStatus.Set(VIEWCHANGE)

	SetView(v + 1)
	//view = view + 1
	viewInt := utils.IntValue{}
	viewInt.Set(v + 1)
	db.PersistValue("view", &viewInt, db.PersistCritical)
	log.Printf("Starting view change to view %v", v+1)
	HotStuffStartVC()
	tmp, _ := utils.Int64ToInt(id)
	if LeaderID(v+1) != tmp {
		curStatus.Set(READY)
		if consensus == CrossABC {
			StartRotatingTimer(v + 1)
		}
	}
}

// This func can only be invoked in func StartViewChange.
func HotStuffStartVC() {
	log.Printf("hostuff start view change to view %v", LocalView())
	vcTime = utils.MakeTimestamp()

	curStatus.Set(VIEWCHANGE)

	msg := message.HotStuffMessage{
		Mtype:  pb.MessageType_VIEWCHANGE,
		Source: id,
		View:   LocalView(),
		TS:     utils.MakeTimestamp(),
		Num:    quorum.NSize(),
	}

	msg.Seq = curBlock.Height
	blockbyte, _ := curBlock.Serialize()
	msg.PreHash = blockbyte // it seems that msg.QC = blockbyte is more suitable

	awaitingBlocks.Init()
	db.PersistValue("awaitingBlocks", &awaitingBlocks, db.PersistAll)

	/* wcx: I think these block awaiting buffers should not be cleared at the start of a new view.
	awaitingDecision.Init()
	db.PersistValue("awaitingDecision", &awaitingDecision, db.PersistAll)
	awaitingDecisionCopy.Init()
	db.PersistValue("awaitingDecisionCopy", &awaitingDecisionCopy, db.PersistAll)
	*/

	msgbyte, err := msg.Serialize()
	if err != nil {
		logging.PrintLog(true, logging.ErrorLog, "[QCVCMessage Error] Not able to serialize the message")
		return
	}
	curLeader := LeaderID(LocalView())
	cl := utils.IntToInt64(curLeader)
	p := fmt.Sprintf("[QC] starting view change to view %d sending qc-vc to %d", LocalView(), cl)
	logging.PrintLog(verbose, logging.NormalLog, p)

	log.Printf("sending a vc message...")
	if cl == id {
		request, _ := message.SerializeWithSignature(id, msgbyte)
		go HandleQCByteMsg(request)
	}
	sender.SendToNode(msgbyte, cl, message.CrossABC)
}

func VerifyQC(qc message.QCBlock) bool {
	if qc.Hash == nil {
		return true
	}

	cons := config.Consensus()
	switch cons {
	case 103:
		if len(qc.QC) != len(qc.IDs) || len(qc.QC) < quorum.QuorumGroupBSize() {
			return false
		}

	default:
		if len(qc.QC) != len(qc.IDs) || len(qc.QC) < quorum.QuorumSize() {
			return false
		}
	}

	//log.Printf("--[%v] length of QC %v", qc.Height, len(qc.QC))

	//t1 := utils.MakeTimestamp()
	for i := 0; i < len(qc.QC); i++ {
		if !cryptolib.VerifySig(qc.IDs[i], qc.Hash, qc.QC[i]) {
			p := fmt.Sprintf("signature not verified for height %v, source %v", qc.Height, qc.IDs[i])
			logging.PrintLog(true, logging.ErrorLog, p)
			return true
		}
	}
	//t2 := utils.MakeTimestamp()
	//log.Printf("time for verify QC %v", t2-t1)
	//log.Printf("[%v] signature verified ", qc.Height)
	return true
}

func HandleQCVCMessage(content message.HotStuffMessage, vcm message.MessageWithSignature) { //For new leader to collect vc messages. Todo: double check VC rules @QC
	log.Printf("receive a VCQC msg from replica %v for new view %v", content.Source, content.View)
	viewMux.Lock()
	defer viewMux.Unlock()
	if content.View < LocalView() {
		log.Printf("[VCQC Error]: current view is %v, while content.View is %v", LocalView(), content.View)
		return
	}
	if content.View != LocalView() && content.View != LocalView()+1 {
		// content.View == LocalView()+1 might happen,
		//since this replica might not yet start view change.
		p := fmt.Sprintf("[QC] Handle view change to view %d from %v, local view %d", content.View, content.Source, LocalView())
		logging.PrintLog(true, logging.ErrorLog, p)
		return
	}
	tmp, _ := utils.Int64ToInt(id)
	if LeaderID(content.View) != tmp {
		log.Printf("[VCQC Error]:  QCVC is for view %d leader %v", content.View, LeaderID(content.View))
		return
	}

	cb := message.DeserializeQCBlock(content.PreHash)

	if !VerifyQC(cb) {
		log.Printf("qc not verified in QCVC %v", content.View)
		return
	}

	hash := utils.BytesToString(cryptolib.GenHash(utils.IntToBytes(content.View)))
	bufferLock.Lock()
	v, _ := GetBufferContent("VCQC"+hash, BUFFER)
	if v == PREPARED {
		log.Printf("[VCQC]: enough VCQC for view %v has been received", content.View)
		bufferLock.Unlock()
		return
	}
	quorum.AddToIntBuffer(content.View, content.Source, vcm, quorum.VC)
	if cb.Hash != nil && cb.Height >= curBlock.Height {
		// log.Printf("cb(%v) from %v is no lower than curBlock(%v)", cb.Height, content.Source, curBlock.Height)
		cblock.Lock()
		curBlock = cb
		db.PersistValue("curBlock", &curBlock, db.PersistAll)
		cblock.Unlock()
		UpdateSeq(cb.Height)
		vs, _ := vcAwaitingVotes.Get(content.View)
		if cb.Height > vs {
			vcAwaitingVotes.Delete(content.View)
			db.PersistValue("vcAwaitingVotes", &vcAwaitingVotes, db.PersistAll)
		}
	}

	if curStatus.Get() == READY {
		// This is for the case, where the new leader is still in the last view
		//and the timeout has not happened at it.
		log.Printf("[VCQC]: receiving VCQC for view %v when READY", content.View)
		bufferLock.Unlock()
		return
	}

	if quorum.CheckIntQuorum(content.View, quorum.VC) {
		SetView(content.View)
		UpdateBufferContent("VCQC"+hash, PREPARED, BUFFER)

		curStatus.Set(READY)
		//timer.Stop()
		//HandleCachedMsg()
		go RequestMonitor(LocalView())
	}
	bufferLock.Unlock()
}

func StartQCNewView() {

	V := quorum.GetVCMsgs(view, quorum.VC)

	o := GetQCOpsfromV(V, false)

	msg := message.ViewChangeMessage{
		Mtype:  pb.MessageType_NEWVIEW,
		View:   view,
		Source: id,
		V:      quorum.GetVCMsgs(view, quorum.VC),
		O:      o,
	}

	msgbyte, err := msg.Serialize()
	if err != nil {
		p := fmt.Sprintf("[View Change Error] Not able to serialize NEW-VIEW message: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
	} else {
		sender.RBCByteBroadcast(msgbyte)
	}
	go RequestMonitor(LocalView())
	//storage.ClearInMemoryStoreVC(lastSeq, Leader()) //Clear in memory data for view
}

func GetQCOpsfromV(VV []message.MessageWithSignature, verify bool) map[int]message.MessageWithSignature {
	o := make(map[int]message.MessageWithSignature)
	min := 1<<(UintSize-1) - 1 // set min to inifinity
	max := 0

	for i := 0; i < len(VV); i++ {
		V := message.DeserializeViewChangeMessage(VV[i].Msg)
		pCer := V.P
		for k, v := range pCer {
			msgs := v.GetMsgs()[0]
			bl := message.DeserializeQCBlock(msgs)
			//log.Printf("k %v, height %v, hash %s", k, bl.Height, bl.Hash)
			_, exist := o[k]
			if exist {
				continue
			}
			if !VerifyQC(bl) {
				continue
			}
			if k < min {
				min = k
			}
			if k > max {
				max = k
			}

			tmpmsg := message.HotStuffMessage{
				Mtype:   pb.MessageType_QC,
				Seq:     k,
				Source:  id,
				View:    view,
				Hash:    bl.Hash,
				PreHash: msgs,
			}
			op := message.CreateMessageWithSig(tmpmsg)

			o[k] = op
		}

	}

	return o
}

func HandleQCNewView(rawmsg []byte) {
	vcm := message.DeserializeViewChangeMessage(rawmsg)

	nlID, _ := utils.Int64ToInt(id)
	if LeaderID(vcm.View) == nlID {
		p := fmt.Sprintf("[View Change] Replica %d becomes leader", id)
		logging.PrintLog(verbose, logging.NormalLog, p)
		leader = true
	} else {
		leader = false
	}

	curStatus.Set(READY)
	//timer.Stop()
	OPS := vcm.O
	for k, rm := range OPS {
		msg, err := rm.Serialize()
		if err != nil {
			p := fmt.Sprintf("[New View Error] Serialize the PRE-PREPARE message of OPS in the NEW-VIEW message failed: %v", err)
			logging.PrintLog(true, logging.ErrorLog, p)
		}

		p := fmt.Sprintf("[New View] handle PP from new view, seq = %d", k)
		logging.PrintLog(verbose, logging.NormalLog, p)

		go HandleQCByteMsg(msg)
	}
}
