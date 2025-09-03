package consensus

import (
	"log"
	"sync"

	"CrossRBC/src/add"
	"CrossRBC/src/broadcast/ra"
	"CrossRBC/src/broadcast/sig"
	"CrossRBC/src/communication/sender"
	"CrossRBC/src/config"
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/db"
	"CrossRBC/src/utils"
)

var msg []int

type ConsensusType int

const (
	CrossRA       ConsensusType = 100
	CrossABC      ConsensusType = 101
	CrossSig      ConsensusType = 102
	CrossSigGroup ConsensusType = 103
	DeltaBA       ConsensusType = 104
)

type RbcType int

const (
	RBC   RbcType = 0
	ECRBC RbcType = 1
)

type TypeOfBuffer int32

const (
	BUFFER TypeOfBuffer = 0
	CACHE  TypeOfBuffer = 1
)

type ConsensusStatus int

const (
	PREPREPARED ConsensusStatus = 0
	PREPARED    ConsensusStatus = 1
	COMMITTED   ConsensusStatus = 2
	BOTHPANDC   ConsensusStatus = 3
)

func StartProcessing(data []byte, txNum int) {
	switch consensus {
	case CrossRA:
		log.Printf("[RA] Start processing, txNum %v", txNum)
		StartCrossBA(data, txNum)
	case CrossABC:
		log.Printf("[ABC] Start processing, txNum %v", txNum)
		StartCrossABC(data, txNum)
	case CrossSig:
		log.Printf("[SIG] Start processing, txNum %v", txNum)
		StartCrossSig(data, txNum)
	case CrossSigGroup:
		log.Printf("[SIGGroup] Start processing, txNum %v, id %v", txNum, id)
		StartSigCross(data, txNum)
	case DeltaBA:
		log.Printf("[DeltaBA] Start processing, txNum %v", txNum)
		SendDeltaPropose(data, txNum)
	}

}

func GetInstanceID(input int) int {
	return input + n*epoch.Get() //baseinstance*epoch.Get()
}

func GetIndexFromInstanceID(input int, e int) int {
	return input - n*e
}

func GetInstanceIDsOfEpoch() []int {
	var output []int
	for i := 0; i < len(members); i++ {
		output = append(output, GetInstanceID(members[i]))
	}
	return output
}

func StartHandler(rid string) {
	id, errs = utils.StringToInt64(rid)

	if errs != nil {
		log.Printf("[Error] Replica id %v is not valid. Double check the configuration file", id)
		return
	}
	iid, _ = utils.StringToInt(rid)

	// config.LoadConfig()
	cryptolib.StartCrypto(id, config.CryptoOption())
	consensus = ConsensusType(config.Consensus())
	// rbcType = RbcType(config.RBCType())

	n = config.FetchNumReplicas()
	n_groupB = config.FetchNumReplicasGroupB()
	curStatus.Init()
	confirmStatus.Init()

	epoch.Init()

	queue.Init()
	MsgQueue.Init()
	db.PersistValue("queue", &MsgQueue, db.PersistAll)
	verbose = config.FetchVerbose()
	sleepTimerValue = config.FetchSleepTimer()
	addVersion := config.FetchAdd()

	nodes := config.FetchNodes()
	for i := 0; i < len(nodes); i++ {
		nid, _ := utils.StringToInt(nodes[i])
		members = append(members, nid)
	}

	// log.Printf("sleeptimer value %v", sleepTimerValue)

	switch consensus {
	case CrossRA:
		// log.Printf("[RA(consensus)] running consensus#%v, server id#%v, epoch number#%v", consensus, id, epoch.Get())
		InitCrossRA()
		ra.InitRBC(id, n, verbose)
		addVersion := config.FetchAdd()
		if addVersion == 1 {
			add.InitADD(id, n_groupB, verbose)
		}

		go RequestMonitor(epoch.Get())
		sender.StartSender(rid)

	case CrossABC:
		log.Printf("[ABC(consensus)] running consensus#%v, server id#%v, epoch number#%v", consensus, id, epoch.Get())
		add.InitADD(id, n_groupB, verbose)
		InitHotStuff(id)
		switch addVersion {
		case 0:
			InitCROSS(id, n_groupB, verbose)
		case 1:
			InitADD(id, n_groupB, verbose)
		}

		sender.StartSender(rid)
		go RequestMonitor(epoch.Get())

	case CrossSig:
		log.Printf("[Sig(consensus)] running consensus%v, id %v", consensus, id)
		InitSig()
		InitHotStuff(id)
		sig.InitCrossSig(id, n, verbose)
		sender.StartSender(rid)
		go RequestMonitor(epoch.Get())

	case CrossSigGroup:
		log.Printf("[SigGroup(consensus)] running consensus%v, id%v", consensus, id)
		InitGroupSig(id, n, verbose)
		InitHotStuff(id)
		sender.StartSender(rid)
		go RequestMonitor(epoch.Get())

	case DeltaBA:
		log.Printf("[DeltaBA(consensus)] running DELTA")
		InitDelta(id, n, verbose)
		sender.StartSender(rid)
		go RequestMonitor(epoch.Get())
	default:
		log.Fatalf("Consensus type %v not supported", consensus)
	}

	// sender.StartSender(rid)
	// go RequestMonitor(epoch.Get())
}

