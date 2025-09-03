package message

import (
	"CrossRBC/src/cryptolib"
	//"encoding/json"
	pb "CrossRBC/src/proto/proto/communication"

	"github.com/vmihailenco/msgpack"
)

type ReplicaMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	TS       int64
	Payload  []byte //message payload
	Sig      []byte
	Value    int
	Maj      int
	Round    int
	Epoch    int
}

type SigGroupMessage struct {
	Type TypeOfMessage
	ID   int    //sender id
	R    int    //epoch number
	E    int    //total request number
	OP   []byte //original message
	PP   []byte //history messages
	Hash []byte //hash of the message
	Sig  []byte //signature of the message
}

type MessageWithSignature struct {
	Msg []byte
	Sig []byte
}

type RawOPS struct {
	OPS []pb.RawMessage
}

type SigRequests struct {
	OP []byte
}

type FPCCMessage struct {
	FPCC [][]byte
}

type CrossMessage struct {
	OP [][]byte
}

type CBCMessage struct {
	Value         map[int][]byte
	RawData       [][]byte
	MerkleBranch  [][][]byte
	MerkleIndexes [][]int64
}

type WRBCMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	Value    []int
	Epoch    int
}

type PBFTMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	Value    []byte
	Epoch    int
}

type PBFTECMessage struct {
	Mtype    TypeOfMessage
	Instance int
	Source   int64
	Hash     []byte
	FD       []byte //Hash value of hash(hash(fi)...)
	Fi       []byte //frag
	Epoch    int
}

type ViewChangeMessage struct {
	Mtype    pb.MessageType
	View     int
	Conf     int            //Dynamic membership only. Configuration number.
	ConfAddr map[int]string //Dynamic membership only. Address of the nodes.
	Seq      int
	Source   int64
	P        map[int]Cer                  //Prepare certificate
	PP       map[int]HotStuffMessage      //Pre-prepare message. Used for dynamic membership only.
	E        []int                        //Dynamic membership only. Current membership group
	O        map[int]MessageWithSignature // Operations. Used in new-view message only
	V        []MessageWithSignature       // A quorum of view-change messages. Used in new-view message only
}

type FragMessage struct {
	RawData       []byte
	MerkleBranch  [][]byte
	MerkleIndexes []int64
}

type Signatures struct {
	Hash []byte
	Sigs [][]byte
	IDs  []int64
}

type QCBlock struct {
	View       int
	Height     int //height
	Hash       []byte
	PreHash    []byte
	PrePreHash []byte
	QC         [][]byte
	Aux        []byte
	AuxQC      []byte
	IDs        []int64
}

type HotStuffMessage struct {
	Mtype     pb.MessageType
	Seq       int
	Source    int64
	View      int
	Conf      int
	OPS       []pb.RawMessage
	MemOPS    []pb.RawMessage
	Hash      []byte
	PreHash   []byte
	Sig       []byte
	QC        []byte
	LQC       []byte
	ComBlocks []byte
	Hashes    map[string]int
	SidInfo   []int64
	TS        int64
	Num       int
	Epoch     int
	Count     int
	V         []MessageWithSignature
}

func (r *QCBlock) Serialize() ([]byte, error) {
	msgser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return msgser, nil
}

func (r *QCBlock) Deserialize(input []byte) error {
	return msgpack.Unmarshal(input, r)
}

func DeserializeQCBlock(input []byte) QCBlock {
	var qcblock = new(QCBlock)
	msgpack.Unmarshal(input, qcblock)
	return *qcblock
}

