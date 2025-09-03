package utils

import (
	"bytes"
	"log"
	"sort"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

type IntVec struct {
	m map[int][]int
	sync.RWMutex
}

func (s *IntVec) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int][]int)
}

func (s *IntVec) Insert(key int, value []int) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *IntVec) Get(key int) []int {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key]
	}
	return nil
}

type IntIntSliceMap struct {
	m map[int][]int
	sync.RWMutex
}

func (s *IntIntSliceMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int][]int)
}

func (s *IntIntSliceMap) InsertInt(key int, value int) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = append(s.m[key], value)
}

func (s *IntIntSliceMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

func (s *IntIntSliceMap) Get(key int) ([]int, bool) {
	s.RLock()
	defer s.RUnlock()
	value, exist := s.m[key]
	return value, exist
}

func (s *IntIntSliceMap) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s.m)
}

func (s *IntIntSliceMap) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s.m)
}

type StringBoolMap struct {
	m map[string]bool
	sync.RWMutex
}

func (s *StringBoolMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[string]bool)
}

func (s *StringBoolMap) Delete(key string) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		delete(s.m, key)
	}
}

func (s *StringBoolMap) Get(key string) (bool, bool) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key], true
	}
	return false, false
}

func (s *StringBoolMap) Insert(key string, value bool) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *StringBoolMap) GetAll() map[string]bool {
	return s.m
}

func (s *StringBoolMap) StringBoolMapList() []string {
	s.Lock()
	defer s.Unlock()
	list := []string{}
	for item := range s.m {
		list = append(list, item)
	}
	return list
}

func (s *StringBoolMap) SetValue(tagSet []string) {
	for _, v := range tagSet {
		s.Insert(v, true)
	}
}

func (s *StringBoolMap) IsTrue(item string) bool {
	s.Lock()
	defer s.Unlock()
	if s.m[item] == true {
		return true
	}
	return false
}

func (s *StringBoolMap) GetAllAndInit() map[string]bool {
	s.Lock()
	defer s.Unlock()
	defer func() {
		s.m = make(map[string]bool)
	}()
	return s.m
}

type StringIntMap struct {
	m map[string]int
	sync.RWMutex
}

func (s *StringIntMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[string]int)
}

func (s *StringIntMap) Get(key string) (int, bool) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key], true
	}
	return 0, false
}

func (s *StringIntMap) GetAll() map[string]int {
	return s.m
}

func (s *StringIntMap) Insert(key string, value int) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *StringIntMap) Delete(key string) {
	_, exist := s.Get(key)
	if exist {
		s.Lock()
		defer s.Unlock()
		delete(s.m, key)
	}

}

type IntSetMap struct {
	m map[int]IntSet
	sync.RWMutex
}

func (s *IntSetMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]IntSet)
}

func (s *IntSetMap) Insert(key int, value int) {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		ns := *NewIntSet()
		ns.AddItem(value)
		s.m[key] = ns
	}
	if es.Len() == 0 {
		es = *NewIntSet()
	}
	es.AddItem(value)
	s.m[key] = es
}

func (s *IntSetMap) Contains(key int, value int) bool {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		return false
	}
	return es.IsTrue(value)
}

func (s *IntSetMap) Get(key int) []int {
	s.RLock()
	defer s.RUnlock()
	es, exist := s.m[key]
	if !exist {
		var empty []int
		return empty
	}
	return es.IntSetList()
}

func (s *IntSetMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

func (s *IntSetMap) GetLen(key int) int {
	s.RLock()
	defer s.RUnlock()
	es, exist := s.m[key]
	if !exist {
		return 0
	}
	return es.Len()
}

func (s *IntSetMap) GetLenAndVal(key int) (int, int) {
	s.RLock()
	defer s.RUnlock()
	es, exist := s.m[key]
	if !exist {
		return 0, 0
	}
	return es.Len(), es.IntSetList()[0]
}

func (s *IntSetMap) GetCount(key int, value int) int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		return 0
	}
	return es.GetCount(value)
}

type IntSetMap_N struct {
	m map[int]IntSet_N
}

func (s *IntSetMap_N) Init() {
	s.m = make(map[int]IntSet_N)
}

func (s *IntSetMap_N) Insert(key int, value int) {
	es, exist := s.m[key]
	if !exist {
		ns := *NewIntSet_N()
		ns.AddItem(value)
		s.m[key] = ns
		return
	}
	if es.Len() == 0 {
		es = *NewIntSet_N()
	}
	es.AddItem(value)
	s.m[key] = es
}

func (s *IntSetMap_N) Contains(key int, value int) bool {
	es, exist := s.m[key]
	if !exist {
		return false
	}
	return es.IsTrue(value)
}

func (s *IntSetMap_N) Get(key int) []int {
	es, exist := s.m[key]
	if !exist {
		var empty []int
		return empty
	}
	return es.IntSetList()
}

func (s *IntSetMap_N) Delete(key int) {
	delete(s.m, key)
}

func (s *IntSetMap_N) GetLen(key int) int {
	es, exist := s.m[key]
	if !exist {
		return 0
	}
	return es.Len()
}