// func StartGHandler(rid string) {
// 	id, errs = utils.StringToInt64(rid)

// 	if errs != nil {
// 		log.Printf("[Error] Replica id %v is not valid. Double check the configuration file", id)
// 		return
// 	}
// 	iid, _ = utils.StringToInt(rid)

// 	// configGroup.LoadConfig()
// 	cryptolib.StartCrypto(id, configGroup.CryptoOption())
// 	consensus = ConsensusType(configGroup.Consensus())
// 	// rbcType = RbcType(config.RBCType())

// 	//n 目前是所有replicas的个数
// 	n = configGroup.FetchNumReplicas()
// 	curStatus.Init()
// 	confirmStatus.Init()

// 	epoch.Init()

// 	queue.Init()
// 	MsgQueue.Init()

// 	db.PersistGroupSigValue("queue", &MsgQueue, db.PersistAll)

// 	verbose = configGroup.FetchVerbose()
// 	sleepTimerValue = configGroup.FetchSleepTimer()

// 	nodes := configGroup.FetchNodes()
// 	for i := 0; i < len(nodes); i++ {
// 		nid, _ := utils.StringToInt(nodes[i])
// 		members = append(members, nid)
// 	}

// 	switch consensus {
// 	case CrossSigGroup:
// 		log.Printf("***[SigGroup(consensus)] StartHandler, id***")
// 		InitGroupSig(id, n, verbose)
// 		InitGroupSigHotStuff(id)
// 		sender.StartSender(rid)
// 		go RequestMonitor(epoch.Get())
// 	default:
// 		log.Fatalf("Consensus type %v not supported", consensus)
// 	}
// }

// func StartSigClientHandler(rid string) {
// 	id, errs = utils.StringToInt64(rid)

// 	if errs != nil {
// 		log.Printf("[Error] Replica id %v is not valid. Double check the configuration file", id)
// 		return
// 	}
// 	iid, _ = utils.StringToInt(rid)

// 	// config.LoadConfig()
// 	cryptolib.StartCrypto(id, config.CryptoOption())
// 	consensus = ConsensusType(config.Consensus())

// 	n = config.FetchNumReplicas()
// 	repStatus.Init()
// 	confirmStatus.Init()
// 	fetchStatus.Init()
// 	catchUpStatus.Init()

// 	epoch.Init()
// 	queue.Init()
// 	MsgQueue.Init()
// 	db.PersistValue("queue", &MsgQueue, db.PersistAll)
// 	verbose = config.FetchVerbose()
// 	sleepTimerValue = config.FetchSleepTimer()

// 	// nodes := config.FetchNodes()
// 	// for i := 0; i < len(nodes); i++ {
// 	// 	nid, _ := utils.StringToInt(nodes[i])
// 	// 	members = append(members, nid)
// 	// }

// 	// log.Printf("sleeptimer value %v", sleepTimerValue)

// 	switch consensus {
// 	case CrossSig:
// 		log.Printf("[Sig(consensus)] client handler running consensus%v, id %v", consensus, id)
// 		InitSig()
// 		sig.InitCrossSig(id, n, verbose)
// 	default:
// 		log.Fatalf("Consensus type %v not supported", consensus)
// 	}

// 	// sender.StartSender(rid)
// 	go SigClientRequestMonitor(epoch.Get())
// }

type QueueHead struct {
	Head string
	sync.RWMutex
}

func (c *QueueHead) Set(head string) {
	c.Lock()
	defer c.Unlock()
	c.Head = head
}

func (c *QueueHead) Get() string {
	c.RLock()
	defer c.RUnlock()
	return c.Head
}

type CurStatus struct {
	enum Status
	sync.RWMutex
}

type Status int

const (
	READY      Status = 0
	PROCESSING Status = 1
	VIEWCHANGE Status = 2
	SLEEPING   Status = 3
	RECOVERING Status = 4
)

func (c *CurStatus) Set(status Status) {
	c.Lock()
	defer c.Unlock()
	c.enum = status
}

func (c *CurStatus) Init() {
	c.Lock()
	defer c.Unlock()
	c.enum = READY
}

func (c *CurStatus) Get() Status {
	c.RLock()
	defer c.RUnlock()
	return c.enum
}
