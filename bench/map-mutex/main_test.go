package main

import (
	"strconv"
	"sync"
	"testing"
)

var MutexWorstCaseMap = map[string]map[string]map[string]map[string]int{}
var rwmutex = &sync.RWMutex{}
var mutex = &sync.Mutex{}
var isDone = make(chan bool, 10000000)

func main() {

	worstCaseMap := map[string]map[string]map[string]map[string]int32{}

	for i := 0; i < 1000; i++ {
		for j := 0; j < 1000; j++ {
			for k := 0; k < 1000; k++ {
				for l := 0; l < 1000; l++ {
					worstCaseMap["a"][strconv.Itoa(i)][strconv.Itoa(j)][strconv.Itoa(k)] = 1
				}
			}
		}
	}
}

func BenchmarkWorstCaseMap_Create(b *testing.B) {
	worstCaseMap := map[string]map[string]map[string]map[string]int{}

	for i := 0; i < b.N; i++ {
		if _, ok := worstCaseMap[strconv.Itoa(i)]; !ok {
			worstCaseMap[strconv.Itoa(i)] = map[string]map[string]map[string]int{}
		}
		if _, ok := worstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)]; !ok {
			worstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)] = map[string]map[string]int{}
		}
		if _, ok := worstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)]; !ok {
			worstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)] = map[string]int{}
		}
		worstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)] = 1
	}
}

func BenchmarkWorstCaseMap_Update(b *testing.B) {
	worstCaseMap := map[string]map[string]map[string]map[string]int{
		"a": {
			"b": {
				"c": {
					"d": 1,
				},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		worstCaseMap["a"]["b"]["c"]["d"]++
	}
}

func BenchmarkWorstCaseMap_Goroutines(b *testing.B) {
	for i := 0; i < b.N; i++ {
		go insertToMapRWMut(i)
	}
	for {
		if len(isDone) >= b.N {
			return
		}
	}
}
func BenchmarkWorstCaseMap_Goroutines2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		go insertToMapMut(i)
	}
	for {
		if len(isDone) >= b.N {
			return
		}
	}
}

func insertToMap(i int) {
	if _, ok := MutexWorstCaseMap[strconv.Itoa(i)]; !ok {
		MutexWorstCaseMap[strconv.Itoa(i)] = map[string]map[string]map[string]int{}
	}
	if _, ok := MutexWorstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)]; !ok {
		MutexWorstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)] = map[string]map[string]int{}
	}
	if _, ok := MutexWorstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)]; !ok {
		MutexWorstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)] = map[string]int{}
	}
	MutexWorstCaseMap[strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)][strconv.Itoa(i)] = 1
	isDone <- true
}

func insertToMapMut(i int) {
	mutex.Lock()
	defer mutex.Unlock()
	insertToMap(i)
}
func insertToMapRWMut(i int) {
	rwmutex.Lock()
	defer rwmutex.Unlock()
	insertToMap(i)
}