func (s *IntSetMap_N) GetLenAndVal(key int) (int, int) {
	es, exist := s.m[key]
	if !exist {
		return 0, 0
	}
	return es.Len(), es.IntSetList()[0]
}

func (s *IntSetMap_N) GetCount(key int, value int) int {
	es, exist := s.m[key]
	if !exist {
		return 0
	}
	return es.GetCount(value)
}

type IntDoubleSetMap struct {
	m map[int]Set
	q map[int]Set
	sync.RWMutex
}

func (s *IntDoubleSetMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]Set)
	s.q = make(map[int]Set)
}

func (s *IntDoubleSetMap) Insert(key int, input int, value int64) {
	s.Lock()
	defer s.Unlock()
	switch input {
	case 0:
		es, exist := s.m[key]
		if !exist {
			ns := *NewSet()
			ns.AddItem(value)
			s.m[key] = ns
		}
		if es.Len() == 0 {
			es = *NewSet()
		}
		es.AddItem(value)
		s.m[key] = es
	case 1:
		es, exist := s.q[key]
		if !exist {
			ns := *NewSet()
			ns.AddItem(value)
			s.q[key] = ns
		}
		if es.Len() == 0 {
			es = *NewSet()
		}
		es.AddItem(value)
		s.q[key] = es
	}

}

func (s *IntDoubleSetMap) GetCount(key int, input int) int {
	s.Lock()
	defer s.Unlock()
	switch input {
	case 0:
		es, exist := s.m[key]
		if !exist {
			return 0
		}
		return es.Len()
	case 1:
		es, exist := s.q[key]
		if !exist {
			return 0
		}
		return es.Len()
	}
	return 0
}

func (s *IntDoubleSetMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
	delete(s.q, key)
}

type IntIntDoubleSetMap struct {
	m map[int]IntDoubleSetMap
	sync.RWMutex
}

func (s *IntIntDoubleSetMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]IntDoubleSetMap)
}

func (s *IntIntDoubleSetMap) Insert(index int, rindex int, input int, value int64) {
	s.Lock()
	defer s.Unlock()
	es, exi := s.m[index]
	if !exi {
		var newtmp IntDoubleSetMap
		newtmp.Init()
		newtmp.Insert(rindex, input, value)
		s.m[index] = newtmp
		return
	}
	es.Insert(rindex, input, value)
	s.m[index] = es
}

func (s *IntIntDoubleSetMap) GetCount(index int, rindex int, input int) int {
	s.Lock()
	defer s.Unlock()
	es, exi := s.m[index]
	if !exi {
		return 0
	}
	return es.GetCount(rindex, input)
}

func (s *IntIntDoubleSetMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

type IntInt64SetMap struct {
	m map[int]Set
	sync.RWMutex
}

func (s *IntInt64SetMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]Set)
}

func (s *IntInt64SetMap) Insert(key int, value int64) {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		ns := *NewSet()
		ns.AddItem(value)
		s.m[key] = ns
	}
	if es.Len() == 0 {
		es = *NewSet()
	}
	es.AddItem(value)
	s.m[key] = es
}

func (s *IntInt64SetMap) Contains(key int, value int64) bool {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		return false
	}
	return es.HasItem(value)
}

func (s *IntInt64SetMap) Get(key int) []int64 {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		var empty []int64
		return empty
	}
	return es.SetList()
}

func (s *IntInt64SetMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

func (s *IntInt64SetMap) GetLen(key int) int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		return 0
	}
	return es.Len()
}

type IntInt64SetMap_N struct {
	m map[int]Set_N
	sync.RWMutex
}

func (s *IntInt64SetMap_N) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]Set_N)
}

func (s *IntInt64SetMap_N) Insert(key int, value int64) {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		ns := *NewSet_N()
		ns.AddItem(value)
		s.m[key] = ns
	}
	if es.Len() == 0 {
		es = *NewSet_N()
	}
	es.AddItem(value)
	s.m[key] = es
}

func (s *IntInt64SetMap_N) Contains(key int, value int64) bool {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		return false
	}
	return es.HasItem(value)
}

func (s *IntInt64SetMap_N) Get(key int) []int64 {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		var empty []int64
		return empty
	}
	return es.SetList()
}

func (s *IntInt64SetMap_N) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

func (s *IntInt64SetMap_N) GetLen(key int) int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[key]
	if !exist {
		return 0
	}
	return es.Len()
}

type IntBoolMap struct {
	m     map[int]bool
	count IntSet
	sync.RWMutex
}

func (s *IntBoolMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]bool)
	s.count = *NewIntSet()
}

func (s *IntBoolMap) Get(key int) (bool, bool) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key], true
	}
	return false, false
}

func (s *IntBoolMap) GetStatus(key int) bool {
	s.Lock()
	defer s.Unlock()
	v, exist := s.m[key]
	if exist {
		return v
	}
	return false
}

func (s *IntBoolMap) GetAll() map[int]bool {
	s.Lock()
	defer s.Unlock()
	item := make(map[int]bool)
	for key, val := range s.m {
		item[key] = val
	}
	return item
}

func (s *IntBoolMap) Insert(key int, value bool) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
	s.count.AddItem(key)
}

func (s *IntBoolMap) GetCount() int {
	s.Lock()
	defer s.Unlock()
	return s.count.Len()
}

