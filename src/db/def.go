package db

import (
	"CrossRBC/src/config"
	"CrossRBC/src/logging"
	"fmt"
	"log"
)

type PersistLevelType int

const (
	PersistAll      PersistLevelType = 1
	PersistCritical PersistLevelType = 2
	NoPersist       PersistLevelType = 3
)

type DBValue interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

// hotstuff

// var Sequence utils.IntValue  //current Sequence number
// var curBlock message.QCBlock //recently voted block
// var votedBlocks utils.IntByteMap
// var lockedBlock message.QCBlock //locked block
// var curHash utils.ByteValue     //current hash
// var awaitingBlocks utils.IntByteMap
// var awaitingDecision utils.IntByteMap
// var awaitingDecisionCopy utils.IntByteMap
// var vcAwaitingVotes utils.IntIntMap

// viewchange
// var view int

// event
// var queue consensus.Queue

//messages
// var msgQueue consensus.Queue

// delivered blocks
// var committedBlocks utils.IntByteMap

func PersistValue(key string, value DBValue, level PersistLevelType) {
	if PersistLevelType(config.GPersistLevel()) <= level {
		msg := fmt.Sprintf("store value in database: %s, persist level: %d", key, level)
		logging.PrintLog(false, logging.NormalLog, msg)
		// time.Sleep(100 * time.Millisecond)
		err := WriteDB(key, value)
		if err != nil {
			log.Fatalf("error writing Sequence to database: %v", err)
		}
	}
}

func PersistGroupSigValue(key string, value DBValue, level PersistLevelType) {
	if PersistLevelType(config.PersistLevel()) <= level {
		msg := fmt.Sprintf("store value in database: %s, persist level: %d", key, level)
		logging.PrintLog(false, logging.NormalLog, msg)
		// time.Sleep(100 * time.Millisecond)
		err := WriteDB(key, value)
		if err != nil {
			log.Fatalf("error writing Sequence to database: %v", err)
		}
	}
}
