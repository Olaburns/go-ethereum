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

import "C"
import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/square/inspect/metrics"
	"github.com/square/inspect/os/pidstat"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	tracers.DefaultDirectory.Register("storageTracer", newStorageTracer, false)
}

// storageTracer is a go implementation of the Tracer interface which
// performs no action. It's mostly useful for testing purposes.
type storageTracer struct {
	pMetrics     *pidstat.PerProcessStatMetrics
	IOReadBytes  []float64
	IOWriteBytes []float64
	IOUsage      []float64
}

// newstorageTracer returns a new noop tracer.
func newStorageTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &storageTracer{
		IOReadBytes:  []float64{},
		IOWriteBytes: []float64{},
		IOUsage:      []float64{},
	}, nil
}

func (t *storageTracer) createProcessStats() {
	m := metrics.NewMetricContext("system")
	pstat := pidstat.NewProcessStat(m, time.Millisecond*50)
	pstat.Collect()
	pid := strconv.Itoa(os.Getpid())
	WriteToFile("pid.txt", pid)
	WriteToFile("pid_list.txt", joinMapValues(pstat.Processes))
	pMetrics := pstat.Processes[pid].Metrics

	WriteToFile("log.txt", "I am here")

	t.pMetrics = pMetrics
}

func (t *storageTracer) readProcessStats() {
	WriteToFile("log.txt", t.pMetrics.Pid)
	t.pMetrics.Collect()
	o := t.pMetrics
	t.IOReadBytes = append(t.IOReadBytes, o.IOReadBytes.ComputeRate())
	t.IOWriteBytes = append(t.IOWriteBytes, o.IOWriteBytes.ComputeRate())
	t.IOUsage = append(t.IOUsage, o.IOReadBytes.ComputeRate()+o.IOWriteBytes.ComputeRate())
}

func joinMapValues(m map[string]*pidstat.PerProcessStat) string {
	var sb strings.Builder

	for _, value := range m {
		// Each value starts in a new line
		sb.WriteString("\n")
		sb.WriteString(value.Metrics.Pid)
	}

	return sb.String()
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *storageTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.createProcessStats()

}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *storageTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.readProcessStats()
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *storageTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	t.readProcessStats()
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *storageTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *storageTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *storageTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (*storageTracer) CaptureTxStart(gasLimit uint64) {}

func (*storageTracer) CaptureTxEnd(restGas uint64) {}

// GetResult returns an empty json object.
func (t *storageTracer) GetResult() (json.RawMessage, error) {
	csvString, err := ArraysToCSV(t.IOReadBytes, t.IOWriteBytes, t.IOUsage)

	// Encode the slice of slices to JSON
	jsonBytes, err := json.Marshal(csvString)
	if err != nil {
		fmt.Println(err)
		return json.RawMessage(`{}`), err
	}

	return jsonBytes, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *storageTracer) Stop(err error) {
}

func ArraysToCSV(ioReadBytes []float64, ioWriteBytes []float64, ioUsage []float64) (string, error) {
	// Checking if all slices have the same length
	if len(ioReadBytes) != len(ioWriteBytes) || len(ioWriteBytes) != len(ioUsage) {
		return "", fmt.Errorf("all input slices should have the same length")
	}

	b := &bytes.Buffer{}
	writer := csv.NewWriter(b)

	// Write the headers to the csv
	err := writer.Write([]string{"IOReadBytes", "IOWriteBytes", "IOUsage"})
	if err != nil {
		return "", err
	}

	// Loop through the slices and write the data to the csv
	for i := range ioReadBytes {
		row := []string{
			strconv.FormatFloat(ioReadBytes[i], 'f', -1, 64),
			strconv.FormatFloat(ioWriteBytes[i], 'f', -1, 64),
			strconv.FormatFloat(ioUsage[i], 'f', -1, 64),
		}
		err := writer.Write(row)
		if err != nil {
			return "", err
		}
	}
	writer.Flush()

	// Check for any error occurred while writing
	if err := writer.Error(); err != nil {
		return "", err
	}

	return b.String(), nil
}