func (s *IntBoolMap) GetLen() int {
	s.Lock()
	defer s.Unlock()
	return len(s.m)
}

func (s *IntBoolMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		delete(s.m, key)
		s.count.RemoveItem(key)
	}
}

type IntIntMap struct {
	m map[int]int
	sync.RWMutex
}

func (s IntIntMap) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s.m)
}

func (s *IntIntMap) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s.m)
}

func (s *IntIntMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]int)
}

func (s *IntIntMap) Get(key int) (int, bool) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key], true
	}
	return 0, false
}

func (s *IntIntMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

func (s *IntIntMap) GetAll() map[int]int {
	return s.m
}

func (s *IntIntMap) Insert(key int, value int) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *IntIntMap) Increment(key int) int {
	s.Lock()
	defer s.Unlock()
	v, exist := s.m[key]
	if !exist {
		s.m[key] = 1
		return 1
	}
	v = v + 1
	s.m[key] = v
	return v
}

func (s *IntIntMap) Set(key int, v int) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = v
}

func (s *IntIntMap) IsExist(key int, v int) bool {
	s.Lock()
	defer s.Unlock()
	tmp, exi := s.m[key]
	if !exi {
		return false
	}
	return tmp == v
}

type IntBytesMapArr struct {
	msg []IntBytesMap
	sync.Mutex
}

func (s *IntBytesMapArr) Init(n int) {
	s.Lock()
	defer s.Unlock()
	for i := 0; i < n; i++ {
		var tmp IntBytesMap
		tmp.Init()
		s.msg = append(s.msg, tmp)
	}
}

func (s *IntBytesMapArr) Get(index int) *IntBytesMap {
	s.Lock()
	defer s.Unlock()
	if index >= len(s.msg) {
		var tmp IntBytesMap
		return &tmp
	}
	return &s.msg[index]
}

func (s *IntBytesMapArr) Insert(index int, input *IntBytesMap) {
	s.Lock()
	defer s.Unlock()
	if index >= len(s.msg) {
		return
	}
	s.msg[index] = *input
}

func (s *IntBytesMapArr) InsertValue(index int, rindex int, msg []byte) {
	s.Lock()
	defer s.Unlock()
	if index >= len(s.msg) {
		return
	}
	tmp := s.msg[index]
	tmp.Insert(rindex, msg)
	s.msg[index] = tmp
}

func (s *IntBytesMapArr) GetAndClear(index int, rindex int) [][]byte {
	s.Lock()
	defer s.Unlock()
	if index >= len(s.msg) {
		var tmpresult [][]byte
		return tmpresult
	}
	tmp := s.msg[index]
	msgs, _ := tmp.GetAndClear(rindex)
	s.msg[index] = tmp
	return msgs
}

// type IntIntByteMap struct {
// 	m map[int]IntByteMap
// 	sync.RWMutex
// }

// func (s *IntIntByteMap) Init(n int) {
// 	s.Lock()
// 	defer s.Unlock()
// 	for i := 0; i < n; i++ {
// 		var tmp IntBytesMap
// 		tmp.Init()
// 		s.msg = append(s.msg, tmp)
// 	}
// }

// func (s *IntIntByteMap) Insert(index int, rindex int, value []byte) {
// 	s.Lock()
// 	defer s.Unlock()
// 	if _, exists := s.m[index]; !exists {
// 		s.m[index] = make(map[int][]byte)
// 	}
// 	s.m[index][rindex] = value
// }

// func (s *IntIntByteMap) Get(index int, rindex int) ([]byte, bool) {
// 	s.RLock()
// 	defer s.RUnlock()
// 	if _, exists := s.m[index]; exists {
// 		value, exists := s.m[index][rindex]
// 		return value, exists
// 	}
// 	return nil, false
// }

// func (s *IntIntByteMap) Delete(index int, rindex int) {
// 	s.Lock()
// 	defer s.Unlock()
// 	if _, exists := s.m[index]; exists {
// 		delete(s.m[index], rindex)
// 		if len(s.m[index]) == 0 {
// 			delete(s.m, index)
// 		}
// 	}
// }

// func (s *IntIntByteMap) GetAll(index int) (map[int][]byte, bool) {
// 	s.RLock()
// 	defer s.RUnlock()
// 	value, exists := s.m[index]
// 	return value, exists
// }

// func (s *IntIntByteMap) GetLen(index int) int {
// 	s.RLock()
// 	defer s.RUnlock()
// 	if _, exists := s.m[index]; exists {
// 		return len(s.m[index])
// 	}
// 	return 0
// }

// func (s *IntIntByteMap) GetRLen(index int, rindex int) int {
// 	s.RLock()
// 	defer s.RUnlock()
// 	if _, exists := s.m[index]; exists {
// 		return len(s.m[index])
// 	}
// 	return 0
// }

type IntIntBytesMapArr struct {
	msg map[int]IntBytesMap
	sync.Mutex
}

func (s *IntIntBytesMapArr) Init(n int) {
	s.Lock()
	defer s.Unlock()
	s.msg = make(map[int]IntBytesMap)
}

