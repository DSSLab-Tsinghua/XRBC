package message

import (
	//"encoding/json"
	"CrossRBC/src/logging"
	pb "CrossRBC/src/proto/proto/communication"
	"fmt"

	"github.com/vmihailenco/msgpack"
)

type ClientRequest struct {
	Type pb.MessageType
	ID   int64
	OP   []byte // Message payload. Opt for contract.
	TS   int64  // Timestamp
	R    int64  // Epoch number
}

func (r *ClientRequest) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeClientRequest(input []byte) ClientRequest {
	var clientRequest = new(ClientRequest)
	msgpack.Unmarshal(input, &clientRequest)
	return *clientRequest
}

// CrossConsensus mode 3: Sig
type ClientSigRequest struct {
	Type pb.MessageType
	R    int64 //epoch number
	OP   []byte
	ID   int64
	PI   []byte //signature of the PP(r')
	PP   []byte //storing the (r', vr', pai'）
}

func (r *ClientSigRequest) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeClientSigRequest(input []byte) ClientSigRequest {
	// p := fmt.Sprintf("[Sig(message)] Deserialze request msg %v", input)
	// logging.PrintLog(true, logging.NormalLog, p)

	var clientSigRequest = new(ClientSigRequest)
	err := msgpack.Unmarshal(input, &clientSigRequest)
	if err != nil {
		p := fmt.Sprintf("[Sig(message)] Fail to deserialize request msg %v", input)
		logging.PrintLog(true, logging.ErrorLog, p)
	}
	return *clientSigRequest
}

type CrossRepMessage struct {
	Type   pb.MessageType
	R      int64  //epoch number
	HASHOP []byte //hash of the OP
	ID     int64
	Sig    []byte
}

func (r *CrossRepMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeClientSigReplyRequest(input []byte) CrossRepMessage {
	var crossRepMessage = new(CrossRepMessage)
	msgpack.Unmarshal(input, &crossRepMessage)
	return *crossRepMessage
}

type CrossConfirmMessage struct {
	Type   pb.MessageType
	R      int64  //epoch number
	HASHOP []byte //hash of the OP
	ID     int64
	Sig    []byte //signature of the n-f signatures in REP message

}

func (r *CrossConfirmMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCrossConfirmMessage(input []byte) CrossConfirmMessage {
	var crossConfirmMessage = new(CrossConfirmMessage)
	msgpack.Unmarshal(input, &crossConfirmMessage)
	return *crossConfirmMessage
}

type CrossFetchMessage struct {
	Type  pb.MessageType
	ID    int64
	LR    int64 //  epoch bar in the server
	LastR int64 // last epoch number of PP
}

func (r *CrossFetchMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCrossFetchMessage(input []byte) CrossFetchMessage {
	var crossFetchMessage = new(CrossFetchMessage)
	msgpack.Unmarshal(input, &crossFetchMessage)
	return *crossFetchMessage
}

type CrossCatchUpMessage struct {
	Type pb.MessageType
	ID   int64
	R    int64
	PP   []byte
}

func (r *CrossCatchUpMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCrossCatchUpMessage(input []byte) CrossCatchUpMessage {
	var crossCatchUpMessage = new(CrossCatchUpMessage)
	msgpack.Unmarshal(input, &crossCatchUpMessage)
	return *crossCatchUpMessage
}
