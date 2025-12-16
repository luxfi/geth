// +build ignore

package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

func main() {
	data, err := os.ReadFile("/tmp/blocks.rlp")
	if err != nil {
		panic(err)
	}

	reader := bytes.NewReader(data)
	stream := rlp.NewStream(reader, 0)

	var prevHash string
	for i := 0; i < 5; i++ {
		// Use DecodeBlock which sets rawRLP
		var block types.Block
		if err := stream.Decode(&block); err != nil {
			panic(err)
		}
		
		header := block.Header()
		hash := header.Hash()
		parent := header.ParentHash
		
		fmt.Printf("Block %d:\n", header.Number.Uint64())
		fmt.Printf("  Hash:   %s\n", hash.Hex())
		fmt.Printf("  Parent: %s\n", parent.Hex())
		fmt.Printf("  rawRLP set: %v (%d bytes)\n", len(header.RawRLP()) > 0, len(header.RawRLP()))
		
		if i > 0 && parent.Hex() != prevHash {
			fmt.Printf("  *** CHAIN BREAK! Expected parent %s\n", prevHash)
		}
		
		prevHash = hash.Hex()
		fmt.Println()
	}
}
