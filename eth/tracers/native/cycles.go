//go:build linux
// +build linux

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
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Olaburns/perf-utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"strconv"
)

func init() {
	tracers.DefaultDirectory.Register("cycleTracer", newCycleTracer, false)
}

type cycleTracer struct {
	opcodes      []vm.OpCode
	cycles       []int
	cost         []int
	cb           func()
	fd           int
	remainingGas int
	opcodeCosts  *OpcodeCosts
}

// newTimingTracer returns a new noop tracer.
func newCycleTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	t := &cycleTracer{
		opcodes:      []vm.OpCode{},
		cycles:       []int{},
		cost:         []int{},
		remainingGas: 0,
		opcodeCosts:  NewOpcodeCosts(),
	}

	return t, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *cycleTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.startMeasuring()
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *cycleTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {

}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *cycleTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	pv, err2 := perf.StopCPUCycles(t.cb, t.fd)

	if err2 != nil {
		fmt.Println("StopCPUCycles failed:", err2)
	}

	cycels := int(pv.Value)
	if t.remainingGas == 0 {
		t.remainingGas = int(gas)
	} else {
		gasCost := t.remainingGas - int(gas)
		t.cost = append(t.cost, gasCost)
		t.remainingGas = int(gas)
	}

	t.cycles = append(t.cycles, int(cycels))
	t.opcodes = append(t.opcodes, op)
	t.startMeasuring()
}

func (t *cycleTracer) startMeasuring() {
	cb, fd, err := perf.StartCPUCycles()
	if err != nil {
		fmt.Println("StopCPUCycles failed:", err)
	}
	t.cb = cb
	t.fd = fd
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
	t.cost = append(t.cost, t.remainingGas-int(restGas))
	perf.StopCPUCycles(t.cb, t.fd)
}

// GetResult returns an empty json object.
func (t *cycleTracer) GetResult() (json.RawMessage, error) {
	csvData, err := CyclesToCSV(t.opcodes, t.cycles, t.cost)

	// Encode the slice of slices to JSON
	jsonBytes, err := json.Marshal(csvData)
	if err != nil {
		fmt.Println(err)
		return json.RawMessage(`{}`), err
	}

	return jsonBytes, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *cycleTracer) Stop(err error) {
}

func CyclesToCSV(opcodes []vm.OpCode, cycles, cost []int) (string, error) {
	// Check if all slices have the same length
	if len(opcodes) != len(cycles) || len(cycles) != len(cost) {
		return "", errors.New("all slices must have the same length")
	}

	// Create a buffer to hold the CSV data
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)

	// Write the headers to the CSV
	err := w.Write([]string{"opcodes", "cycles", "cost"})
	if err != nil {
		return "", err
	}

	// Write data to CSV
	for i := 0; i < len(opcodes); i++ {
		row := []string{
			opcodes[i].String(),
			strconv.Itoa(cycles[i]),
			strconv.Itoa(cost[i]),
		}
		err = w.Write(row)
		if err != nil {
			return "", err
		}
	}

	// Flush any remaining data to the writer
	w.Flush()

	// Check for any errors during write
	err = w.Error()
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
