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
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"os"
	"strconv"
	"strings"
)

func init() {
	tracers.DefaultDirectory.Register("storageTracer", newStorageTracer, false)
}

// storageTracer is a go implementation of the Tracer interface which
// performs no action. It's mostly useful for testing purposes.
type storageTracer struct {
	PIOMetrics []*ProcIO
	resolution int
	opCounter  int
}

// newstorageTracer returns a new noop tracer.
func newStorageTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &storageTracer{
		PIOMetrics: []*ProcIO{},
		resolution: 100,
		opCounter:  0,
	}, nil
}

type ProcIO struct {
	Rchar               int64
	Wchar               int64
	Syscr               int64
	Syscw               int64
	ReadBytes           int64
	WriteBytes          int64
	CancelledWriteBytes int64
}

func (t *storageTracer) readProcessStats() {
	pid := os.Getpid()
	pidStr := strconv.Itoa(pid)
	pMetrics, err := ReadProcIO(pidStr)
	if err != nil {
		fmt.Errorf("Can not read metrics %v", err)
	}
	t.PIOMetrics = append(t.PIOMetrics, pMetrics)
}

func ReadProcIO(pid string) (*ProcIO, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%s/io", pid))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	result := &ProcIO{}
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			continue
		}

		value, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		switch parts[0] {
		case "rchar":
			result.Rchar = value
		case "wchar":
			result.Wchar = value
		case "syscr":
			result.Syscr = value
		case "syscw":
			result.Syscw = value
		case "read_bytes":
			result.ReadBytes = value
		case "write_bytes":
			result.WriteBytes = value
		case "cancelled_write_bytes":
			result.CancelledWriteBytes = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *storageTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.readProcessStats()
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *storageTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.readProcessStats()
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *storageTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if 0 == t.opCounter%t.resolution {
		t.readProcessStats()
	}
	t.opCounter = t.opCounter + 1
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
	csvString, err := procIOToCSV(t.PIOMetrics)

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

func procIOToCSV(procIOs []*ProcIO) (string, error) {
	// Create a buffer to write our output to
	b := &bytes.Buffer{}

	// Create a CSV writer that writes to our buffer
	writer := csv.NewWriter(b)

	// Write the header to the CSV file
	if err := writer.Write([]string{"Rchar", "Wchar", "Syscr", "Syscw", "ReadBytes", "WriteBytes"}); err != nil {
		return "", err
	}

	// Iterate through the input and write each ProcIO's data to the CSV writer
	for _, procIO := range procIOs {
		record := []string{
			strconv.FormatInt(procIO.Rchar, 10),
			strconv.FormatInt(procIO.Wchar, 10),
			strconv.FormatInt(procIO.Syscr, 10),
			strconv.FormatInt(procIO.Syscw, 10),
			strconv.FormatInt(procIO.ReadBytes, 10),
			strconv.FormatInt(procIO.WriteBytes, 10),
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	// Flush any remaining data from the writer to the buffer
	writer.Flush()

	// Check for any error that occurred during the write
	if err := writer.Error(); err != nil {
		return "", err
	}

	// Convert the buffer's contents to a string and return it
	return b.String(), nil
}
