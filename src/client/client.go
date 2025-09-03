package client

import (
	sender "CrossRBC/src/communication/clientsender"
	"CrossRBC/src/config"
	"CrossRBC/src/cryptolib"

	// "encoding/binary"
	// "crypto"
	//"encoding/json"
	"github.com/vmihailenco/msgpack"

	logging "CrossRBC/src/logging"
	"CrossRBC/src/message"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/utils"
	"fmt"
	"log"
)

var cid int64
var err error
var clientTimer int
var pp utils.IntsBytesMap  // epoch是key，msg和pi是value
var Cer utils.IntsBytesMap //epoch是key，hashmsg和sigma是value
var epoch utils.IntValue
var piList utils.IntByteMap //epoch是key，pi是value
var opList utils.IntByteMap //epoch是key，op是value
var verbose bool

func GetCID() int64 {
	return cid
}

func SignedRequest(cid int64, dataSer []byte) ([]byte, bool) {
	request := message.MessageWithSignature{
		Msg: dataSer,
		Sig: []byte(""), //cryptolib.GenSig(cid, dataSer),
	}

	requestSer, err := request.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the request with signiture: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return requestSer, true
}

func SendWriteRequest(op []byte, epoch int) {
	dataSer, result1 := CreateWORequest(cid, op, epoch)
	if !result1 {
		log.Fatal("[Client error] fail to create the message.")
		return
	}

	// requestSer, result2 := SignedRequest(cid, dataSer)
	// if !result2 {
	// 	return
	// }

	// var requestArr [][]byte
	// requestArr = append(requestArr, dataSer)
	// byteRequsets := SerializeRequests(requestArr)
	// if err != nil {
	// 	log.Fatal("[Client error] fail to serialize the message.")
	// 	return
	// }

	sender.BroadcastRequest(pb.MessageType_WRITE, dataSer)
}

// func SendWriteGRequest(op []byte, numReq int) {
// 	dataSer, result1 := CreateGroupSigRequest(cid, op, numReq)
// 	if !result1 {
// 		log.Fatal("[Client error] fail to create the message.")
// 		return
// 	}

// 	// requestSer, result2 := SignedRequest(cid, dataSer)
// 	// if !result2 {
// 	// 	return
// 	// }

// 	var requestArr [][]byte
// 	requestArr = append(requestArr, dataSer)
// 	byteRequsets := SerializeRequests(requestArr)
// 	if err != nil {
// 		log.Fatal("[Client error] fail to serialize the message.")
// 		return
// 	}

// 	sender.BroadcastGRequest(pb.MessageType_WRITE, byteRequsets)
// }

func SendBatchRequest(op []byte, batchSize int, epoch int) {
	var requestArr [][]byte

	//Based on batch size, create a batch of requests
	for i := 0; i < batchSize; i++ {

		dataSer, result1 := CreateWORequest(cid, op, epoch)
		if !result1 {
			log.Fatal("[Client error] fail to create the message.")
			return
		}

		// requestSer, result2 := SignedRequest(cid, dataSer)
		// if !result2 {
		// 	log.Fatal("[Client error] fail to sign the message.")
		// 	return
		// }
		requestArr = append(requestArr, dataSer)
	}

	p := fmt.Sprintf("[Client-epoch#%v] Add msg to array(length: %v)", epoch, len(requestArr))
	logging.PrintLog(true, logging.NormalLog, p)

	byteRequsets, err := SerializeBatchRequests(requestArr)
	if err != nil {
		log.Fatal("[Client error] fail to serialize the message.")
		return
	}

	sender.BroadcastRequest(pb.MessageType_WRITE_BATCH, byteRequsets)
}

func SerializeRequests(r [][]byte) []byte {
	crossmsg := message.CrossMessage{OP: r}
	serialized_requests, err := crossmsg.Serialize()
	if err != nil {
		log.Fatal("[Client error] fail to serialize the message.")
	}
	return serialized_requests
}

func SerializeBatchRequests(r [][]byte) ([]byte, error) {
	jsons, err := msgpack.Marshal(r)
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the message %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), err
	}
	return jsons, nil
}

