/*
Verify whether a replica/client has received matching messages from sufficient number of replicas
*/
package quorum

import (
	"CrossRBC/src/config"
	"CrossRBC/src/message"
)

// Used for normal operation
var buffer BUFFER       //prepare certificate. Client uses it as reply checker.
var bufferc BUFFER      //commit certificate. Client API uses it as reply checker.
var bufferd BUFFER      //forward certificate.
var cer CERTIFICATE     //used for vcbc only. Store the set of signatures
var intbuffer INTBUFFER // view changes

var n int
var f int
var quorum int
var squorum int
var half int

type Step int32

const (
	PP Step = 0
	CM Step = 1
	FW Step = 2
	VC Step = 3
)

/*
Clear in-memory data for view changes.
*/
func ClearCer() {
	bufferd.Init()
	bufferc.Init()
	buffer.Init()
	cer.Init()
	intbuffer.Init(n)
}

func Add(id int64, hash string, msg []byte, step Step) {
	switch step {
	case PP:
		buffer.InsertValue(hash, id, msg, step)
	case CM:
		bufferc.InsertValue(hash, id, msg, step)
	case FW:
		bufferd.InsertValue(hash, id, msg, step)
	}
}

// add without whole msg
func AddWithoutMsg(id int64, hash string, step Step) {
	msgempty := []byte{}
	switch step {
	case PP:
		buffer.InsertValue(hash, id, msgempty, step)
	case CM:
		bufferc.InsertValue(hash, id, msgempty, step)
	case FW:
		bufferd.InsertValue(hash, id, msgempty, step)
	}
}

func GetBuffercList(key string) []int64 {
	_, exist := bufferc.Buffer[key]
	if exist {
		s := bufferc.Buffer[key]
		return s.SetList()
	} else {
		return []int64{}
	}
}

func CheckQuorum(input string, step Step) bool {
	switch step {
	case PP:
		return buffer.GetLen(input) >= quorum
	case CM:
		return bufferc.GetLen(input) >= quorum
	case FW:
		return bufferd.GetLen(input) >= quorum
	}

	return false
}

func CheckCurNum(input string, step Step) int {
	switch step {
	case PP:
		return buffer.GetLen(input)
	case CM:
		return bufferc.GetLen(input)
	case FW:
		return bufferd.GetLen(input)
	}
	return 0
}

func CheckEqualQuorum(input string, step Step) bool {
	switch step {
	case PP:
		return buffer.GetLen(input) == quorum
	case CM:
		return bufferc.GetLen(input) == quorum
	case FW:
		return bufferd.GetLen(input) == quorum
	}

	return false
}

func CheckSmallQuorum(input string, step Step) bool {
	switch step {
	case PP:
		return buffer.GetLen(input) >= squorum
	case CM:
		return bufferc.GetLen(input) >= squorum
	case FW:
		return bufferd.GetLen(input) >= squorum
	}

	return false
}

func CheckCrossSmallQuorum(input string, step Step) bool {
	smallQuorum := (len(config.FetchANodes())-1)/3 + 1
	switch step {
	case PP:
		return buffer.GetLen(input) >= smallQuorum
	case CM:
		return bufferc.GetLen(input) >= smallQuorum
	case FW:
		return bufferd.GetLen(input) >= smallQuorum
	}

	return false
}

func CheckDeltaSmallQuorum(input string, step Step) bool {
	smallQuorum := squorum
	switch step {
	case PP:
		return buffer.GetLen(input) >= smallQuorum
	}

	return false
}

func CheckSigGroupBQuorum(input string, step Step) bool {
	bNodeList := len(config.FetchBNodes())
	quorum_b_group := (bNodeList+2)/3 + 1

	//print bNodeList and  quorum_b_group
	// p1 := fmt.Sprintf("[SigGroup-quorum] bNodeList %v and  quorum_b_group %v", bNodeList, quorum_b_group)
	// logging.PrintLog(true, logging.NormalLog, p1)

	switch step {
	case PP:
		return buffer.GetLen(input) >= quorum_b_group
	case CM:
		return bufferc.GetLen(input) >= quorum_b_group
	case FW:
		return bufferd.GetLen(input) >= quorum_b_group
	}

	return false
}

