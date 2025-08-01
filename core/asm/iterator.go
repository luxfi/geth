// Copyright 2019 The go-ethereum Authors
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

package asm

import (
	"errors"
	"fmt"

	"github.com/luxfi/geth/core/vm"
)

// InstructionIterator iterates over EVM bytecode instructions.
type InstructionIterator struct {
	code    []byte
	pc      int
	arg     []byte
	op      vm.OpCode
	error   error
	stopped bool
}

// NewInstructionIterator creates a new instruction iterator.
func NewInstructionIterator(code []byte) *InstructionIterator {
	return &InstructionIterator{code: code}
}

// Next moves to the next instruction. Returns true if there is a next instruction.
func (it *InstructionIterator) Next() bool {
	if it.error != nil || it.stopped {
		return false
	}
	if it.pc >= len(it.code) {
		it.stopped = true
		return false
	}

	it.op = vm.OpCode(it.code[it.pc])
	var a int
	if it.op.IsPush() {
		a = int(it.op) - int(vm.PUSH1) + 1
	}
	if it.pc+1+a > len(it.code) {
		it.error = fmt.Errorf("incomplete push instruction at position %d", it.pc)
		return false
	}
	it.arg = it.code[it.pc+1 : it.pc+1+a]
	it.pc += 1 + a
	return true
}

// Error returns any error that occurred during iteration.
func (it *InstructionIterator) Error() error {
	return it.error
}

// PC returns the program counter of the current instruction.
func (it *InstructionIterator) PC() int {
	return it.pc - 1 - len(it.arg)
}

// Op returns the opcode of the current instruction.
func (it *InstructionIterator) Op() vm.OpCode {
	return it.op
}

// Arg returns the argument of the current instruction.
func (it *InstructionIterator) Arg() []byte {
	return it.arg
}

// HasValidJumpdest returns true if the destination is a valid jump destination.
func HasValidJumpdest(code []byte) bool {
	it := NewInstructionIterator(code)
	for it.Next() {
		if it.Op() == vm.JUMPDEST {
			return true
		}
	}
	return false
}

var errNoJUMPDEST = errors.New("no JUMPDEST found")

// IsProgramCounterWithinJumpDest returns true if the program counter is within a valid jump destination.
func IsProgramCounterWithinJumpDest(code []byte, pc int) error {
	if pc < 0 || pc >= len(code) {
		return fmt.Errorf("invalid program counter %d", pc)
	}
	it := NewInstructionIterator(code)
	for it.Next() {
		if it.PC() == pc {
			if it.Op() == vm.JUMPDEST {
				return nil
			}
			return errNoJUMPDEST
		}
	}
	return errNoJUMPDEST
}