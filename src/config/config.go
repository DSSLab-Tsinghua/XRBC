/*
Loading configuration parameters from configuration file.
The file path is hard-coded to ../etc/conf.json.
TODO:
1) The file path can be tailored to different paths according to the deployment environments.
2) Need to validate the format of the configuration parameters
*/

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"CrossRBC/src/utils"
)

var nodeIDs []string
var nodeGroupAIDs []string
var nodeGroupBIDs []string
var nodes map[string]string

var clientnodes map[string]string
var clients map[string]string
var portMap map[string]string
var clientportMap map[string]string
var nodesReverse map[string]string
var clientnodesReverse map[string]string
var verbose bool
var maxBatchSize int
var sleepTimer int
var volume int
var add int
var clientTimer int
var broadcastTimer int
var evalMode int
var evalInterval int
var local bool
var maliciousNode bool
var maliciousMode int
var maliciousNID []int64

var shardIDs []string
var shardMap map[string]string
var shards map[string][]string

var partChurn bool
var viewChange bool
var rotatingTime int

var cryptoOpt int
var splitPorts bool
var logOpt int
var consensus int
var persistLevel int
var rbc int

type System struct {
	MaxBatchSize   int      `json:"maxBatchSize"`   // Max batch size for consensus
	SleepTimer     int      `json:"sleepTimer"`     // Timer for the while loops to monitor the status of requests. Should be a small value
	ClientTimer    int      `json:"clientTimer"`    // Timer for clients to monitor the responses and see whether the requests should be re-transmitted.
	BroadcastTimer int      `json:"broadcastTimer"` // Timer used for replicas to send gRPC messages to each other. Should be set to a value that is close to RTT
	Verbose        bool     `json:"verbose"`        // Whether log messages should be printed.
	EvalMode       int      `json:"evalMode"`       // Evaluation mode.
	EvalInterval   int      `json:"evalInterval"`   // Interval for assessing throughput
	CryptoOpt      int      `json:"cryptoOpt"`      // Crypto library option
	LogOpt         int      `json:"logOpt"`
	Local          bool     `json:"local"`         // Local or not
	MaliciousNode  bool     `json:"maliciousNode"` // Simulate a simple malicious node
	MaliciousMode  int      `json:"maliciousMode"` //
	MaliciousNID   string   `json:"maliciousNID"`  // Malicious node id
	SplitPorts     bool     `json:"splitPorts"`    // Split ports for request handler and server
	Consensus      int      `json:"consensus`      // Protocol
	PersistLevel   int      `json:"PersistLevel"`
	RBCType        int      `json:"RBCType"`  //RBC
	Replicas       []Shards `json:"replicas"` // Replica information
	Clients        []Client `json:"clients"`  // Client information
	PartChurn      bool     `json:"partChurn"`
	ViewChange     bool     `json:"viewChange"`
	RotatingTime   int      `json:"rotatingTime"`
	Volume         int      `json:"volume"`
	Add            int      `json:"add"`
}

type Shards struct {
	Shard  string    `json:"shard"`
	Config []Replica `json:"config"`
}

type Replica struct {
	ID   string `json:"id"`   // ID of the node
	Host string `json:"host"` // IP address
	Port string `json:"port"` // Port number
}

type Client struct {
	ID   string `json:"id"`   // ID of the node
	Host string `json:"host"` // IP address
	Port string `json:"port"` // Port number
}