func CheckSigGroupBFullQuorum(input string, step Step) bool {
	bNodeList := len(config.FetchBNodes())
	quorum_b_group := (2*bNodeList + 1) / 3
	switch step {
	case PP:
		return buffer.GetLen(input) >= quorum_b_group
	case CM:
		return bufferc.GetLen(input) >= quorum_b_group
	case FW:
		return bufferd.GetLen(input) >= quorum_b_group
	}

	return false
}

func CheckCrossQuorum(input string, step Step) bool {
	normalQuorum := 3
	switch step {
	case PP:
		return buffer.GetLen(input) >= normalQuorum
	case CM:
		return bufferc.GetLen(input) >= normalQuorum
	case FW:
		return bufferd.GetLen(input) >= normalQuorum
	}

	return false
}

func CheckOverSmallQuorum(input string) bool {
	return bufferc.GetLen(input) >= squorum
}

func CheckEqualSmallQuorum(input string) bool {
	return bufferc.GetLen(input) == squorum
}

func ClearBuffer(input string, step Step) {
	switch step {
	case PP:
		buffer.Clear(input)
	case CM:
		bufferc.Clear(input)
	case FW:
		bufferd.Clear(input)
	}
}

func ClearBufferPC(input string) {
	buffer.Clear(input)
	bufferc.Clear(input)
	bufferd.Clear(input)
	cer.Clear(input)
}

func QuorumSize() int {
	return quorum
}

func QuorumGroupBSize() int {
	q := len(config.FetchBNodes())
	return q
}

func HalfSize() int {
	return half
}

func SQuorumSize() int {
	return squorum
}

func FSize() int {
	return f
}

func NSize() int {
	return n
}

func SetQuorumSizes(num int) {
	n = num
	f = (n - 1) / 3
	quorum = (n + f + 1) / 2
	if (n+f+1)%2 > 0 {
		quorum += 1
	}
	squorum = f + 1
}

func CheckOverHalf(input string) bool {
	return bufferc.GetLen(input) >= half
}

func CheckHalf(input string) bool {
	return bufferc.GetLen(input) == half
}

func StartQuorum(num int) {
	// n = num
	n = num - 4
	f = (n - 1) / 3
	quorum = (n + f + 1) / 2
	if (n+f+1)%2 > 0 {
		quorum += 1
	}
	squorum = f + 1

	bufferd.Init()
	bufferc.Init()
	buffer.Init()
	cer.Init()

	// fmt.Printf("[Quorum Start] n:%v, f:%v, quorum:%v, squorum:%v\n", n, f, quorum, squorum)
}

func StartDeltaQuorum(num int) {
	// n = num
	n = num
	f = (n - 1) / 3
	quorum = (n + f + 1) / 2
	if (n+f+1)%2 > 0 {
		quorum += 1
	}
	squorum = f + 1

	bufferd.Init()
	bufferc.Init()
	buffer.Init()
	cer.Init()

	// fmt.Printf("[Quorum Start] n:%v, f:%v, quorum:%v, squorum:%v\n", n, f, quorum, squorum)
}

func StartMajQuorum(num int) {

	n = num

	f = (n - 1) / 2
	quorum = n - f
	squorum = f + 1
	half = n/2 + 1

	bufferd.Init()
	bufferc.Init()
	buffer.Init()
	cer.Init()
}

func AddToIntBuffer(view int, source int64, content message.MessageWithSignature, step Step) {
	switch step {
	case VC:
		intbuffer.InsertValue(view, source, content)
	}

}

/*
Check whether a quorum of messages have been received. Used for view changes and garbage collection.
*/
func CheckIntQuorum(input int, step Step) bool {
	switch step {
	case VC:
		return intbuffer.GetLen(input) >= quorum
	}
	return false
}

func GetVCMsgs(input int, step Step) []message.MessageWithSignature {
	switch step {
	case VC:
		//return intbuffer.V[input]
		return intbuffer.GetV(input)
	}
	var emptyqueue []message.MessageWithSignature
	return emptyqueue
}
