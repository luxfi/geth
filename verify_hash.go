// +build ignore

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

func main() {
	data, err := os.ReadFile("/tmp/blocks.rlp")
	if err != nil {
		panic(err)
	}

	reader := bytes.NewReader(data)
	stream := rlp.NewStream(reader, 0)

	for i := 0; i < 5; i++ {
		raw, err := stream.Raw()
		if err != nil {
			break
		}

		var parts []rlp.RawValue
		if err := rlp.DecodeBytes(raw, &parts); err != nil {
			panic(err)
		}

		headerRLP := []byte(parts[0])
		
		// Hash the ORIGINAL header RLP (keccak256)
		hash := common.Keccak256Hash(headerRLP)
		
		// Get parent hash from first field
		var fields []rlp.RawValue
		rlp.DecodeBytes(headerRLP, &fields)
		parentHash := hex.EncodeToString(fields[0][1:]) // skip RLP prefix byte
		
		// Get block number from field 8
		numberBytes := fields[8]
		var number uint64
		if len(numberBytes) == 1 && numberBytes[0] < 0x80 {
			number = uint64(numberBytes[0])
		} else {
			// RLP string
			for _, b := range numberBytes[1:] {
				number = number*256 + uint64(b)
			}
		}
		
		fmt.Printf("Block %d:\n", number)
		fmt.Printf("  Original RLP hash: %s\n", hash.Hex())
		fmt.Printf("  Parent hash:       0x%s\n", parentHash)
		fmt.Println()
	}
}
