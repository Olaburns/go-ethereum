package native

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"runtime"
)

// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

func init() {
	tracers.DefaultDirectory.Register("memoryTransactionTracer", newMemoryTransactionTracer, false)
}

// memoryTracer is a go implementation of the Tracer interface which
// performs no action. It's mostly useful for testing purposes.
type memoryTransactionTracer struct {
	heapAllocList  []int
	heapSysList    []int
	heapIdleList   []int
	heapInuseList  []int
	stackInUseList []int
	stackSysList   []int
	memStats       runtime.MemStats
}

// newmemoryTransactionTracer returns a new noop tracer.
func newMemoryTransactionTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &memoryTransactionTracer{
		heapAllocList:  []int{},
		heapSysList:    []int{},
		heapIdleList:   []int{},
		heapInuseList:  []int{},
		stackInUseList: []int{},
		stackSysList:   []int{},
	}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *memoryTransactionTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.addHeapProfile()
}

func (t *memoryTransactionTracer) addHeapProfile() {
	heapAlloc, heapSys, heapIdle, heapInuse, stackInUse, stackSys := t.getHeapAndStackMetrics()

	t.heapAllocList = append(t.heapAllocList, heapAlloc)
	t.heapSysList = append(t.heapSysList, heapSys)
	t.heapIdleList = append(t.heapIdleList, heapIdle)
	t.heapInuseList = append(t.heapInuseList, heapInuse)
	t.stackInUseList = append(t.stackInUseList, stackInUse)
	t.stackSysList = append(t.stackSysList, stackSys)
}

func (t *memoryTransactionTracer) getHeapAndStackMetrics() (int, int, int, int, int, int) {
	//runtime.GC() // get up-to-date statistics
	runtime.ReadMemStats(&t.memStats)
	return bToMb(int(t.memStats.HeapAlloc)),
		bToMb(int(t.memStats.HeapSys)),
		bToMb(int(t.memStats.HeapIdle)),
		bToMb(int(t.memStats.HeapInuse)),
		bToMb(int(t.memStats.StackInuse)),
		bToMb(int(t.memStats.StackSys))
}

// WriteToFile writes the content to the specified filename

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *memoryTransactionTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.addHeapProfile()
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *memoryTransactionTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {

}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *memoryTransactionTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *memoryTransactionTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *memoryTransactionTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (*memoryTransactionTracer) CaptureTxStart(gasLimit uint64) {}

func (*memoryTransactionTracer) CaptureTxEnd(restGas uint64) {}

// GetResult returns an empty json object.
func (t *memoryTransactionTracer) GetResult() (json.RawMessage, error) {
	// Check that all lists have the same length
	if len(t.heapAllocList) != len(t.stackInUseList) || len(t.heapAllocList) != len(t.heapSysList) ||
		len(t.heapAllocList) != len(t.heapIdleList) || len(t.heapAllocList) != len(t.heapInuseList) || len(t.heapAllocList) != len(t.stackSysList) {
		return nil, fmt.Errorf("all lists must have the same length")
	}

	// Prepare the slice to hold all pairs
	pairs := make([][]int, len(t.heapAllocList))

	// Combine each pair of heapAlloc, heapSys, heapIdle, heapInuse, stackInUse, and stackSys values
	for i := range t.heapAllocList {
		pair := []int{t.heapAllocList[i], t.heapSysList[i], t.heapIdleList[i], t.heapInuseList[i], t.stackInUseList[i], t.stackSysList[i]}
		pairs[i] = pair
	}

	// Encode the slice of slices to JSON
	jsonBytes, err := json.Marshal(pairs)
	if err != nil {
		return json.RawMessage(`{}`), err
	}

	return jsonBytes, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *memoryTransactionTracer) Stop(err error) {
}
