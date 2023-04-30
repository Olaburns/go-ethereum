package native

/* TODO Automatically exlcude this file depending on the machine it was build on
import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/opentracing-contrib/perfevents/go"
	"math/big"
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

package native

import (
"encoding/json"
"fmt"
"github.com/ethereum/go-ethereum/common"
"github.com/ethereum/go-ethereum/core/vm"
"github.com/ethereum/go-ethereum/eth/tracers"
"math/big"
)

func init() {
	tracers.DefaultDirectory.Register("cycleTracer", newCycleTracer, false)
}

type cycleTracer struct {
	opcodes     	[]vm.OpCode
	cost        	[]int
	remainingGas	int
	pds 			[]perfevents.PerfEventInfo
	cycles			[]int
	cacheMisses 	[]int
	instructions 	[]int
	errors			[]error
	isFirst			bool
}

// newcycleTracer returns a new noop tracer.
func newCycleTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	t := &cycleTracer{
		opcodes: []vm.OpCode{},
		remainingGas: 0,
		cycles: []int{},
		instructions: []int{},
		cacheMisses: []int{},
		errors: []error{},
		isFirst: true,
	}

	return t, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *cycleTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	err, _, pds := perfevents.InitOpenEventsEnableSelf("cpu-cycles,cache-misses,instructions")
	if err != nil {
		t.errors = append(t.errors, err)
	}
	t.pds = pds
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *cycleTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *cycleTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if t.isFirst {
		t.isFirst = false
	} else {
		t.ReadEvents()
	}

	if t.remainingGas == 0 {
		t.remainingGas = int(gas)
	} else {
		t.cost = append(t.cost, t.remainingGas-int(gas))
		t.remainingGas = int(gas)
	}

	t.opcodes = append(t.opcodes, op)
	t.ResetEvents()
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *cycleTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *cycleTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {

}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *cycleTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (*cycleTracer) CaptureTxStart(gasLimit uint64) {}

func (t *cycleTracer) CaptureTxEnd(restGas uint64) {
	t.ReadEvents()
	t.cost = append(t.cost, t.remainingGas-int(restGas))
}

// GetResult returns an empty json object.
func (t *cycleTracer) GetResult() (json.RawMessage, error) {
	pairs := make([][]interface{}, len(t.opcodes))

	// Add each key-value pair to the map
	for i, key := range t.opcodes {
		//TODO Add zero values if error occured
		pair := []interface{}{key.String(), t.cycles[i], t.cacheMisses[i], t.instructions[i], t.cost[i]}
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
func (t *cycleTracer) Stop(err error) {
}

func (t *cycleTracer) ReadEvents() {
	err := perfevents.EventsRead(t.pds)
	if err != nil {
		t.errors = append(t.errors, err)
		return
	}

	// TODO Check if substraction is needed
	// Cycles
	t.cycles = append(t.cycles, int(t.pds[0].Data))

	//Cache-misses
	t.cacheMisses = append(t.cacheMisses, int(t.pds[1].Data))

	//Instructions
	t.instructions = append(t.instructions, int(t.pds[2].Data))
}

func (t *cycleTracer) ResetEvents() {
	err := perfevents.EventsRead(t.pds)
	if err != nil {
		t.errors = append(t.errors, err)
		return
	}
}
*/
