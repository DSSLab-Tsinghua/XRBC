// Implementation of CrossConsensus Sig mode

package consensus

import (
	"CrossRBC/src/broadcast/sig"
	"CrossRBC/src/config"
	"CrossRBC/src/logging"
	"CrossRBC/src/message"
	pb "CrossRBC/src/proto/proto/communication"

	// "CrossRBC/src/quorum"
	"CrossRBC/src/utils"
	"fmt"
	"time"
)

var sig_t1 int64
var sig_t2 int64
var localEpoch utils.IntValue

func StartCrossSig(data []byte, txNum int) {
	message_epoch := message.DeserializeClientSigRequest(data).R

	if message_epoch == 0 {
		sig_t1 = utils.MakeTimestamp()
	}

	epoch.Set(int(message_epoch))
	sig.SetEpoch(int(message_epoch))

	p := fmt.Sprintf("[Sig-Cross(consensus)] epoch:%v (from env), epoch:%v (from msg)", epoch.Get(), message_epoch)
	logging.PrintLog(true, logging.NormalLog, p)

	go MonitorSigStatus(int(message_epoch), txNum) //monitor

	sig.HandleSigMessage(data)
}

func StartCrossSigClient(data []byte) {
	message_epoch := message.DeserializeClientSigRequest(data).R
	p := fmt.Sprintf("[Sig-Cross(consensus)] epoch:%v (from env), epoch:%v (from msg)", epoch.Get(), message_epoch)
	logging.PrintLog(true, logging.NormalLog, p)

	// sig_t1 = utils.MakeTimestamp()

	e := epoch.Get()
	sig.SetEpoch(e)

	sig.HandleSigMessage(data)
}

func StartFetchSigClient(data []byte) {
	message_lr := message.DeserializeCrossFetchMessage(data).LR
	message_hr := message.DeserializeCrossFetchMessage(data).LastR
	p := fmt.Sprintf("[Sig-Cross(consensus)] epoch:%v (from env), lr-%v, hr%v", epoch.Get(), message_lr, message_hr)
	logging.PrintLog(true, logging.NormalLog, p)

	// sig_t1 = utils.MakeTimestamp()

	e := epoch.Get()
	sig.SetEpoch(e)

	sig.HandleSigMessage(data)
}

func InitSig() {
	InitStatus(n)
	localEpoch.Init()
}

func MonitorSigStatus(e int, txNum int) {
	if e < localEpoch.Get() {
		return
	}

	for {
		status, _ := sig.QueryFinalStatus(e)

		p := fmt.Sprintf("[Sig(consensus)] monitor sig status: epoch:%v, status check:%v (2-term,3-hs,4-delivered,5-ended)", e, status)
		logging.PrintLog(true, logging.NormalLog, p)

		if e < localEpoch.Get() || status > int(sig.STATUS_DELIVERED) {
			return
		}

		if status == int(sig.STATUS_TERMINATE) {
			sig.SetFinalStatus(e, int(sig.STATUS_HOTSTUFF))
			p2 := fmt.Sprintf("[Sig(consensus)] epoch:%v, status check:%v(=2), a-broadcast", e, status)
			logging.PrintLog(verbose, logging.NormalLog, p2)

			msg := sig.GetMsg(e)

			// change byte to rawmessage
			msg_rm := []pb.RawMessage{{Msg: msg}}

			height_HS := GetHSHeight()

			p := fmt.Sprintf("[Sig(consensus)] check hs height: id:%v, (epoch:height_HS):(%v:%v) status check:%v(=2), ", id, e, height_HS, status)
			logging.PrintLog(true, logging.NormalLog, p)

			if id == 0 && height_HS < e+1 {
				// start hotstuff
				crossABCStatus.Insert(e, int(STATUS_ABC))
				StartHotStuff(msg_rm, e+1)

				go EpochHTForSigDecideChecker(e)

			}
		} else if status == int(sig.STATUS_DELIVERED) {
			p3 := fmt.Sprintf("[Sig(consensus)] epoch:%v, status check:%v, x-deliver", e, status)
			logging.PrintLog(true, logging.NormalLog, p3)

			sig.SetFinalStatus(e, int(sig.STATUS_END))
			localEpoch.Increment()

			if e == txNum-1 {
				sig_t2 = utils.MakeTimestamp()

				lat := sig_t2 - sig_t1

				sig.SetLr(e)

				p := fmt.Sprintf("[Sig-Test Result] f:%v, numOfReq:%v: latency%v, throughput:%v, tps:%v", (config.FetchNumReplicas()-1)/3, txNum, lat, int64(txNum*1000)/(lat))
				logging.PrintLog(true, logging.NormalLog, p)

				p2 := fmt.Sprintf("%v, %v: %v, %v, %v", (config.FetchNumReplicas()-1)/3, txNum, lat, int64(txNum*1000)/(lat))
				logging.PrintLog(true, logging.EvaluationLog, p2)

				// curStatus.Set(READY)

				return
			}
		} else {
			time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
		}
	}
}

func EpochHTForSigDecideChecker(seq int) {
	if seq < localEpoch.Get() {
		return
	}
	// 检查当前 seq 是否已经达成共识
	for {
		status_hs := CrossABCStatus(seq)
		status_sig, _ := sig.QueryFinalStatus(seq)

		p := fmt.Sprintf("[EpochHTForSigDecideChecker] Epoch %d, localEpoch %v, hs status: %d(=3), sig status:%v(=3)", seq, localEpoch.Get(), status_hs, status_sig)
		logging.PrintLog(true, logging.NormalLog, p)

		//decided := CheckIfDecided(seq)
		//if decided && status == int(STATUS_COMMIT) {
		if status_hs == int(STATUS_COMMIT) {
			crossABCStatus.Insert(seq, int(STATUS_DELIVERED))
			sig.SetFinalStatus(seq, int(sig.STATUS_DELIVERED))

			p2 := fmt.Sprintf("[EpochHTForSigDecideChecker] Epoch %d has decided. localEpoch %v", seq, localEpoch.Get())
			logging.PrintLog(true, logging.NormalLog, p2)

			return
			// t_abc_end := utils.MakeTimestamp()
			// latency := t_abc_end - t_abc_start
			// log.Printf("[EpochDecideChecker] Epoch %d has decided. Latency: %d ms", seq, latency)
			// p := fmt.Sprintf("[EpochDecideChecker] Epoch %d has decided. Latency: %d ms", seq, latency)
			// logging.PrintLog(true, logging.NormalLog, p)
		} else {
			time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
		}
	}
}