func (s *IntIntBytesMapArr) Get(index int) *IntBytesMap {
	s.Lock()
	defer s.Unlock()
	data, exi := s.msg[index]
	if !exi {
		var tmp IntBytesMap
		return &tmp
	}
	return &data
}

func (s *IntIntBytesMapArr) Insert(index int, input *IntBytesMap) {
	s.Lock()
	defer s.Unlock()
	s.msg[index] = *input
}

func (s *IntIntBytesMapArr) InsertValue(index int, rindex int, msg []byte) {
	s.Lock()
	defer s.Unlock()

	tmp, exi := s.msg[index]
	if !exi {
		var newtmp IntBytesMap
		newtmp.Init()
		newtmp.Insert(rindex, msg)
		s.msg[index] = newtmp
		return
	}
	tmp.Insert(rindex, msg)
	s.msg[index] = tmp
}

func (s *IntIntBytesMapArr) GetAndClear(index int, rindex int) [][]byte {
	s.Lock()
	defer s.Unlock()

	tmp, exi := s.msg[index]
	if !exi {
		var tmpresult [][]byte
		return tmpresult
	}
	msgs, _ := tmp.GetAndClear(rindex)
	s.msg[index] = tmp
	return msgs
}

func (s *IntIntBytesMapArr) InsertValueAndInt(index int, rindex int, value []byte, intval int64) {
	s.Lock()
	defer s.Unlock()

	tmp, exi := s.msg[index]
	if !exi {
		var newtmp IntBytesMap
		newtmp.Init()
		newtmp.InsertValueAndInt(rindex, value, intval)
		s.msg[index] = newtmp
		return
	}
	tmp.InsertValueAndInt(rindex, value, intval)
	s.msg[index] = tmp
}

func (s *IntIntBytesMapArr) GetAllValue(index int, rindex int) ([]int64, [][]byte) {
	s.Lock()
	defer s.Unlock()

	tmp, exi := s.msg[index]
	if !exi {
		var tmpresult1 []int64
		var tmpresult2 [][]byte
		return tmpresult1, tmpresult2
	}
	return tmp.GetAllValue(rindex)
}

type IntIntMapArr struct {
	msg map[int]IntIntMap
	sync.RWMutex
}

func (s *IntIntMapArr) Init() {
	s.Lock()
	defer s.Unlock()
	s.msg = make(map[int]IntIntMap)
}

func (s *IntIntMapArr) Get(index int) *IntIntMap {
	s.Lock()
	defer s.Unlock()
	data, exi := s.msg[index]
	if !exi {
		var tmp IntIntMap
		return &tmp
	}
	return &data
}

func (s *IntIntMapArr) Insert(index int, input *IntIntMap) {
	s.Lock()
	defer s.Unlock()
	s.msg[index] = *input
}

func (s *IntIntMapArr) InsertValue(index int, rindex int, msg int) {
	s.Lock()
	defer s.Unlock()

	tmp, exi := s.msg[index]
	if !exi {
		var newtmp IntIntMap
		newtmp.Init()
		newtmp.Insert(rindex, msg)
		s.msg[index] = newtmp
		return
	}
	tmp.Insert(rindex, msg)
	s.msg[index] = tmp
}

func (s *IntIntMapArr) GetValue(index int, rindex int) int {
	s.RLock()
	defer s.RUnlock()

	tmp, exi := s.msg[index]
	if !exi {
		return -1
	}
	v, e := tmp.Get(rindex)
	if !e {
		return -1
	}
	return v
}

func (s *IntIntMapArr) Contains(key int, key1 int, value int) bool {
	s.Lock()
	defer s.Unlock()
	es, exist := s.msg[key]
	if !exist {
		return false
	}

	return es.IsExist(key1, value)
}

type IntBytesMap struct {
	m map[int][][]byte
	v map[int][]int64
	c map[int]int
	sync.Mutex
}

func (s *IntBytesMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int][][]byte)
	s.v = make(map[int][]int64)
	s.c = make(map[int]int)
}

func (s *IntBytesMap) InitKey(initkey int, maxkey int, size int) {
	s.Lock()
	defer s.Unlock()
	tmp := make([][]byte, size)
	for i := 0; i < size; i++ {
		tmp[i] = make([]byte, size)
	}
	for i := initkey; i < maxkey; i++ {
		s.m[i] = tmp
	}
}

func (s *IntBytesMap) Get(key int) ([][]byte, bool) {
	s.Lock()
	defer s.Unlock()
	v, exist := s.m[key]
	if exist {
		return v, true
	}
	var emptybytes [][]byte
	return emptybytes, false
}

func (s *IntBytesMap) GetByIndex(key int, idx int) []byte {
	s.Lock()
	defer s.Unlock()
	v, exist := s.m[key]
	if !exist || len(v) < idx {
		return nil
	}
	return v[idx]
}

func (s *IntBytesMap) SetValue(key int, value [][]byte) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *IntBytesMap) GetLen(key int) int {
	s.Lock()
	defer s.Unlock()
	v, exist := s.m[key]
	if exist {
		return 0
	}
	return len(v)
}

func (s *IntBytesMap) GetAndClear(key int) ([][]byte, bool) {
	s.Lock()
	defer s.Unlock()
	var emptybytes [][]byte
	v, exist := s.m[key]
	if exist {
		delete(s.m, key)
		return v, true
	}
	return emptybytes, false
}

