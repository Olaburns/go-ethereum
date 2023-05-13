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

package native

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"time"
)

func init() {
	tracers.DefaultDirectory.Register("timingTracer", newTimingTracer, false)
}

type timingTracer struct {
	opcodes      []vm.OpCode
	timings      []int
	cost         []int
	time         time.Time
	remainingGas int
}

// newTimingTracer returns a new noop tracer.
func newTimingTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	t := &timingTracer{
		opcodes:      []vm.OpCode{},
		timings:      []int{},
		remainingGas: 0,
	}

	return t, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *timingTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.time = time.Now()
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *timingTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {

}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *timingTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	elapsedTime := time.Since(t.time)

	if t.remainingGas == 0 {
		t.remainingGas = int(gas)
	} else {
		t.cost = append(t.cost, t.remainingGas-int(gas))
		t.remainingGas = int(gas)
	}

	t.timings = append(t.timings, int(elapsedTime.Nanoseconds()))
	t.opcodes = append(t.opcodes, op)
	t.time = time.Now()
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *timingTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *timingTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {

}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *timingTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (*timingTracer) CaptureTxStart(gasLimit uint64) {}

func (t *timingTracer) CaptureTxEnd(restGas uint64) {
	t.cost = append(t.cost, t.remainingGas-int(restGas))
}

func (t *timingTracer) GetResult() (json.RawMessage, error) {
	pairs := make([][]interface{}, len(t.opcodes))

	// Add each key-value pair to the map
	for i, key := range t.opcodes {
		pair := []interface{}{key.String(), t.timings[i], t.cost[i]}
		pairs[i] = pair
	}

	// Encode the slice of slices to JSON
	jsonBytes, err := json.Marshal(pairs)
	if err != nil {
		fmt.Println(err)
		return json.RawMessage(`{}`), err
	}

	return jsonBytes, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *timingTracer) Stop(err error) {
}
