package consensus

import (
	"CrossRBC/src/communication/sender"
	"CrossRBC/src/config"
	pb "CrossRBC/src/proto/proto/communication"

	"CrossRBC/src/cryptolib"
	"CrossRBC/src/cryptolib/reedsolomon"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	"CrossRBC/src/quorum"
	"CrossRBC/src/utils"

	"fmt"
	"log"
	"sync"
)

var addlock sync.Mutex

// var wlock sync.Mutex
var t int                          // global threshold
var localM utils.IntByteMap        // local message in group B, reconstructed by OEC
var tShareList utils.IntMulByteMap // share phase: Stores shares for each epoch, keyed by epoch number
var tsADDEnd int64
var htStart []int

var crossCompleteList utils.IntByteMap
var tsADDStart int64
var tEpoch int
var addtotalEpoch int

func ADDHandleADDMsg(inputMsg []byte) {
	tmp := message.DeserializeMessageWithSignature(inputMsg)
	input := tmp.Msg
	content := message.DeserializeReplicaMessage(input)
	mtype := content.Mtype
	if addtotalEpoch == 0 {
		addtotalEpoch = content.Round
	}

	p := fmt.Sprintf("[ADD] Received #%v ADD message", addtotalEpoch)
	logging.PrintLog(verbose, logging.NormalLog, p)

	if !cryptolib.VerifyMAC(content.Source, tmp.Msg, tmp.Sig) {
		log.Printf("[Authentication Error] The signature of rbc message has not been verified.")
		return
	}

	//log.Printf("handling message from %v, type %v", source, mtype)
	switch mtype {
	case message.CrossAdd_Cross:
		p := fmt.Sprintf("[ADD-Cross] Received %v CROSS message(type:%v) from %v(group B)", addtotalEpoch, mtype, content.Source)
		logging.PrintLog(verbose, logging.NormalLog, p)
		ADDHandleAddCross(content)
	case message.CrossAdd_Share:
		p := fmt.Sprintf("[ADD-Share] Received SHARE message(type:%v) from %v(group B)", mtype, content.Source)
		logging.PrintLog(verbose, logging.NormalLog, p)
		ADDHandleShare(content, addtotalEpoch)
	default:
		log.Printf("[ADD mode]message type #%v not supported", mtype)
	}

}

// Group A: Encode and Send Cross
// input: n-number of nodes, t-threshold, M-original msg
// output: shares
func ADDEncode(n int, thres int, M []byte) ([][]byte, bool) {
	coder, err := reedsolomon.NewRScoder(thres, n)
	if err != nil {
		panic(err)
	}

	if n == 0 || thres == 0 || M == nil {
		return nil, false
	}

	if thres > n {
		p := fmt.Sprintf("[ADD-Encode]threshold(%d) cannot nodes(%d)", thres, n)
		logging.PrintLog(verbose, logging.ErrorLog, p)
		return nil, false
	}

	shares := coder.Encode(M)

	return shares, true
}

// epoch 是要发几个cross message
func ADDSendCross(M []byte, epoch int) {
	addlock.Lock()
	defer addlock.Unlock()

	t1 := utils.MakeTimestamp()

	t = quorum.SQuorumSize()

	p := fmt.Sprintf("[ADD-Encode-#%v] Start ADD Encode, number=%v, threshold=%v, msize=%v", epoch, n, t, len(M))
	logging.PrintLog(verbose, logging.NormalLog, p)

	err := false
	disperseData, err := ADDEncode(n, t, M)
	if !err {
		log.Fatal("Fail to encode erasure coding fragments")
		return
	}

	// group b's id is from 4 to ..., and group a's id is from 0 to 3
	// mStar = disperseData[id-4]

	groupBNodelist := config.FetchBNodes()
	for i := 0; i < epoch; i++ {
		for j, destidStr := range groupBNodelist {
			msg := message.ReplicaMessage{
				Instance: i,
				Payload:  disperseData[j],
				Source:   id,
				Mtype:    message.CrossAdd_Cross,
				Round:    epoch,
			}
			msgbyte, _ := msg.Serialize()

			destidStr, _ := utils.StringToInt64(destidStr)

			sender.SendToNode(msgbyte, destidStr, message.CrossADD)

			p := fmt.Sprintf("[ADD-CROSS-#%v] sending CROSS messages to Node #%v in group B", i, destidStr)
			logging.PrintLog(verbose, logging.NormalLog, p)
		}
	}
	t2 := utils.MakeTimestamp()
	p2 := fmt.Sprintf("[DEBUG-ADD] GroupA Send lat:%v", t2-t1)
	logging.PrintLog(verbose, logging.EvaluationLog, p2)
}