func (s *IntBytesMap) Remove(key int, value []byte) {
	s.Lock()
	defer s.Unlock()
	val, exist := s.m[key]
	if !exist {
		return
	}
	newval := val
	for i := 0; i < len(val); i++ {
		if bytes.Compare(val[i], value) != 0 {
			newval = append(newval, val[i])
		}
	}
	s.m[key] = newval
}

func (s *IntBytesMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		delete(s.m, key)
	}
	_, exist = s.v[key]
	if exist {
		delete(s.v, key)
	}
}

func (s *IntBytesMap) Insert(key int, value []byte) {
	s.Lock()
	defer s.Unlock()
	val, exist := s.m[key]
	if !exist {
		var v [][]byte
		v = append(v, value)
		s.m[key] = v
		return
	}
	val = append(val, value)
	s.m[key] = val
}

func (s *IntBytesMap) InsertValue(key int, idx int, value []byte) {
	s.Lock()
	defer s.Unlock()
	val, exist := s.m[key]
	if !exist {
		log.Printf("lenval %v %v", len(val), idx)
		if len(val) <= idx {

			for i := 0; i < idx-len(val)+1; i++ {
				val = append(val, nil)
			}
			log.Printf("lenval %v", len(val))
			val[idx] = value
		} else {
			val[idx] = value
		}

		s.m[key] = val
		s.c[key] = 1
		return
	}
	if len(val) < idx {
		return
	}
	if value != nil {
		s.c[key] += 1
	}
	val[idx] = value
	s.m[key] = val
}

func (s *IntBytesMap) GetCount(key int) int {
	s.Lock()
	defer s.Unlock()
	val, _ := s.c[key]
	return val
}

func (s *IntBytesMap) GetAllValue(key int) ([]int64, [][]byte) {
	s.Lock()
	defer s.Unlock()
	var output1 []int64
	var output2 [][]byte

	v, exist := s.m[key]
	intv, exist1 := s.v[key]
	if exist {
		output2 = v
	}
	if exist1 {
		output1 = intv
	}

	return output1, output2
}

func (s *IntBytesMap) InsertValueAndInt(key int, value []byte, intval int64) {
	s.Lock()
	defer s.Unlock()
	val, exist := s.m[key]
	intv, exist2 := s.v[key]
	if !exist {
		var val [][]byte
		val = append(val, value)
		s.m[key] = val
	} else {
		val = append(val, value)
		s.m[key] = val
	}

	if !exist2 {
		var val []int64
		val = append(val, intval)
		s.v[key] = val
	} else {
		intv = append(intv, intval)
		s.v[key] = intv
	}
}

type IntMulByteMap struct {
	m map[int][][]byte
	sync.RWMutex
}

func (s *IntMulByteMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int][][]byte)
}

func (s *IntMulByteMap) Insert(key int, num int, value []byte) {
	s.Lock()
	defer s.Unlock()

	// Ensure the map is initialized
	if s.m == nil {
		s.m = make(map[int][][]byte)
	}

	// Initialize or expand the slice
	if _, exist := s.m[key]; !exist {
		s.m[key] = make([][]byte, num+1)
	} else if num >= len(s.m[key]) {
		newSize := max(len(s.m[key])*2, num+1)
		newSlice := make([][]byte, newSize)
		copy(newSlice, s.m[key])
		s.m[key] = newSlice
	}

	s.m[key][num] = value

	// Debug: Log the state of the map after insertion
	// p := fmt.Sprintf("[DEBUG-Insert] Key: %v, Num: %v, Value: %v, Map State: %v\n", key, num, value, s.m)
	// logging.PrintLog(true, logging.NormalLog, p)
}

func (s *IntMulByteMap) GetByInt(key int) ([][]byte, bool) {
	s.RLock()
	defer s.RUnlock()

	// Check if the key exists
	values, exist := s.m[key]
	if !exist {
		// Debug: Log if the key is not found
		// p := fmt.Sprintf("[DEBUG-GetByInt] Key: %v not found in map\n", key)
		// logging.PrintLog(true, logging.NormalLog, p)
		return nil, false
	}

	// // Debug: Log the retrieved values
	// p := fmt.Sprintf("[DEBUG-GetByInt] Key: %v, Retrieved Values: %v\n", key, values)
	// logging.PrintLog(true, logging.NormalLog, p)
	return values, true
}

func (s IntMulByteMap) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s.m)
}

func (s *IntMulByteMap) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s.m)
}

func (s *IntMulByteMap) GetByIntNum(key int, num int) ([]byte, bool) {
	s.RLock()
	defer s.RUnlock()
	values, exist := s.m[key]
	if !exist {
		return nil, false
	}

	if num < 0 || num >= len(values) {
		return nil, false
	}

	return values[num], true
}

func (s *IntMulByteMap) GetAll() map[int][][]byte {
	return s.m
}

func (s *IntMulByteMap) GetLenByInt(key int) int {
	s.RLock()
	defer s.RUnlock()

	// Count non-empty []byte entries
	count := 0
	for _, value := range s.m[key] {
		if len(value) > 0 {
			count++
		}
	}
	return count
}

