// Copyright 2025 The go-ethereum Authors
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

package trie

// StackTrieOptions represents options for stack trie
type StackTrieOptions struct {
	writer OnTrieNode
}

// NewStackTrieOptions creates new stack trie options
func NewStackTrieOptions() OnTrieNode {
	return nil
}

// WithWriter sets the writer for the stack trie options
func (o OnTrieNode) WithWriter(writer OnTrieNode) OnTrieNode {
	return writer
}