// Group B: Handle Cross Message
func ADDHandleAddCross(m message.ReplicaMessage) {
	if t_abc_start == 0 {
		t_abc_start = utils.MakeTimestamp()
	}

	addlock.Lock()
	defer addlock.Unlock()

	ts := utils.MakeTimestamp()
	if tsADDStart == 0 {
		tsADDStart = ts
		p1 := fmt.Sprintf("[DEBUG-ADD] group B: Start ADD ts:%v", tsADDStart)
		logging.PrintLog(verbose, logging.EvaluationLog, p1)
	}

	sourceID := m.Source
	sourceMsg := m.Payload
	sourceEpoch := m.Instance
	sourceMsgType := m.Mtype
	tEpoch = m.Round

	//tag = epoch number
	if sourceMsgType == message.CrossAdd_Cross {
		quorum.Add(sourceID, string(sourceEpoch), sourceMsg, quorum.PP)
	}

	p := fmt.Sprintf("[ADD-CROSS-#%v] Handling CROSS message from Client#%v: quorum size %v(%v)", sourceEpoch, sourceID, quorum.CheckCurNum(string(sourceEpoch), quorum.PP), quorum.SQuorumSize())
	logging.PrintLog(verbose, logging.NormalLog, p)

	_, exist := crossCompleteList.Get(sourceEpoch)

	if quorum.CheckCrossSmallQuorum(string(sourceEpoch), quorum.PP) && !exist {
		crossCompleteList.Insert(sourceEpoch, sourceMsg)

		p := fmt.Sprintf("[ADD-CROSS-#%v] meet quorum and add to list(len:%v)", sourceEpoch, crossCompleteList.GetLen())
		logging.PrintLog(verbose, logging.NormalLog, p)

		ADDSendShare(sourceEpoch, sourceMsg)
	}
}

func ADDSendShare(epoch int, share []byte) {
	groupBNodelist := config.FetchBNodes()
	for _, destidStr := range groupBNodelist {
		msg := message.ReplicaMessage{
			Instance: epoch,
			Payload:  share,
			Source:   id,
			Mtype:    message.CrossAdd_Share,
		}
		msgbyte, _ := msg.Serialize()

		destidStr, _ := utils.StringToInt64(destidStr)

		sender.SendToNode(msgbyte, destidStr, message.CrossADD)

		p := fmt.Sprintf("[ADD-Share-#%v] sending SHARE messages to Node #%v in group B", epoch, destidStr)
		logging.PrintLog(verbose, logging.NormalLog, p)
	}
}

// Group B: Share phase
// 更新一个新shares集合 tShareList
// 收到每一个<Share, m*j>，如果是来自j的第一个的share，添加(j, m*j)到tShareList
// 如果收到的share

func ADDHandleShare(msg message.ReplicaMessage, totalEpoch int) {
	addlock.Lock()
	defer addlock.Unlock()

	// addtotalEpoch = totalEpoch
	p1 := fmt.Sprintf("[ADD] Handle share total epoch:#%v", addtotalEpoch)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	// Check if the instance is already delivered
	_, e := localM.Get(msg.Instance)
	if e {
		p := fmt.Sprintf("[ADD-Share-#%v] local message already exists", msg.Instance)
		logging.PrintLog(verbose, logging.NormalLog, p)
		return
	}

	// Insert the share into tShareList
	key := msg.Instance
	index := int(msg.Source) - 4
	tShareList.Insert(key, index, msg.Payload)

	// Debug: Log the key and index used for insertion
	p := fmt.Sprintf("[DEBUG-HandleShare] Inserted into tShareList with Key: %v, Index: %v,", key, index)
	logging.PrintLog(verbose, logging.NormalLog, p)

	// Retrieve shares from tShareList
	shares, err := tShareList.GetByInt(key)
	if !err {
		p := fmt.Sprintf("[ADD-Share-#%v] Retrieved shares len:%v", key, len(shares))
		logging.PrintLog(verbose, logging.NormalLog, p)
	}
	// else {
	// 	p := fmt.Sprintf("[ADD-Share-#%v] Failed to get shares: %v", key, err)
	// 	logging.PrintLog(verbose, logging.ErrorLog, p)
	// 	return
	// }

	// Check if we have enough shares to proceed
	smallQuorum := quorum.SQuorumSize()

	if tShareList.GetLenByInt(key) > smallQuorum {
		// Start OEC
		p := fmt.Sprintf("[ADD-Share-#%v] %v shares reached and start OEC", key, tShareList.GetLenByInt(key))
		logging.PrintLog(verbose, logging.NormalLog, p)

		completeMessage, err := ADDDecode(key, shares)
		if !err {
			p := fmt.Sprintf("[ADD-Share-#%v]  Decode not succeed:%v", key, err)
			logging.PrintLog(verbose, logging.ErrorLog, p)
			return
		}

		localM.Insert(key, completeMessage)

		// p = fmt.Sprintf("[ADD-Share-#%v] Decoded, local M: %v", key, localM.GetAll())
		// logging.PrintLog(verbose, logging.NormalLog, p)

		hashMessage := cryptolib.GenHash(completeMessage)

		p = fmt.Sprintf("[ADD-Share-#%v] Start ABC in group b, hash: %x", key, hashMessage)
		logging.PrintLog(verbose, logging.NormalLog, p)

		// ra.SetStatus(key, int(ra.STATUS_SEND))
		tsADDEnd = utils.MakeTimestamp()
		consensusVersion := config.Consensus()

		switch consensusVersion {
		// case 100:
		// 	ra.StartRBC(key, hashMessage, tEpoch)
		case 101:
			// to-do: Start Hotstuff
			hsMsg := []pb.RawMessage{{Msg: hashMessage}}
			if id == 4 {
				if t_abc_abc == 0 {
					t_abc_abc = utils.MakeTimestamp()
					p1 := fmt.Sprintf("[DEBUG-ABC] group B: Start ABC ts:%d", t_abc_abc)
					logging.PrintLog(true, logging.EvaluationLog, p1)
				}
				StartHotStuff(hsMsg, key)
				// go ABCEpochDecideChecker()
			}

		}
	}
}

