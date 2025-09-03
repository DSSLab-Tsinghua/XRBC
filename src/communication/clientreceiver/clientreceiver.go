/*
Receiver functions.
It implements all the gRPC services defined in communication.proto file.
*/

package clientreceiver

import (
	"google.golang.org/grpc"

	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"CrossRBC/src/broadcast/ra"
	"CrossRBC/src/config"
	"CrossRBC/src/consensus"
	"CrossRBC/src/cryptolib"
	"CrossRBC/src/db"
	logging "CrossRBC/src/logging"
	pb "CrossRBC/src/proto/proto/communication"
	"CrossRBC/src/utils"
)

var id string
var wg sync.WaitGroup
var sleepTimerValue int
var con int
var verbose bool

type server struct {
	pb.UnimplementedSendServer
}

type reserver struct {
	pb.UnimplementedSendServer
}

/*
Handle replica messages (consensus normal operations)
*/
func (s *server) SendMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	//go handler.HandleByteMsg(in.GetMsg())
	return &pb.Empty{}, nil
}

func (s *server) SendRequest(ctx context.Context, in *pb.Request) (*pb.RawMessage, error) {
	return HandleRequest(in)
}

func (s *reserver) SendRequest(ctx context.Context, in *pb.Request) (*pb.RawMessage, error) {
	return HandleRequest(in)
}

func HandleRequest(in *pb.Request) (*pb.RawMessage, error) {
	wtype := in.GetType()
	switch wtype {
	case pb.MessageType_WRITE_BATCH:
		consensus.HandleBatchRequest(in.GetRequest())
		reply := []byte("batch rep")

		return &pb.RawMessage{Msg: reply}, nil
	default:
		h := cryptolib.GenHash(in.GetRequest()) //h is different because of different client id
		go consensus.HandleRequest(in.GetRequest(), utils.BytesToString(h))

		reply := []byte("rep")

		return &pb.RawMessage{Msg: reply}, nil
	}

}

func (s *server) RBCSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.CrossRA:
		go ra.HandleRBCMsg(in.GetMsg())

		//to-do: add other consensus types

	}
	return &pb.Empty{}, nil
}

// // CrossConsensus-mode 3 Sig Cross
// func (s *server) SendSigRequest(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
// 	return HandleSigRequest(in)
// }

// CrossConsensus-mode 3 Sig Reply
// func (s *server) SendReplyRequest(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
// 	return HandleSigRequest(in)
// }

// func HandleSigRequest(in *pb.Request) (*pb.Empty, error) {
// 	wtype := in.GetType()

// 	p := fmt.Sprintf("[Sig(clientreceiver)]Server %v received %v", id, wtype)
// 	logging.PrintLog(true, logging.NormalLog, p)

// 	switch wtype {
// 	// case pb.MessageType_CROSS:
// 	// 	h := cryptolib.GenHash(in.GetRequest())
// 	// 	go consensus.HandleSigRequest(in.GetRequest(), utils.BytesToString(h))
// 	// 	return nil, nil

// 	case pb.MessageType_CROSSREPLY:
// 		request_reply := in.GetRequest()
// 		reply_op := message.DeserializeClientSigReplyRequest(request_reply).HASHOP

// 		p := fmt.Sprintf("[Sig=%v(clientreceiver)]Server %v start handling request hash:%v", wtype, id, reply_op)
// 		logging.PrintLog(true, logging.NormalLog, p)

// 		h := cryptolib.GenHash(request_reply)
// 		h_string := utils.BytesToString(h)

// 		go func() {
// 			// log.Println("Starting goroutine for HandleSigRequest")
// 			consensus.HandleSigRequest(request_reply, h_string)
// 			// log.Println("Goroutine for HandleSigRequest finished")
// 		}()

// 		return &pb.Empty{}, nil

// 	case pb.MessageType_CROSSFETCH:
// 		request_fetch := in.GetRequest()
// 		fetch_lr := message.DeserializeCrossFetchMessage(request_fetch).LR
// 		fetch_hr := message.DeserializeCrossFetchMessage(request_fetch).LastR

