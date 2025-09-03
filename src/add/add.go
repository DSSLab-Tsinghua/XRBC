package add

import (
	"CrossRBC/src/broadcast/ra"
	"CrossRBC/src/communication/sender"
	"CrossRBC/src/config"
	"time"

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

var elock sync.Mutex

// var wlock sync.Mutex

var t int // global threshold
// var localM utils.IntByteMap        // local message in group B, reconstructed by OEC
var tShareList utils.IntMulByteMap // share phase: Stores shares for each epoch, keyed by epoch number
var tsADDEnd int64

// var htStart []int

var crossCompleteList sync.Map
var tsADDStart int64
var tEpoch int

// Group A: Encode and Send Cross
// input: n-number of nodes, t-threshold, M-original msg
// output: shares
func Encode(n int, thres int, M []byte) ([][]byte, bool) {
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

func thresOEC() int {
	numOfBNode := config.FetchNumBNodes()
	quorumf := (numOfBNode - 1) / 3
	res := 2*quorumf + 1
	p := fmt.Sprintf("[ADD]threshold:%v", res)
	logging.PrintLog(verbose, logging.NormalLog, p)
	return res
}

// epoch 是要发几个cross message
func SendCross(M []byte, epoch int) {
	elock.Lock()
	defer elock.Unlock()

	if tsADDStart == 0 {
		tsADDStart = utils.MakeTimestamp()
		p1 := fmt.Sprintf("[DEBUG-ADD+RA] group A: Start ADD ts:%v", tsADDStart)
		logging.PrintLog(verbose, logging.EvaluationLog, p1)
	}

	t1 := utils.MakeTimestamp()

	t = thresOEC()

	p := fmt.Sprintf("[ADD-Encode-#%v] Start ADD Encode, number=%v, threshold=%v, msize=%v", epoch, n, t, len(M))
	logging.PrintLog(verbose, logging.NormalLog, p)

	err := false
	disperseData, err := Encode(n, t, M)
	if !err {
		log.Fatal("Fail to encode erasure coding fragments")
		return
	}

	groupBNodelist := config.FetchBNodes()
	for i := 0; i < epoch; i++ {
		for j, destidStr := range groupBNodelist {
			msg := message.ReplicaMessage{
				Instance: i,
				Payload:  disperseData[j], // share
				Source:   id,
				Mtype:    message.CrossAdd_Cross,
				Round:    epoch,
				TS:       tsADDStart,
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
	logging.PrintLog(true, logging.EvaluationLog, p2)
}

// Group B: Handle Cross Message

func HandleAddCross(m message.ReplicaMessage) {
	elock.Lock()
	defer elock.Unlock()

	p2 := fmt.Sprintf("[ADD-%v] handle add cross", m.Instance)
	logging.PrintLog(verbose, logging.NormalLog, p2)

	sourceID := m.Source
	sourceMsg := m.Payload
	sourceEpoch := m.Instance
	sourceMsgType := m.Mtype
	tEpoch = m.Round
	if tsADDStart == 0 {
		tsADDStart = m.TS
	}

	ra.SetGlobalTotalEpoch(tEpoch)

	//tag = epoch number
	if sourceMsgType == message.CrossAdd_Cross {
		quorum.Add(sourceID, string(sourceEpoch), sourceMsg, quorum.PP)
	}

	p := fmt.Sprintf("[ADD-CROSS-#%v] Handling CROSS message from Client#%v: quorum size %v(%v)", sourceEpoch, sourceID, quorum.CheckCurNum(string(sourceEpoch), quorum.PP), quorum.SQuorumSize())
	logging.PrintLog(verbose, logging.NormalLog, p)

	_, exist := crossCompleteList.Load(sourceEpoch)

	if quorum.CheckCrossSmallQuorum(string(sourceEpoch), quorum.PP) && !exist {
		crossCompleteList.Store(sourceEpoch, sourceMsg)

		p := fmt.Sprintf("[ADD-#%v] meet cross quorum and send share", sourceEpoch)
		logging.PrintLog(verbose, logging.NormalLog, p)

		SendShare(sourceEpoch, sourceMsg)
	}
}

func SendShare(epoch int, share []byte) {
	p2 := fmt.Sprintf("[ADD-%v] SHARE send", epoch)
	logging.PrintLog(verbose, logging.NormalLog, p2)

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

// only used when verbose is true
var ts_tmp_start_list sync.Map
var ts_tmp_end_list sync.Map
var decodeList utils.IntBoolMap

func HandleShare(msg message.ReplicaMessage) {
	elock.Lock()
	defer elock.Unlock()

	p2 := fmt.Sprintf("[ADD-%v] SHARE handle and OEC", msg.Instance)
	logging.PrintLog(verbose, logging.NormalLog, p2)

	if verbose {
		_, ok := ts_tmp_start_list.Load(msg.Instance)
		if !ok {
			ts_tmp_start_list.Store(msg.Instance, utils.MakeTimestamp())
		}
	}

	// Check if the instance is already delivered
	_, e := decodeList.Get(msg.Instance)
	if e {
		p := fmt.Sprintf("[ADD-Share-#%v] OEC has completed", msg.Instance)
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
	quorumDecode := thresOEC()

	// _, tag := decodeList.Load(key)

	if tShareList.GetLenByInt(key) == quorumDecode {
		// Start OEC
		p := fmt.Sprintf("[ADD-Share-#%v] %v shares reached and start OEC", key, tShareList.GetLenByInt(key))
		logging.PrintLog(verbose, logging.NormalLog, p)

		completeMessage, su := Decode(key, shares)

		if !su {
			p := fmt.Sprintf("[ADD-Share-#%v]  Decode not succeed:%v", key, err)
			logging.PrintLog(verbose, logging.ErrorLog, p)

			go OECAgain(key)
		}

		// decodeList.Store(key, true)

		// localM.Insert(key, completeMessage)

		// p = fmt.Sprintf("[ADD-Share-#%v] Decoded, local M: %v", key, localM.GetAll())
		// logging.PrintLog(verbose, logging.NormalLog, p)

		hashMessage := cryptolib.GenHash(completeMessage)

		tsADDEnd = utils.MakeTimestamp()

		if verbose {
			_, ok := ts_tmp_end_list.Load(msg.Instance)
			if !ok {
				ts_tmp_end_list.Store(msg.Instance, utils.MakeTimestamp())
			}

			ts_start_decode, _ := ts_tmp_start_list.Load(msg.Instance)
			ts_end_decode, _ := ts_tmp_end_list.Load(msg.Instance)

			start, ok1 := ts_start_decode.(int64)
			end, ok2 := ts_end_decode.(int64)

			if ok1 && ok2 {
				duration := end - start

				p = fmt.Sprintf("[ADD-#%v] Handle Share, ts:%d", key, duration)
				logging.PrintLog(verbose, logging.EvaluationLog, p)
			}
		}
		// ra.UpdateEpochStatus(key, ra.STATUS_CROSS)

		p = fmt.Sprintf("[ADD-Share-#%v] Start RA in group b", key)
		logging.PrintLog(verbose, logging.NormalLog, p)
		// ra.StartRBC(key, hashMessage)
		ra.AddMessageTocrossMsgQueue(hashMessage, key)

	}
}

func OECAgain(key int) {
	maxRetries := 5 // 最大重试次数
	for retries := 0; retries < maxRetries; retries++ {
		p := fmt.Sprintf("[ADD-Share-#%v] Retry OEC, attempt %d", key, retries+1)
		logging.PrintLog(verbose, logging.NormalLog, p)

		shares, _ := tShareList.GetByInt(key)
		completeMessage, su := Decode(key, shares)
		if su {
			hashMessage := cryptolib.GenHash(completeMessage)

			tsADDEnd = utils.MakeTimestamp()

			if verbose {
				_, ok := ts_tmp_end_list.Load(key)
				if !ok {
					ts_tmp_end_list.Store(key, utils.MakeTimestamp())
				}
			}

			p = fmt.Sprintf("[ADD-Share-#%v] Start RA in group b", key)
			logging.PrintLog(verbose, logging.NormalLog, p)
			// ra.StartRBC(key, hashMessage)
			ra.AddMessageTocrossMsgQueue(hashMessage, key)

			return
		}
		time.Sleep(1 * time.Second)
	}

	p := fmt.Sprintf("[ADD-Share-#%v] Retry OEC failed after %d attempts", key, maxRetries)
	logging.PrintLog(verbose, logging.ErrorLog, p)
}

func Decode(epoch int, shares [][]byte) ([]byte, bool) {
	// Decode the shares to reconstruct the original message
	number := config.FetchNumBNodes()
	threshold := thresOEC()

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

func ReadADDStart() int64 {
	return tsADDStart
}

// func HTStart() []int {
// 	elock.Lock()
// 	defer elock.Unlock()

// 	epochList := htStart
// 	htStart = []int{}
// 	p := fmt.Sprintf("[ADD-Share] HSList:%v ready to hotstuff and release:%v", epochList, htStart)
// 	logging.PrintLog(false, logging.NormalLog, p)
// 	return epochList
// }