func ADDDecode(epoch int, shares [][]byte) ([]byte, bool) {
	// Decode the shares to reconstruct the original message
	number := quorum.NSize()
	threshold := quorum.SQuorumSize()

	coder, err := reedsolomon.NewRScoder(threshold, number)
	if err != nil {
		p := fmt.Sprintf("[DEBUG-Decode] Failed to initialize RScoder: %v", err)
		logging.PrintLog(verbose, logging.ErrorLog, p)
		return nil, false
	}

	// Debug: Log the shares before decoding
	p1 := fmt.Sprintf("[RA-ADD-Decode] Start decode, n: %v, thres:%v, shares(len:%v)", number, threshold, len(shares))
	logging.PrintLog(verbose, logging.NormalLog, p1)

	// Check for empty shares
	validShares := [][]byte{}
	for i, share := range shares {
		if len(share) == 0 {
			p := fmt.Sprintf("[Decode-WARN] Skipping empty share at index %v", i)
			logging.PrintLog(verbose, logging.ErrorLog, p)
			continue
		}
		validShares = append(validShares, share)
	}

	if len(validShares) < threshold {
		p := fmt.Sprintf("[DEBUG-Decode-#%v] Not enough valid shares: got %v, need at least %v", epoch, len(validShares), threshold)
		logging.PrintLog(verbose, logging.ErrorLog, p)
		return nil, false
	}

	// Decode the valid shares
	result, err := coder.Decode(validShares)
	if result == nil {
		p := fmt.Sprintf("[DEBUG-RA-Decode] decode err: %v", err)
		logging.PrintLog(verbose, logging.ErrorLog, p)
		return nil, false
	}

	// Remove the padding based on the last byte
	paddingLength := int(result[len(result)-1])
	if paddingLength > len(result) {
		p := fmt.Sprintf("[DEBUG-Decode] Invalid padding length: %v", paddingLength)
		logging.PrintLog(verbose, logging.ErrorLog, p)
		return nil, false
	}

	return result[:len(result)-paddingLength], true
}

func ADDReadADDStart() int64 {
	return tsADDStart
}

// func ADDHTStart() []int {
// 	addlock.Lock()
// 	defer addlock.Unlock()

// 	epochList := htStart
// 	htStart = []int{}
// 	p := fmt.Sprintf("[ADD-Share] HSList:%v ready to hotstuff and release:%v", epochList, htStart)
// 	logging.PrintLog(verbose, logging.NormalLog, p)
// 	return epochList
// }

// var id int64
// var n int
// var verbose bool

func InitADD(thisid int64, numNodes int, ver bool) {
	id = thisid
	n = numNodes //group b
	verbose = ver
	tShareList.Init()
	crossCompleteList.Init()
	tsADDStart = 0
	addtotalEpoch = 0
	tEpoch = 0
	// // disperseCompleteList.Init()
	localM.Init()

	log.Printf("[RA-ADD]RBC-ADD initialized")
}