func (s *IntMulByteMap) DeleteByKey(key int, num int) {
	s.Lock()
	defer s.Unlock()

	values, exist := s.m[key]
	if !exist || num < 0 || num >= len(values) {
		return
	}

	// 删除元素（置 nil 防止内存泄漏）
	values[num] = nil

	// 缩容判断（当利用率低于25%时缩容一半）
	if len(values) > 0 && countNonNil(values) <= len(values)/4 {
		newSize := len(values) / 2
		newValues := make([][]byte, newSize)
		copy(newValues, values[:newSize])
		s.m[key] = newValues
	}
}

// 统计非空元素数量
func countNonNil(slice [][]byte) int {
	count := 0
	for _, v := range slice {
		if v != nil {
			count++
		}
	}
	return count
}

type IntByteMap struct {
	m map[int][]byte
	sync.RWMutex
}

func (s IntByteMap) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s.m)
}

func (s *IntByteMap) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s.m)
}

func (s *IntByteMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int][]byte)
}

func (s *IntByteMap) Get(key int) ([]byte, bool) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		return s.m[key], true
	}
	return []byte(""), false
}

func (s *IntByteMap) GetAll() map[int][]byte {
	return s.m
}

func (s *IntByteMap) GetLen() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.m)
}

func (s *IntByteMap) Insert(key int, value []byte) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *IntByteMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		delete(s.m, key)
	}
}

func (s *IntByteMap) GetSortedKeysDesc() []int {
	s.RLock()
	defer s.RUnlock()
	keys := make([]int, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys))) // 降序排序
	return keys
}

type IntIntSetMap struct {
	m map[int]IntSetMap_N
	sync.RWMutex
}

func (s *IntIntSetMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]IntSetMap_N)
}

func (s *IntIntSetMap) Get(index int) *IntSetMap_N {
	s.Lock()
	defer s.Unlock()
	data, exi := s.m[index]
	if !exi {
		var tmp IntSetMap_N
		return &tmp
	}
	return &data
}

func (s *IntIntSetMap) GetValue(index int, rindex int) []int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		var empty []int
		return empty
	}
	return es.Get(rindex)
}

func (s *IntIntSetMap) Insert(index int, input *IntSetMap_N) {
	s.Lock()
	defer s.Unlock()
	s.m[index] = *input
}

func (s *IntIntSetMap) InsertValue(index int, rindex int, msg int) {
	s.Lock()
	defer s.Unlock()

	tmp, exi := s.m[index]
	if !exi {
		var newtmp IntSetMap_N
		newtmp.Init()
		newtmp.Insert(rindex, msg)
		s.m[index] = newtmp
		return
	}
	tmp.Insert(rindex, msg)
	s.m[index] = tmp
}

func (s *IntIntSetMap) Contains(index int, rindex int, value int) bool {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		return false
	}

	return es.Contains(rindex, value)
}

func (s *IntIntSetMap) GetLen(index int, rindex int) int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		return 0
	}
	return es.GetLen(rindex)
}

func (s *IntIntSetMap) GetCount(index int, rindex int, value int) int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		return 0
	}
	return es.GetCount(rindex, value)
}

func (s *IntIntSetMap) GetLenAndVal(index int, rindex int) (int, int) {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		return 0, 0
	}
	return es.GetLenAndVal(rindex)
}

func (s *IntIntSetMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	_, exist := s.m[key]
	if exist {
		delete(s.m, key)
	}
}

type IntIntInt64SetMap struct {
	m map[int]IntInt64SetMap_N
	sync.RWMutex
}

func (s *IntIntInt64SetMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]IntInt64SetMap_N)
}

func (s *IntIntInt64SetMap) Insert(index int, rindex int, value int64) {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		var newtmp IntInt64SetMap_N
		newtmp.Init()
		newtmp.Insert(rindex, value)
		s.m[index] = newtmp
		return
	}
	es.Insert(rindex, value)
	s.m[index] = es
}

func (s *IntIntInt64SetMap) Contains(index int, rindex int, value int64) bool {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		return false
	}
	return es.Contains(rindex, value)
}

func (s *IntIntInt64SetMap) Get(index int, rindex int) []int64 {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		var empty []int64
		return empty
	}
	return es.Get(rindex)
}

func (s *IntIntInt64SetMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}

func (s *IntIntInt64SetMap) GetLen(index int, rindex int) int {
	s.Lock()
	defer s.Unlock()
	es, exist := s.m[index]
	if !exist {
		return 0
	}
	return es.GetLen(rindex)
}

type IntByteIntByteMap struct {
	v  map[int][]byte
	pi map[int][]byte
	sync.RWMutex
}

func (s *IntByteIntByteMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.v = make(map[int][]byte)
	s.pi = make(map[int][]byte)
}

func (s *IntByteIntByteMap) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s)
}

func (s *IntByteIntByteMap) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s)
}

func (s *IntByteIntByteMap) InsertV(epoch int, value []byte) {
	s.Lock()
	defer s.Unlock()
	s.v[epoch] = value
}

func (s *IntByteIntByteMap) InsertPI(epoch int, value []byte) {
	s.Lock()
	defer s.Unlock()
	s.pi[epoch] = value
}

