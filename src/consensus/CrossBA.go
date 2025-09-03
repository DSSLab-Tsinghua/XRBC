// Implementation of CrossConsensus RA mode
package consensus

import (
	// "CrossRBC/src/add"
	// "CrossRBC/src/add"
	"CrossRBC/src/add"
	"CrossRBC/src/broadcast/ra"

	"CrossRBC/src/config"
	"CrossRBC/src/logging"
	"fmt"
	"time"

	// "CrossRBC/src/quorum"
	"CrossRBC/src/utils"
)

type TimeRecord struct {
	Ts       int64
	Recorded bool
}

var T_ra_end int64
var T_ra_start int64
var requestVolume int

// check if all txs in the queue is completed, and record total latency+throughput+tps, exit
func MonitorRBCStatus() {
	for {
		if ra.IsXDelivered() {
			T_ra_end = utils.MakeTimestamp()

			p1 := fmt.Sprintf("[DEBUG-RA Mode] End ts:%v", T_ra_end)
			logging.PrintLog(verbose, logging.EvaluationLog, p1)

			ts_rbc := ra.ReadRBCBreakdown()
			T_ra_start = ra.ReadRACrossStart()
			numEpoch := ra.ReadTotalEpoch()
			tsADDStart := add.ReadADDStart()

			num := len(config.FetchBNodes())
			f_num := (num - 1) / 3
			numA := len(config.FetchANodes())
			t_num := (numA - 1) / 3

			// numReq := 1
			addVersion := config.FetchAdd()
			switch addVersion {
			case 0:
				lat_cross := ts_rbc - T_ra_start
				lat_rbc := T_ra_end - ts_rbc

				p1 := fmt.Sprintf("[RA-TestResult] f:%v, t:%v, Epoch:%v, Latency(ms):%d, Breakdown(ms, cross-rbc):%d-%d, Throughput(tx/sec):%v", f_num, t_num, numEpoch, lat_cross+lat_rbc, lat_cross, lat_rbc, int64(numEpoch*1000)/int64(lat_cross+lat_rbc))
				logging.PrintLog(true, logging.NormalLog, p1)

				p2 := fmt.Sprintf("%v, %v, %v, %d: %d-%d, %v", f_num, t_num, numEpoch, lat_cross+lat_rbc, lat_cross, lat_rbc, int64(numEpoch*1000)/int64(lat_cross+lat_rbc))
				logging.PrintLog(true, logging.EvaluationLog, p2)

				p3 := fmt.Sprintf("checking start ts:%v", T_ra_start)
				logging.PrintLog(verbose, logging.EvaluationLog, p3)
				return
			case 1:
				lat_add := ts_rbc - tsADDStart
				lat_rbc := T_ra_end - ts_rbc

				p1 := fmt.Sprintf("[ADD+RA-TestResult] f:%v, Epoch:%v, Latency(ms):%d, Breakdown(ms, add-ra):%d-%d, Throughput(tx/sec):%d", f_num, numEpoch, lat_add+lat_rbc, lat_add, lat_rbc, int64(numEpoch*1000)/int64(lat_add+lat_rbc))
				logging.PrintLog(true, logging.NormalLog, p1)

				p2 := fmt.Sprintf("%v, %v, %v: %v-%v, %v", f_num, numEpoch, lat_add+lat_rbc, lat_add, lat_rbc, int64(numEpoch*1000)/int64(lat_add+lat_rbc))
				logging.PrintLog(true, logging.EvaluationLog, p2)

				p3 := fmt.Sprintf("checking start ts:%v", tsADDStart)
				logging.PrintLog(verbose, logging.EvaluationLog, p3)
				return
			default:
				p := fmt.Sprintf("[Cross-TestResult] CANNOT calculate result, add version:%v", addVersion)
				logging.PrintLog(true, logging.ErrorLog, p)
				return
			}
		} else {
			time.Sleep(time.Duration(sleepTimerValue) * time.Millisecond)
		}
	}
}

var rbcflag bool
var tsRAStart int64

func StartCrossBA(data []byte, txNum int) {
	addVersion := config.FetchAdd()
	switch addVersion {
	case 1:
		add.SendCross(data, txNum)
	case 0:
		ra.SendCross(data, txNum)
	default:
		p := fmt.Sprintf("[Cross-BA] CANNOT send CROSS message, add version is %v", addVersion)
		logging.PrintLog(true, logging.ErrorLog, p)
	}
}

func InitCrossRA() {
	rbcflag = false
	tsRAStart = 0
	InitCrossRAStatus()
	// ra.SetrbcEpoch(epoch.Get())
	go MonitorRBCStatus()
}
