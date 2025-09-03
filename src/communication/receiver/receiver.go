/*
Receiver functions.
It implements all the gRPC services defined in communication.proto file.
*/

package receiver

import (
	"google.golang.org/grpc"

	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"CrossRBC/src/add"
	"CrossRBC/src/broadcast/ra"
	"CrossRBC/src/communication"
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
		// batch mode
		p := fmt.Sprintf("[CROSS(batch mode)-Receiver]Server %v received %v msg", id, wtype)
		logging.PrintLog(verbose, logging.NormalLog, p)

		consensus.HandleBatchRequest(in.GetRequest())

		reply := []byte("batch rep")
		return &pb.RawMessage{Msg: reply}, nil

	default:
		h := cryptolib.GenHash(in.GetRequest())
		p := fmt.Sprintf("[CROSS(write mode)-Receiver]Server %v received %v msg, hash:%v", id, wtype, h)
		logging.PrintLog(verbose, logging.NormalLog, p)

		go consensus.HandleRequest(in.GetRequest(), utils.BytesToString(h))

		reply := []byte("rep")

		return &pb.RawMessage{Msg: reply}, nil
	}

}

func (s *server) RBCSendByteMsg(ctx context.Context, in *pb.RawMessage) (*pb.Empty, error) {
	switch consensus.ConsensusType(con) {
	case consensus.CrossRA:
		if config.FetchAdd() == 1 {
			go add.HandleADDMsg(in.GetMsg())
		} else if config.FetchAdd() == 0 {
			go ra.HandleRBCMsg(in.GetMsg())
		}
	case consensus.CrossABC:
		if config.FetchAdd() == 1 {
			go consensus.ADDHandleADDMsg(in.GetMsg())
		} else if config.FetchAdd() == 0 {
			go consensus.XHandleRBCMsg(in.GetMsg())
		}
	case consensus.CrossSigGroup:
		go consensus.HandleGroupSigMessage(in.GetMsg())
	case consensus.DeltaBA:
		go consensus.HandleDeltaMsg(in.GetMsg())

	}
	return &pb.Empty{}, nil
}

// CrossConsensus-mode 3 Sig Cross
// func (s *server) SendSigRequest(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
// 	return HandleSigRequest(in)
// }

// // // CrossConsensus-mode 3 Sig Confirm
// func (s *server) SendConfirmRequest(ctx context.Context, in *pb.Request) (*pb.Empty, error) {
// 	return HandleSigRequest(in)
// }

// func HandleSigRequest(in *pb.Request) (*pb.Empty, error) {
// 	wtype := in.GetType()

// 	p := fmt.Sprintf("[Sig(receiver)]Server %v received %v", id, wtype)
// 	logging.PrintLog(verbose, logging.NormalLog, p)

// 	switch wtype {
// 	case pb.MessageType_CROSS:
// 		h := cryptolib.GenHash(in.GetRequest())
// 		go consensus.HandleSigRequest(in.GetRequest(), utils.BytesToString(h))
// 		return nil, nil

// 	// case pb.MessageType_CROSSREPLY:
// 	// 	h := cryptolib.GenHash(in.GetRequest())
// 	// 	go consensus.HandleSigRequest(in.GetRequest(), utils.BytesToString(h))
// 	// 	return nil, nil

// 	case pb.MessageType_CROSSCONFIRM:
// 		h := cryptolib.GenHash(in.GetRequest())
// 		id_int, err := utils.StringToInt64(id)
// 		if err != nil {
// 			p := fmt.Sprintf("[Sig-Confirm(receiver)]Server %v failed to convert id to int", id)
// 			logging.PrintLog(verbose, logging.ErrorLog, p)
// 		}

// 		sig.IDReset(id_int)
// 		go consensus.HandleSigRequest(in.GetRequest(), utils.BytesToString(h))
// 		return nil, nil

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
	lis, err := net.Listen("tcp", port)

	if err != nil {
		p := fmt.Sprintf("[Communication Receiver Error] failed to listen %v", err)
		logging.PrintLog(verbose, logging.ErrorLog, p)
		os.Exit(1)
	}
	if config.FetchVerbose() {
		p := fmt.Sprintf("[Communication Receiver] listening to port %v", port)
		logging.PrintLog(config.FetchVerbose(), logging.NormalLog, p)
	}

	log.Printf("ready to listen to port %v", port)
	go serveGRPC(lis, splitPort)

}

/*
Have serve grpc as a function (could be used together with goroutine)
*/
func serveGRPC(lis net.Listener, splitPort bool) {
	defer wg.Done()

	if splitPort {

		s1 := grpc.NewServer(grpc.MaxRecvMsgSize(52428800), grpc.MaxSendMsgSize(52428800))

		pb.RegisterSendServer(s1, &reserver{})
		log.Printf("listening to split port")
		if err := s1.Serve(lis); err != nil {
			p := fmt.Sprintf("[Communication Receiver Error] failed to serve: %v", err)
			logging.PrintLog(verbose, logging.ErrorLog, p)
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

}

/*
Start receiver parameters initialization
*/
func StartReceiver(rid string, cons bool) {
	id = rid
	logging.SetID(rid)

	config.LoadConfig()
	logging.SetLogOpt(config.FetchLogOpt())

	con = config.Consensus()
	verbose = config.FetchVerbose()

	sleepTimerValue = config.FetchSleepTimer()

	if cons {
		consensus.StartHandler(rid)
	}

	if config.SplitPorts() {
		//wg.Add(1)
		go register(communication.GetPortNumber(config.FetchPort(rid)), true)
	}

	wg.Add(1)
	register(config.FetchPort(rid), false)
	wg.Wait()

}

// func StartGReceiver(rid string, cons bool) {
// 	id = rid
// 	logging.SetID(rid)

// 	configGroup.LoadConfig()
// 	// config.LoadConfig()
// 	logging.SetLogOpt(configGroup.FetchLogOpt())

// 	con = configGroup.Consensus()

// 	sleepTimerValue = configGroup.FetchSleepTimer()

// 	if cons {
// 		consensus.StartGHandler(rid)
// 	}

// 	if configGroup.SplitPorts() {
// 		//wg.Add(1)
// 		go register(communication.GetPortNumber(configGroup.FetchPort(rid)), true)
// 	}

// 	wg.Add(1)
// 	register(configGroup.FetchPort(rid), false)
// 	wg.Wait()

// }

// /*
// Start receiver parameters initialization of CrossConsensus-mode 3 Sig
// */
// func StartSigReceiver(rid string, cons bool) {
// 	id = rid
// 	logging.SetID(rid)

// 	config.LoadConfig()
// 	logging.SetLogOpt(config.FetchLogOpt())

// 	con = config.Consensus()

// 	sleepTimerValue = config.FetchSleepTimer()

// 	if cons {
// 		consensus.StartHandler(rid)
// 		p := fmt.Sprintf("[Sig(receiver)]Server %v started HANDLER", id)
// 		logging.PrintLog(verbose, logging.NormalLog, p)
// 	}

// 	if config.SplitPorts() {
// 		//wg.Add(1)
// 		go register(communication.GetPortNumber(config.FetchPort(rid)), true)
// 	}

// 	wg.Add(1)
// 	register(config.FetchPort(rid), false)
// 	wg.Wait()

// }