func (r *FPCCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeFPCCMessage(input []byte) FPCCMessage {
	var fpcc = new(FPCCMessage)
	msgpack.Unmarshal(input, &fpcc)
	return *fpcc
}

func (r *CrossMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCrossMessage(input []byte) CrossMessage {
	var cross = new(CrossMessage)
	msgpack.Unmarshal(input, &cross)
	return *cross
}

func (r *FragMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeFragMessage(input []byte) FragMessage {
	var fragMessage = new(FragMessage)
	msgpack.Unmarshal(input, &fragMessage)
	return *fragMessage
}

func (r *Signatures) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeSignatures(input []byte) ([]byte, [][]byte, []int64) {
	var sigs = new(Signatures)
	msgpack.Unmarshal(input, &sigs)
	return sigs.Hash, sigs.Sigs, sigs.IDs
}

func (r *CBCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeCBCMessage(input []byte) CBCMessage {
	var cbcMessage = new(CBCMessage)
	msgpack.Unmarshal(input, &cbcMessage)
	return *cbcMessage
}

func (r *WRBCMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeWRBCMessage(input []byte) WRBCMessage {
	var wrbcmessage = new(WRBCMessage)
	msgpack.Unmarshal(input, &wrbcmessage)
	return *wrbcmessage
}

func (r *PBFTMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializePBFTMessage(input []byte) PBFTMessage {
	var pbftmessage = new(PBFTMessage)
	msgpack.Unmarshal(input, &pbftmessage)
	return *pbftmessage
}

func (r *PBFTECMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializePBFTECMessage(input []byte) PBFTECMessage {
	var pbftecmessage = new(PBFTECMessage)
	msgpack.Unmarshal(input, &pbftecmessage)
	return *pbftecmessage
}

func (r *RawOPS) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to MessageWithSignature
*/
func DeserializeRawOPS(input []byte) RawOPS {
	var rawOPS = new(RawOPS)
	msgpack.Unmarshal(input, &rawOPS)
	return *rawOPS
}

/*
Serialize MessageWithSignature
*/
func (r *MessageWithSignature) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

/*
Deserialize []byte to MessageWithSignature
*/
func DeserializeMessageWithSignature(input []byte) MessageWithSignature {
	var messageWithSignature = new(MessageWithSignature)
	msgpack.Unmarshal(input, &messageWithSignature)
	return *messageWithSignature
}

func (r *ReplicaMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeReplicaMessage(input []byte) ReplicaMessage {
	var replicaMessage = new(ReplicaMessage)
	msgpack.Unmarshal(input, &replicaMessage)
	return *replicaMessage
}

func (r *SigGroupMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

func DeserializeSigGroupMessage(input []byte) SigGroupMessage {
	var sigGroupMessage = new(SigGroupMessage)
	msgpack.Unmarshal(input, &sigGroupMessage)
	return *sigGroupMessage
}

func SerializeWithSignature(id int64, msg []byte) ([]byte, error) {
	request := MessageWithSignature{
		Msg: msg,
		Sig: cryptolib.GenSig(id, msg),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		return []byte(""), err
	}
	return requestSer, err
}

func SerializeWithMAC(id int64, dest int64, msg []byte) ([]byte, error) {
	request := MessageWithSignature{
		Msg: msg,
		Sig: cryptolib.GenMAC(id, msg),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		return []byte(""), err
	}
	return requestSer, err
}

// hotstuff message related functions

/*
Get hash of the entire batch
*/
func (r *HotStuffMessage) GetMsgHash() []byte {
	if len(r.OPS) == 0 {
		return []byte("")
	}
	return cryptolib.GenBatchHash(r.OPS)
}

/*
Serialize ReplicaMessage
*/
func (r *HotStuffMessage) Serialize() ([]byte, error) {
	ser, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return ser, nil
}

func DeserializeHotStuffMessage(input []byte) HotStuffMessage {
	var hotStuffMessage = new(HotStuffMessage)
	msgpack.Unmarshal(input, &hotStuffMessage)
	return *hotStuffMessage
}

/*
Create MessageWithSignature where the signature of each message (in ReplicaMessage) is attached.
The ReplicaMessage is first serialized into bytes and then signed.
Input

	tmpmsg: ReplicaMessage

Output

	MessageWithSignature: the struct
*/
func CreateMessageWithSig(tmpmsg HotStuffMessage) MessageWithSignature {
	tmpmsgSer, err := tmpmsg.Serialize()
	if err != nil {
		var emptymsg MessageWithSignature
		return emptymsg
	}

	op := MessageWithSignature{
		Msg: tmpmsgSer,
		Sig: cryptolib.GenSig(tmpmsg.Source, tmpmsgSer),
	}
	return op
}

/*
Deserialize []byte to ViewChangeMessage
*/
func DeserializeViewChangeMessage(input []byte) ViewChangeMessage {
	var viewChangeMessage = new(ViewChangeMessage)
	msgpack.Unmarshal(input, &viewChangeMessage)
	return *viewChangeMessage
}

/*
Serialize ViewChangeMessage
*/
func (r *ViewChangeMessage) Serialize() ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		return []byte(""), err
	}
	return jsons, nil
}

// Cer PREPARE certificates and COMMIT certificates
type Cer struct {
	Msgs [][]byte
}

/*
Add a message to the certificate
*/
func (r *Cer) Add(msg []byte) {
	r.Msgs = append(r.Msgs, msg)
}

/*
Get messages in a certificate
*/
func (r *Cer) GetMsgs() [][]byte {
	return r.Msgs
}

/*
Get the number of messages in the certiicate
*/
func (r *Cer) Len() int {
	return len(r.Msgs)
}
