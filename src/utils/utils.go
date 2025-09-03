package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"strconv"
	"sync"
	"time"
)

type IntValue struct {
	m int
	sync.RWMutex
}

func (s IntValue) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	ser := IntToBytes(s.m)
	return ser, nil
}

func (s *IntValue) Deserialize(input []byte) error {
	s.Lock()
	s.m = BytesToInt(input)
	defer s.Unlock()
	return nil
}
func (s *IntValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = 0
}

func (s *IntValue) Get() int {
	s.Lock()
	defer s.Unlock()
	return s.m
}

func (s *IntValue) Increment() int {
	s.Lock()
	defer s.Unlock()
	s.m = s.m + 1
	return s.m
}

func (s *IntValue) Set(n int) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

type BoolValue struct {
	m bool
	sync.RWMutex
}

func (s *BoolValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = false
}

func (s *BoolValue) Get() bool {
	s.Lock()
	defer s.Unlock()
	return s.m
}

func (s *BoolValue) Set(n bool) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

type ByteValue struct {
	m []byte
	sync.RWMutex
}

func (s ByteValue) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return s.m, nil
}

func (s *ByteValue) Deserialize(n []byte) error {
	s.Set(n)
	return nil
}

func (s *ByteValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = nil
}

func (s *ByteValue) Get() []byte {
	s.RLock()
	defer s.RUnlock()
	return s.m
}

func (s *ByteValue) Set(n []byte) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

type Vec struct {
	v []int
	sync.Mutex
}

func (s *Vec) Init(size int) {
	s.Lock()
	defer s.Unlock()
	var tmp []int
	s.v = tmp
	for i := 0; i < size; i++ {
		s.v = append(s.v, 0)
	}
}

func (s *Vec) Set(idx int, val int) {
	s.Lock()
	defer s.Unlock()
	s.v[idx] = val
}

func (s *Vec) Get() []int {
	s.Lock()
	defer s.Unlock()
	return s.v
}

func (s *Vec) GetIdx(idx int) int {
	s.Lock()
	defer s.Unlock()
	return s.v[idx]
}

func BytesToInt(bys []byte) int {
	bytebuff := bytes.NewBuffer(bys)
	var data int64
	binary.Read(bytebuff, binary.BigEndian, &data)
	return int(data)
}

func Int64ToString(input int64) string {
	return strconv.FormatInt(input, 10)
}

func StringToInt(input string) (int, error) {
	return strconv.Atoi(input)
}

func IntToString(input int) string {
	return strconv.Itoa(input)
}

func StringToInt64(input string) (int64, error) {
	return strconv.ParseInt(input, 10, 64)
}

func StringToBytes(input string) []byte {
	return []byte(input)
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func BytesToString(input []byte) string {
	return string(input[:])
}

func IntToInt64(input int) int64 {
	return int64(input)
}

func IntToBytes(n int) []byte {
	data := int64(n)
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}

// func BytestoInts(b []byte) []int{
// 	var output []byte
// 	for i:=0; i<len(b); i++{
// 		output = binary.BigEndian.Uint64(b)
// 	}
// 	return output
// }

func Int64ToInt(input int64) (int, error) {
	tmp := strconv.FormatInt(input, 10)
	output, err := strconv.Atoi(tmp)
	return output, err
}

func SerializeBytes(input [][]byte) []byte {
	if len(input) == 0 {
		return nil
	}
	var output []byte
	for i := 0; i < len(input); i++ {
		output = append(output, input[i]...)
	}
	return output
}

func SerializeBytesByBuffer(data [][]byte) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(data)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func DeserializeBytesByBuffer(data []byte, result *[][]byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	return decoder.Decode(result)
}