func LoadConfig() bool {

	nodes = make(map[string]string)
	nodesReverse = make(map[string]string)
	clientnodes = make(map[string]string)
	clientnodesReverse = make(map[string]string)

	portMap = make(map[string]string)
	clientportMap = make(map[string]string)
	nodeIDs = make([]string, 0)

	shards = make(map[string][]string)
	shardIDs = make([]string, 0)
	shardMap = make(map[string]string)

	exepath, err := os.Executable()
	if err != nil {
		fmt.Println("[Configuration Error]  Failed to get path for the executable")
		// logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
		return false
	}

	binDir := filepath.Dir(exepath)
	// fmt.Printf("binDir: %s\n", binDir)

	// 获取项目根目录
	projectDir := filepath.Dir(binDir)
	// fmt.Printf("projectDir: %s\n", projectDir)

	// 构建指向etc目录的路径
	defaultFileName := filepath.Join(projectDir, "etc", "conf.json")
	// fmt.Printf("defaultFileName: %s\n", defaultFileName)
	// p1 := path.Dir(exepath)
	// homepath := path.Dir(p1)
	// fmt.Printf("homepath %s\n", homepath)
	// defaultFileName := path.Join(path.Dir(homepath), "etc", "conf.json")

	f, err := os.Open(defaultFileName)
	if err != nil {
		fmt.Println("[Configuration Error]  Failed to open config file: %v", err)
		// logging.PrintLog(true, logging.ErrorLog, p)
		os.Exit(1)
		return false
	}
	defer f.Close()
	var system System
	byteValue, _ := ioutil.ReadAll(f)

	json.Unmarshal(byteValue, &system)
	for s := 0; s < len(system.Replicas); s++ {
		shardIDs = append(shardIDs, system.Replicas[s].Shard)
		tempIDs := make([]string, 0)
		for i := 0; i < len(system.Replicas[s].Config); i++ {
			tempIDs = append(tempIDs, system.Replicas[s].Config[i].ID)
			nodeIDs = append(nodeIDs, system.Replicas[s].Config[i].ID)

			if s == 0 {
				nodeGroupAIDs = append(nodeGroupAIDs, system.Replicas[s].Config[i].ID)
			} else if s == 1 {
				nodeGroupBIDs = append(nodeGroupBIDs, system.Replicas[s].Config[i].ID)
			}

			addr := system.Replicas[s].Config[i].Host + ":" + system.Replicas[s].Config[i].Port
			nodes[system.Replicas[s].Config[i].ID] = addr
			nodesReverse[addr] = system.Replicas[s].Config[i].ID
			portMap[system.Replicas[s].Config[i].ID] = ":" + system.Replicas[s].Config[i].Port
			shardMap[system.Replicas[s].Config[i].ID] = system.Replicas[s].Shard
		}
		shards[system.Replicas[s].Shard] = tempIDs
	}

	// fmt.Printf("[Configuration]  nodeGroupAIDs:%v, nodeGroupBIDS:%v\n", nodeGroupAIDs, nodeGroupBIDs)
	// fmt.Printf("[Configuration] nodeIDs:%v", nodeIDs)

	maxBatchSize = system.MaxBatchSize

	sleepTimer = system.SleepTimer
	volume = system.Volume
	add = system.Add
	clientTimer = system.ClientTimer
	broadcastTimer = system.BroadcastTimer

	verbose = system.Verbose
	evalMode = system.EvalMode
	evalInterval = system.EvalInterval
	local = system.Local
	cryptoOpt = system.CryptoOpt
	maliciousNode = system.MaliciousNode
	maliciousMode = system.MaliciousMode
	splitPorts = system.SplitPorts
	logOpt = system.LogOpt
	partChurn = system.PartChurn
	viewChange = system.ViewChange
	rotatingTime = system.RotatingTime

	//get malicious node id
	mnidList := system.MaliciousNID
	mlist := strings.Split(mnidList, ",")
	consensus = system.Consensus
	persistLevel = system.PersistLevel
	rbc = system.RBCType

	for i := 0; i < len(mlist); i++ {
		tmp, err := utils.StringToInt64(mlist[i])
		if err != nil {
			fmt.Printf("Incorrect list of malicious node. Please double check conf.json, %v, mlist %v", mlist[i], mlist)
			continue
		}
		maliciousNID = append(maliciousNID, tmp)
	}

	return true
}

func FetchLogOpt() int {
	return logOpt
}

func MaxBatchSize() int {
	return maxBatchSize
}

// IP address of a node
func FetchAddress(id string) string {
	return nodes[id]
}

