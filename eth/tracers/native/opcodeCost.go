package native

import "github.com/ethereum/go-ethereum/core/vm"

// OpcodeCosts keeps track of the cost of opcodes
type OpcodeCosts struct {
	costs map[vm.OpCode]int
}

// NewOpcodeCosts creates a new OpcodeCosts structure
func NewOpcodeCosts() *OpcodeCosts {
	return &OpcodeCosts{costs: make(map[vm.OpCode]int)}
}

// AddOrUpdateOpcode adds a new opcode and its cost, or updates the cost if the opcode already exists
func (oc *OpcodeCosts) AddOpcode(opcode vm.OpCode, cost int) {
	// If the opcode exists in the map, the cost is ignored
	if _, exists := oc.costs[opcode]; exists {
		return
	}

	// Otherwise, add the opcode and its cost to the map
	oc.costs[opcode] = cost
}

// GetCost gets the cost of a specific opcode
func (oc *OpcodeCosts) GetCost(opcode vm.OpCode) (int, bool) {
	cost, exists := oc.costs[opcode]
	return cost, exists
}

func (oc *OpcodeCosts) AddAndGetCost(opcode vm.OpCode, cost int) (int, bool) {
	oc.AddOpcode(opcode, cost)
	return oc.GetCost(opcode)
}
