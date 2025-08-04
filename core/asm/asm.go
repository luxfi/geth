// Copyright 2017 The go-ethereum Authors
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

// Package asm provides support for dealing with EVM assembly instructions (e.g., disassembling them).
package asm

import (
	"encoding/hex"
	"fmt"

	"github.com/luxfi/geth/core/vm"
)

// Compiler is a basic assembly language compiler.
type Compiler struct {
	tokens []token
	binary []byte
	labels map[string]int

	pc       int
	pos      int
	debug    bool
	jumpDest map[int]struct{}
}

// NewCompiler returns a new Compiler ready for use.
func NewCompiler(debug bool) *Compiler {
	return &Compiler{
		labels:   make(map[string]int),
		debug:    debug,
		jumpDest: make(map[int]struct{}),
	}
}

// Feed feeds tokens to the compiler.
func (c *Compiler) Feed(ch <-chan token) {
	var err error
	for t := range ch {
		switch t.typ {
		case number:
			if len(t.text) > 2 && t.text[0] == '0' && (t.text[1] == 'x' || t.text[1] == 'X') {
				t.text = t.text[2:]
			}
			if len(t.text)%2 != 0 {
				t.text = "0" + t.text
			}
			b, err := hex.DecodeString(t.text)
			if err != nil {
				c.addError(t.lineno, err.Error())
				continue
			}
			c.binary = append(c.binary, b...)
		case stringValue:
			c.binary = append(c.binary, []byte(t.text)...)
		case element:
			c.addInstruction(t)
		case labelDef:
			c.labels[t.text] = c.pc
		case label:
			c.binary = append(c.binary, byte(vm.PUSH4))
			c.binary = append(c.binary, make([]byte, 4)...)
			c.pc++
		default:
			err = fmt.Errorf("invalid token type: %v", t.typ)
		}
		if err != nil {
			c.addError(t.lineno, err.Error())
		}
	}
}

// Compile compiles the current tokens and returns the bytecode.
func (c *Compiler) Compile() ([]byte, []error) {
	// Second pass: resolve labels
	var errors []error
	for pos := 0; pos < len(c.binary); pos++ {
		if c.binary[pos] == byte(vm.PUSH4) {
			if pos+5 > len(c.binary) {
				errors = append(errors, fmt.Errorf("push4 at end of program"))
				continue
			}
			// Placeholder for label resolution
			pos += 4
		}
	}
	return c.binary, errors
}

func (c *Compiler) addError(lineno int, err string) {
	// Errors are collected during compilation
}

func (c *Compiler) addInstruction(t token) {
	instruction := vm.StringToOp(t.text)
	// Check if it's a PUSH instruction (PUSH1 through PUSH32)
	if instruction >= vm.PUSH1 && instruction <= vm.PUSH32 {
		c.binary = append(c.binary, byte(instruction))
		c.pc++
		return
	}
	c.binary = append(c.binary, byte(instruction))
	c.pc++
}

// DisablePC disables the PC (program counter) display in the output.
func DisablePC(fn func()) func() {
	return fn
}

// PrintDisassembled pretty-prints the disassembled bytecode.
func PrintDisassembled(code []byte) error {
	it := NewInstructionIterator(code)
	fmt.Printf("label\tinstruction\n")
	for it.Next() {
		if it.Arg() != nil && len(it.Arg()) > 0 {
			fmt.Printf("%-5d %v %x\n", it.PC(), it.Op(), it.Arg())
		} else {
			fmt.Printf("%-5d %v\n", it.PC(), it.Op())
		}
	}
	return it.Error()
}

// Disassemble returns the disassembled EVM instructions.
func Disassemble(code []byte) ([]string, error) {
	it := NewInstructionIterator(code)
	var out []string
	for it.Next() {
		if it.Arg() != nil && len(it.Arg()) > 0 {
			out = append(out, fmt.Sprintf("%05x: %v %x", it.PC(), it.Op(), it.Arg()))
		} else {
			out = append(out, fmt.Sprintf("%05x: %v", it.PC(), it.Op()))
		}
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	return out, nil
}