func FetchClientAddress(id string) string {
	return clientnodes[id]
}

func FetchPort(id string) string {
	// log.Printf("portMap %v, portMap[%v]:%v", portMap, id, portMap[id])
	return portMap[id]
}

func FetchClientPort(id string) string {
	// log.Printf("clientportMap %v, clientportMap[%v]:%v", clientportMap, id, clientportMap[id])
	return clientportMap[id]
}

// Get list of nodes
func FetchNodes() []string {
	// fmt.Printf("[Configuration]  nodeGroupAIDs:%v\n", nodeGroupAIDs)
	return nodeIDs
}

// Get list of nodes -A
func FetchANodes() []string {
	// fmt.Printf("[Configuration]  nodeGroupAIDs:%v\n", nodeGroupAIDs)
	return nodeGroupAIDs
}

// Get list of nodes -B
func FetchBNodes() []string {
	// fmt.Printf("[Configuration]  nodeGroupBIDs:%v\n", nodeGroupBIDs)
	return nodeGroupBIDs
}

func FetchNumBNodes() int {
	return len(nodeGroupBIDs)
}

// Total number of replicas
func FetchNumReplicas() int {
	res := 0
	switch consensus {
	case 104:
		res = len(nodeGroupAIDs) + len(nodeGroupBIDs)
	default:
		res = len(nodes)
	}
	return res
}

func FetchDeltaNumReplicas() int {
	return len(nodeGroupAIDs) + len(nodeGroupBIDs)
}

func FetchNumReplicasGroupB() int {
	return len(nodeGroupBIDs)
}

// Get the id (string format) of a node given the address (ip and port number)
func FetchReplicaID(addr string) string {
	return nodesReverse[addr]
}

func FetchClientID(addr string) string {
	return clientnodesReverse[addr]
}

func FetchSleepTimer() int {
	return sleepTimer
}

func FetchVolume() int {
	return volume
}

func FetchAdd() int {
	return add
}

func FetchClientTimer() int {
	return clientTimer
}

func FetchBroadcastTimer() int {
	return broadcastTimer
}

func FetchVerbose() bool {
	return verbose
}

func IsViewChangeMode() bool {
	return viewChange
}

func EvalMode() int {
	return evalMode
}

func CryptoOption() int {
	return cryptoOpt
}

func EvalInterval() int {
	return evalInterval
}

func Local() bool {
	return local
}

func MaliciousNode() bool {
	return maliciousNode
}

func MaliciousMode() int {
	return maliciousMode
}

func MaliciousNID(nid int64) bool {
	for i := 0; i < len(maliciousNID); i++ {
		if nid == maliciousNID[i] {
			return true
		}
	}
	return false
}

func SplitPorts() bool {
	return splitPorts
}

func Consensus() int {
	return consensus
}

func GConsensus() int {
	exepath, _ := os.Executable()
	binDir := filepath.Dir(exepath)
	projectDir := filepath.Dir(binDir)
	defaultFileName := filepath.Join(projectDir, "etc", "conf.json")

	f, _ := os.Open(defaultFileName)
	defer f.Close()
	var system System
	byteValue, _ := ioutil.ReadAll(f)

	json.Unmarshal(byteValue, &system)
	consensus = system.Consensus

	return consensus
}

func PersistLevel() int {
	return persistLevel
}

func GPersistLevel() int {
	exepath, _ := os.Executable()
	binDir := filepath.Dir(exepath)
	projectDir := filepath.Dir(binDir)
	defaultFileName := filepath.Join(projectDir, "etc", "conf.json")

	f, _ := os.Open(defaultFileName)
	defer f.Close()
	var system System
	byteValue, _ := ioutil.ReadAll(f)

	json.Unmarshal(byteValue, &system)
	persistLevel = system.PersistLevel
	return persistLevel
}

func RBCType() int {
	return rbc
}

func ParticipationChurn() bool {
	return partChurn
}

func FetchRotatingTime() int { return rotatingTime }
