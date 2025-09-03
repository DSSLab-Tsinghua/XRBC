package consensus

import (
	"CrossRBC/src/logging"
	"CrossRBC/src/utils"
	"fmt"
	"sync"
)

var epoch utils.IntValue    //epoch number
var curStatus CurStatus     //status of cross msg
var repStatus CurStatus     //status of rep msg
var confirmStatus CurStatus //status of comfirm msg
var fetchStatus CurStatus   //status of fetch msg
var catchUpStatus CurStatus //status of catchup msg

var echoStatus CurStatus  //status of echo msg
var readyStatus CurStatus //status of ready msg

var output utils.ByteSet     //output set
var astatus utils.IntBoolMap //rbc status, if yes r-delivered
var bstatus utils.IntBoolMap //aba status
var fstatus utils.IntBoolMap

var avalues utils.IntBoolMap //aba outputvalues
var otherlock utils.IntValue

var elock sync.Mutex

var outputCount utils.IntValue ////number of decide 0 or 1
var outputSize utils.IntValue  //number of decide 1

func InitStatus(n int) {
	baseinstance = 0 //Set up baseinstance to avoid conflict
	output = *utils.NewByteSet()
	astatus.Init()
	avalues.Init()
	bstatus.Init()
	fstatus.Init()
	outputCount.Init()
	outputSize.Init()
	epoch.Increment()
	otherlock.Init()

	p := fmt.Sprintf("*************Starting epoch %v************", epoch.Get())
	logging.PrintLog(true, logging.NormalLog, p)
}

func InitCrossRAStatus() {
	baseinstance = 0 //Set up baseinstance to avoid conflict
	outputSize.Init()

	astatus.Init()
	curStatus.Init()
	echoStatus.Init()
	readyStatus.Init()
}

func InitCrossABCStatus() {
	//

}