// RA+ABC mode
func CreateWORequest(cid int64, op []byte, epoch int) ([]byte, bool) {
	data := message.ClientRequest{
		Type: pb.MessageType_WRITE,
		ID:   cid,
		OP:   op,
		R:    int64(epoch), //ra-epoch number; sigGroup-numOfRequests
	}

	p1 := fmt.Sprintf("[CROSS-Client%v-epoch#%v]Create Request", cid, epoch)
	logging.PrintLog(true, logging.NormalLog, p1)

	// epoch.Increment()

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

func CreateGroupSigRequest(cid int64, op []byte, epoch int) ([]byte, bool) {
	data := message.ClientSigRequest{
		Type: pb.MessageType_WRITE,
		ID:   cid,
		OP:   op,
		R:    int64(epoch), //ra-epoch number; sigGroup-numOfRequests
	}

	p1 := fmt.Sprintf("[CROSS-Client%v-epoch#%v]Create Request", cid, epoch)
	logging.PrintLog(true, logging.NormalLog, p1)

	// epoch.Increment()

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

// Sig mode
func CreateRequest(cid int64, op []byte) ([]byte, bool) {

	data := message.ClientRequest{
		Type: pb.MessageType_WRITE,
		ID:   cid,
		OP:   op,
		TS:   utils.MakeTimestamp(),
	}

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

// Sig-CROSS: send CROSS messages(batch) to all servers
func SendSigWriteRequest(requestListSer []byte, epochNum int) {
	var requestList [][]byte

	err := utils.DeserializeBytesByBuffer(requestListSer, &requestList)

	if err != nil {
		p := fmt.Sprintf("[Sig-Cross(client)] Error: fail to deserialize requestList: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
	}

	if len(requestList) == 1 {
		dataSerialized, result1 := CreateSigRequest(cid, requestList[0], epochNum)
		if !result1 {
			return
		}
		p := fmt.Sprintf("[Sig-Cross(client)-epoch#%v]client#%v: Start sending CROSS msg (len: %v)", epochNum, cid, len(requestList[0]))
		logging.PrintLog(true, logging.NormalLog, p)

		sender.BroadcastSigRequest(pb.MessageType_CROSS, dataSerialized)

	} else {
		for i := 0; i < len(requestList); i++ {

			dataSer, result1 := CreateSigRequest(cid, requestList[i], i)
			if !result1 {
				return
			}

			p := fmt.Sprintf("[Sig-Cross(client)-epoch#%v]client#%v: Start sending CROSS msg (len: %v)", i, cid, len(requestList[i]))
			logging.PrintLog(true, logging.NormalLog, p)

			sender.BroadcastSigRequest(pb.MessageType_CROSS, dataSer)
		}
	}
}

func CreateSigRequest(cid int64, op []byte, r int) ([]byte, bool) {
	cryptolib.StartECDSA(cid)

	// pp_v1 := pp.GetAll()
	// p := fmt.Sprintf("[Sig-Cross(client)] epoch#%v check current pp before add:%v", r, pp_v1)
	// logging.PrintLog(true, logging.NormalLog, p)

	// check Cer中所有比r小的epoch是否都已经被deliver，如果没有就加到pp中去
	if r > 0 {
		lastDeliveredR, _ := Cer.GetLargestKey()
		lastCompareR := r - 1

		for int(lastCompareR) > lastDeliveredR {
			piLast, _ := piList.Get(int(lastCompareR))
			pp.Add(int(lastCompareR), op, piLast)
			lastCompareR--
		}
	}

	pp_v2 := pp.GetAll()
	p1 := fmt.Sprintf("[Sig-Cross(client)-epoch#%v] check pp after checking Cer:%v", r, pp_v2)
	logging.PrintLog(verbose, logging.NormalLog, p1)

	ppByte, err := pp.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the pp: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	// to-do: pi是什么
	pi := cryptolib.GenSig(cid, ppByte)
	piList.Insert(int(r), pi)
	opList.Insert(int(r), op)

	p2 := fmt.Sprintf("[Sig-Cross(client)] Set SIG message: epoch#%v, client#%v, PP:%v ", r, cid, pp.GetAll())
	logging.PrintLog(verbose, logging.NormalLog, p2)

	data := message.ClientSigRequest{
		Type: pb.MessageType_CROSS,
		R:    int64(r),
		OP:   op,
		ID:   cid,
		PI:   pi,
		PP:   ppByte,
	}

	dataSer, err := data.Serialize()
	if err != nil {
		p := fmt.Sprintf("[Client error] fail to serialize the write request: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		return []byte(""), false
	}

	return dataSer, true
}

func AddCert(e int) {
	op, opExists := opList.Get(e)
	pi, piExists := piList.Get(e)
	if !opExists || !piExists {
		log.Printf("Error: op or pi not found for epoch %d", e)
		return
	}
	Cer.Add(e, op, pi)
}

// print cer所有内容
func PrintCer() {
	cerAll := Cer.GetAll()
	for epoch, data := range cerAll {
		p := fmt.Sprintf("[Client] Cer: Epoch: %d, Msg: %s, Sig: %s", epoch, data.Byte1, data.Byte2)
		logging.PrintLog(true, logging.NormalLog, p)
	}
}

// func SetEpoch(e int) {
// 	epoch.Set(e)
// }

// CrossConsensus: mod 1+2 RA，ABC
// func StartWOClient(rid string, loadkey bool) {
// 	logging.SetID(rid)
// 	config.LoadConfig()
// 	logging.SetLogOpt(config.FetchLogOpt())

// 	log.Printf("Client %s started.", rid)
// 	cid, err = utils.StringToInt64(rid)
// 	sender.StartClientSender(rid, loadkey)
// 	clientTimer = config.FetchBroadcastTimer()

// 	epoch.Init()
// }

// CrossConsensus: mod 3 Sig
func StartClient(rid string, loadkey bool) {
	logging.SetID(rid)
	// log.Printf("[Sig(client)client#%v set log id ok]", rid)

	configBool := config.LoadConfig()
	if !configBool {
		log.Printf("[Sig(client)]client#%v load config error]", rid)
	}
	// log.Printf("[Sig(client)]client#%v load config ok]", rid)

	logging.SetLogOpt(config.FetchLogOpt())
	// log.Printf("[Sig(client)]client#%v set log option ok]", rid)

	clientTimer = config.FetchBroadcastTimer()
	// log.Printf("[Sig(client)]client#%v fetch timer ok]", rid)
	verbose = config.FetchVerbose()
	epoch.Init() //start with 0
	Cer.Init()
	piList.Init()
	opList.Init()

	// Init pp
	pp.Init()

	// log.Printf("[Sig(client)]Client#%s configs ok", rid)
	cid, err = utils.StringToInt64(rid)
	if err != nil {
		p := fmt.Sprintf("[Sig(client)]Client#%s id is not valid. Double check the configuration file", rid)
		logging.PrintLog(true, logging.ErrorLog, p)
	}

	sender.StartClientSender(rid, loadkey)
	// go receiver.StartSigClientReceiver(rid)

	log.Printf(("Client %s init sender and receiver."), rid)
}
