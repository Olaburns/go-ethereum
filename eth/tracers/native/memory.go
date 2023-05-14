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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"runtime"
	"strconv"
)

func init() {
	tracers.DefaultDirectory.Register("memoryTracer", newMemoryTracer, false)
}

// memoryTracer is a go implementation of the Tracer interface which
// performs no action. It's mostly useful for testing purposes.
type memoryTracer struct {
	opCounter   int
	resolution  int
	csvFileName string
}

// newmemoryTracer returns a new noop tracer.
func newMemoryTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &memoryTracer{
		opCounter:   0,
		resolution:  100,
		csvFileName: "memoryStats.csv",
	}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *memoryTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	err := createCSV(t.csvFileName)
	if err != nil {
		log.Fatalf("Failed to create CSV: %v", err)
	}
}

func createCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"heapAlloc", "heapSys", "heapIdle", "heapInuse", "stackInUse", "stackSys"}
	err = writer.Write(headers) // writing header
	if err != nil {
		return err
	}

	return nil
}

func addMemStatsToCSV(filename string) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	stats := []string{
		strconv.Itoa(bToMb(int(mem.HeapAlloc))),
		strconv.Itoa(bToMb(int(mem.HeapSys))),
		strconv.Itoa(bToMb(int(mem.HeapIdle))),
		strconv.Itoa(bToMb(int(mem.HeapInuse))),
		strconv.Itoa(bToMb(int(mem.StackInuse))),
		strconv.Itoa(bToMb(int(mem.StackSys))),
	}
	err = writer.Write(stats) // writing stats
	if err != nil {
		return err
	}

	return nil
}

func getCSVAsStringAndDelete(filename string) (string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	err = os.Remove(filename)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func bToMb(b int) int {
	return b / 1024 / 1024
}

// WriteToFile writes the content to the specified filename
func WriteToFile(filename, content string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %w", err)
	}

	// Combine cwd and filename to get the full path
	fullPath := fmt.Sprintf("%s/%s", cwd, filename)

	// Write the content to the file
	err = ioutil.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *memoryTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	err = addMemStatsToCSV(t.csvFileName)
	if err != nil {
		log.Fatalf("Failed to add memory stats to CSV: %v", err)
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *memoryTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if 0 == t.opCounter%t.resolution {
		err := addMemStatsToCSV(t.csvFileName)
		if err != nil {
			log.Fatalf("Failed to add memory stats to CSV: %v", err)
		}
	}
	t.opCounter = t.opCounter + 1
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *memoryTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *memoryTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *memoryTracer) CaptureExit(output []byte, gasUsed uint64, err error) {

}

func (*memoryTracer) CaptureTxStart(gasLimit uint64) {}

func (*memoryTracer) CaptureTxEnd(restGas uint64) {

}

// GetResult returns an empty json object.
func (t *memoryTracer) GetResult() (json.RawMessage, error) {
	csvString, err := getCSVAsStringAndDelete(t.csvFileName)

	// Encode the slice of slices to JSON
	jsonBytes, err := json.Marshal(csvString)
	if err != nil {
		return json.RawMessage(`{}`), err
	}

	return jsonBytes, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *memoryTracer) Stop(err error) {
}