func (s *IntByteIntByteMap) Delete(epoch int) {
	s.Lock()
	defer s.Unlock()
	delete(s.v, epoch)
	delete(s.pi, epoch)
}

func (s *IntByteIntByteMap) GetV(epoch int) ([]byte, bool) {
	s.RLock()
	defer s.RUnlock()
	value, exist := s.v[epoch]
	return value, exist
}

func (s *IntByteIntByteMap) GetPI(epoch int) ([]byte, bool) {
	s.RLock()
	defer s.RUnlock()
	value, exist := s.pi[epoch]
	return value, exist
}

func (s *IntByteIntByteMap) GetAll() (map[int][]byte, map[int][]byte) {
	s.RLock()
	defer s.RUnlock()
	vCopy := make(map[int][]byte)
	piCopy := make(map[int][]byte)
	for k, v := range s.v {
		vCopy[k] = v
	}
	for k, v := range s.pi {
		piCopy[k] = v
	}
	return vCopy, piCopy
}

func (s *IntByteIntByteMap) GetSortedKeys() []int {
	s.RLock()
	defer s.RUnlock()
	keys := make([]int, 0, len(s.v))
	for k := range s.v {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func (s *IntByteIntByteMap) GetSorted() ([]int, [][]byte, [][]byte) {
	s.RLock()
	defer s.RUnlock()
	keys := s.GetSortedKeys()
	vSorted := make([][]byte, len(keys))
	piSorted := make([][]byte, len(keys))
	for i, k := range keys {
		vSorted[i] = s.v[k]
		piSorted[i] = s.pi[k]
	}
	return keys, vSorted, piSorted
}

type ByteByteMap struct {
	Byte1 []byte
	Byte2 []byte
}

type IntsBytesMap struct {
	BytesMap map[int]ByteByteMap
	IntList  []int
	sync.RWMutex
}

// Init 初始化 IntsBytesMap 实例，设置 epochNo 为 -1，pastMsg 和 pastSig 为空
func (IBM *IntsBytesMap) Init() {
	IBM.Lock()
	defer IBM.Unlock()

	IBM.BytesMap = make(map[int]ByteByteMap)
	IBM.IntList = []int{}

	// // 直接初始化，不调用 Add 方法
	// IBM.BytesMap[-1] = ByteByteMap{Byte1: []byte{}, Byte2: []byte{}}
	// IBM.IntList = append(IBM.IntList, -1)
}

// Add 添加新的 epochNo, pastMsg 和 pastSig
func (IBM *IntsBytesMap) Add(intValue int, byteValue1, byteValue2 []byte) {
	IBM.Lock()
	defer IBM.Unlock()

	if _, exists := IBM.BytesMap[intValue]; !exists {
		IBM.IntList = append(IBM.IntList, intValue)
	}
	IBM.BytesMap[intValue] = ByteByteMap{Byte1: byteValue1, Byte2: byteValue2}
	sort.Ints(IBM.IntList)
}

// GetSortedKeys 获取按升序排序的 epochNo 列表
func (IBM *IntsBytesMap) GetSortedKeys() []int {
	IBM.RLock()
	defer IBM.RUnlock()

	return IBM.IntList
}

// GetLargestKey 获取最大的 epochNo
func (IBM *IntsBytesMap) GetLargestKey() (int, bool) {
	IBM.RLock()
	defer IBM.RUnlock()

	if len(IBM.IntList) == 0 {
		return -1, false
	}
	return IBM.IntList[len(IBM.IntList)-1], true
}

func (IBM *IntsBytesMap) GetSmallestKey() (int, bool) {
	IBM.RLock()
	defer IBM.RUnlock()

	if len(IBM.IntList) == 0 {
		return 0, false
	}
	return IBM.IntList[0], true
}

// Get 根据 epochNo 获取 pastMsg 和 pastSig
func (IBM *IntsBytesMap) Get(e int) (ByteByteMap, bool) {
	IBM.RLock()
	defer IBM.RUnlock()

	data, exists := IBM.BytesMap[e]
	return data, exists
}

func (IBM *IntsBytesMap) GetEpochList() []int {
	IBM.RLock()
	defer IBM.RUnlock()

	return IBM.IntList
}

// GetAll 获取所有存储的 epochNo 和对应的 PastData
func (IBM *IntsBytesMap) GetAll() map[int]ByteByteMap {
	IBM.RLock()
	defer IBM.RUnlock()

	return IBM.BytesMap
}

// IsEmpty 检查 IntsBytesMap 是否为空
func (IBM *IntsBytesMap) IsEmpty() bool {
	IBM.RLock()
	defer IBM.RUnlock()

	return len(IBM.IntList) == 0
}

// Serialize 将 IntsBytesMap 序列化为字节数组
func (IBM *IntsBytesMap) Serialize() ([]byte, error) {
	IBM.RLock()
	defer IBM.RUnlock()

	data, err := msgpack.Marshal(IBM)
	return data, err
}

// Deserialize 从字节数组反序列化为 IntsBytesMap
func (IBM *IntsBytesMap) Deserialize(dataSer []byte) (*IntsBytesMap, error) {
	data := &IntsBytesMap{}
	err := msgpack.Unmarshal(dataSer, data)
	return data, err
}

type ByteTripleMap struct {
	Byte1 []byte
	Byte2 []byte
	Byte3 []byte
}

// type IntsTripleBytesMap struct {
// 	BytesMap map[int]ByteTripleMap
// 	IntList  []int
// 	sync.RWMutex
// }

// // Init 初始化 IntsTripleBytesMap 实例，设置 epochNo 为 -1，三个 Byte 字段为空
// func (ITBM *IntsTripleBytesMap) Init() {
// 	ITBM.Lock()
// 	defer s.Unlock()

// 	ITBM.BytesMap = make(map[int]ByteTripleMap)
// 	ITBM.IntList = []int{}

// 	// 直接初始化，不调用 Add 方法
// 	ITBM.BytesMap[-1] = ByteTripleMap{Byte1: []byte{}, Byte2: []byte{}, Byte3: []byte{}}
// 	ITBM.IntList = append(ITBM.IntList, -1)
// }

// // Add 添加新的 epochNo 和三个 Byte 字段
// func (ITBM *IntsTripleBytesMap) Add(intValue int, byteValue1, byteValue2, byteValue3 []byte) {
// 	ITBM.Lock()
// 	defer ITBM.Unlock()

// 	if _, exists := ITBM.BytesMap[intValue]; !exists {
// 		ITBM.IntList = append(ITBM.IntList, intValue)
// 	}
// 	ITBM.BytesMap[intValue] = ByteTripleMap{Byte1: byteValue1, Byte2: byteValue2, Byte3: byteValue3}
// 	sort.Ints(ITBM.IntList)
// }

// // GetSortedKeys 获取按升序排序的 epochNo 列表
// func (ITBM *IntsTripleBytesMap) GetSortedKeys() []int {
// 	ITBM.RLock()
// 	defer ITBM.RUnlock()

// 	return ITBM.IntList
// }

// // GetLargestKey 获取最大的 epochNo
// func (ITBM *IntsTripleBytesMap) GetLargestKey() (int, bool) {
// 	ITBM.RLock()
// 	defer ITBM.RUnlock()

// 	if len(ITBM.IntList) == 0 {
// 		return 0, false
// 	}
// 	return ITBM.IntList[len(ITBM.IntList)-1], true
// }

// // Get 根据 epochNo 获取三个 Byte 字段
// func (ITBM *IntsTripleBytesMap) Get(e int) (ByteTripleMap, bool) {
// 	ITBM.RLock()
// 	defer ITBM.RUnlock()

// 	data, exists := ITBM.BytesMap[e]
// 	return data, exists
// }

// // GetAll 获取所有存储的 epochNo 和对应的 ByteTripleMap
// func (ITBM *IntsTripleBytesMap) GetAll() map[int]ByteTripleMap {
// 	ITBM.RLock()
// 	defer ITBM.RUnlock()

// 	return ITBM.BytesMap
// }

// // Serialize 将 IntsTripleBytesMap 序列化为字节数组
// func (ITBM *IntsTripleBytesMap) Serialize() ([]byte, error) {
// 	ITBM.RLock()
// 	defer ITBM.RUnlock()

// 	data, err := msgpack.Marshal(ITBM)
// 	return data, err
// }

// // Deserialize 从字节数组反序列化为 IntsTripleBytesMap
// func (ITBM *IntsTripleBytesMap) Deserialize(dataSer []byte) (*IntsTripleBytesMap, error) {
// 	data := &IntsTripleBytesMap{}
// 	err := msgpack.Unmarshal(dataSer, data)
// 	return data, err
// }

type ByteList struct {
	Bytes [][]byte
	sync.RWMutex
}

type IntByteListMap struct {
	m map[int]ByteList
	sync.RWMutex
}

func (s *ByteList) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s)
}