// 		p := fmt.Sprintf("[Sig=%v(clientreceiver)]Server %v start handling fetch request, lr-%v, hr-%v", wtype, id, fetch_lr, fetch_hr)
// 		logging.PrintLog(true, logging.NormalLog, p)

// 		h := cryptolib.GenHash(request_fetch)
// 		h_string := utils.BytesToString(h)

// 		go func() {
// 			// log.Println("Starting goroutine for HandleSigRequest")
// 			consensus.HandleSigRequest(request_fetch, h_string)
// 			// log.Println("Goroutine for HandleSigRequest finished")
// 		}()

// 		return &pb.Empty{}, nil

// 	default:
// 		// 	h := cryptolib.GenHash(in.GetRequest())
// 		// 	go consensus.HandleSigRequest(in.GetRequest(), utils.BytesToString(h))
// 	}

// 	return nil, nil

// }

/*
Handle join requests for both static membership (initialization) and dynamic membership.
Each replica gets a conformation for a membership request.
*/
func (s *server) Join(ctx context.Context, in *pb.RawMessage) (*pb.RawMessage, error) {

	reply := []byte("hi") //handler.HandleJoinRequest(in.GetMsg())
	result := true

	return &pb.RawMessage{Msg: reply, Result: result}, nil
}

func (s *server) HotStuffSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	if err := ctx.Err(); err != nil {
		// if the ctx is cancelled or timeout, this message has no need to process.
		return nil, err
	}

	//if consensus.SleepFlag.Get() == 0 {
	//	// In the test, a sleeping replica does nothing when it receives a message.
	//	return &pb.Empty{}, nil
	//}

	go consensus.HandleQCByteMsg(in.GetMsg())
	consensus.MsgQueue.AppendAndTrimToMaxSize(in.GetMsg())
	db.PersistValue("MsgQueue", &consensus.MsgQueue, db.PersistAll)
	return &pb.Empty{}, nil
}

/*
Register rpc socket via port number and ip address
*/
func register(port string, splitPort bool) {
	log.Printf("[clientreceiver] id: %s, start register\n", id)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		p := fmt.Sprintf("[Communication Receiver Error(clientreceiver)] failed to listen %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}

	// if config.FetchVerbose() {
	// 	p := fmt.Sprintf("[Communication Receiver (clientreceiver)] listening to port %v", port)
	// 	logging.PrintLog(config.FetchVerbose(), logging.NormalLog, p)
	// }

	go serveGRPC(lis, splitPort)
	log.Printf("[clientreceiver] id#%s-port#%v, register is finished\n", id, port)

	wg.Wait()

}

/*
Have serve grpc as a function (could be used together with goroutine)
*/
func serveGRPC(lis net.Listener, splitPort bool) {
	defer wg.Done()

	// log.Printf("[clientreceiver]lis:%v, splitPort%v\n", lis, splitPort)

	if splitPort {

		s1 := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))

		pb.RegisterSendServer(s1, &reserver{})
		log.Printf("listening to split port")
		if err := s1.Serve(lis); err != nil {
			p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
			logging.PrintLog(true, logging.ErrorLog, p)
			os.Exit(1)
		}

		return
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))

	pb.RegisterSendServer(s, &server{})

	if err := s.Serve(lis); err != nil {
		p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
		logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
	}

	log.Printf("[clientreceiver] grpc serve is finished\n")

}

/*
Start receiver parameters initialization of CrossConsensus-mode 3 Sig
*/
func StartSigClientReceiver(rid string) {
	id = rid

	// config.LoadConfig()
	logging.SetLogOpt(config.FetchLogOpt())
	// verbose = config.FetchVerbose()
	// con = config.Consensus()
	consensus.StartSigClientHandler(rid)

	sleepTimerValue = config.FetchSleepTimer()

	port := config.FetchClientPort(rid)

	consensus.StartHandler(rid)

	log.Printf("[Sig(clientreceiver)]Client %v started HANDLER\n", rid)

	log.Printf("[clientreceiver]client#%v ready to listen to port %v", rid, port)

	wg.Add(1)
	register(port, false)
	wg.Wait()

	log.Printf("[clientreceiver]client#%v StartClientReceiver is finished", rid)

}