func (s *ByteList) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s)
}

func (s *IntByteListMap) Serialize() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return msgpack.Marshal(s.m)
}

func (s *IntByteListMap) Deserialize(input []byte) error {
	s.Lock()
	defer s.Unlock()
	return msgpack.Unmarshal(input, &s.m)
}

func (s *IntByteListMap) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = make(map[int]ByteList)
}

func (s *IntByteListMap) Get(key int) (ByteList, bool) {
	s.RLock()
	defer s.RUnlock()
	value, exist := s.m[key]
	return value, exist
}

func (s *IntByteListMap) GetAll() map[int]ByteList {
	s.RLock()
	defer s.RUnlock()
	return s.m
}

func (s *IntByteListMap) GetLen(key int) int {
	s.RLock()
	defer s.RUnlock()
	if value, exist := s.m[key]; exist {
		return len(value.Bytes)
	}
	return 0
}

func (s *IntByteListMap) Insert(key int, value []byte) {
	s.Lock()
	defer s.Unlock()
	if _, exist := s.m[key]; !exist {
		s.m[key] = ByteList{Bytes: [][]byte{}}
	}
	// 获取 ByteList 的副本，修改副本，然后再将副本放回 map 中
	byteList := s.m[key]
	byteList.Bytes = append(byteList.Bytes, value)
	s.m[key] = byteList
}

func (s *IntByteListMap) Delete(key int) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, key)
}
