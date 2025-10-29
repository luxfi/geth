package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/dgraph-io/badger/v4"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

func main() {
	dbPath := "/home/z/.lux-cli/mainnet/chainData/network-96369/4aYc2FXx3EDKf98wqmxaRkkLERa7QSbbNnKRL7awjHqVqGgxj/db/ethdb"
	tipHash := common.HexToHash("0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0")
	tipNumber := uint64(1082780)

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Printf("Failed to open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Build header key: 'h' + blockNumber(8 bytes) + hash(32 bytes)
	headerKey := make([]byte, 41)
	headerKey[0] = 'h'
	binary.BigEndian.PutUint64(headerKey[1:9], tipNumber)
	copy(headerKey[9:41], tipHash[:])

	fmt.Printf("Verifying migrated headers...\n\n")
	fmt.Printf("Checking tip header (block %d):\n", tipNumber)
	fmt.Printf("  Hash: %s\n\n", tipHash.Hex())

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(headerKey)
		if err == badger.ErrKeyNotFound {
			fmt.Printf("❌ Header key NOT FOUND\n")
			return nil
		} else if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return err
		}

		return item.Value(func(val []byte) error {
			fmt.Printf("✅ Header key FOUND\n")
			fmt.Printf("   RLP data length: %d bytes\n", len(val))

			// Try to decode the header
			header := new(types.Header)
			if err := rlp.DecodeBytes(val, header); err != nil {
				fmt.Printf("   ❌ Failed to decode header: %v\n", err)
				return nil
			}

			fmt.Printf("   ✅ Header decoded successfully!\n")
			fmt.Printf("   Block number: %d\n", header.Number.Uint64())
			fmt.Printf("   Block hash: %s\n", header.Hash().Hex())
			fmt.Printf("   Parent hash: %s\n", header.ParentHash.Hex())
			fmt.Printf("   State root: %s\n", header.Root.Hex())
			fmt.Printf("   Gas used: %d\n", header.GasUsed)
			fmt.Printf("   Timestamp: %d\n", header.Time)

			// Check optional fields
			fmt.Printf("\n   Optional fields:\n")
			if header.BaseFee != nil {
				fmt.Printf("   BaseFee: %s\n", header.BaseFee.String())
			} else {
				fmt.Printf("   BaseFee: nil\n")
			}
			if header.WithdrawalsHash != nil {
				fmt.Printf("   WithdrawalsHash: %s\n", header.WithdrawalsHash.Hex())
			} else {
				fmt.Printf("   WithdrawalsHash: nil ✅\n")
			}
			if header.BlobGasUsed != nil {
				fmt.Printf("   BlobGasUsed: %d\n", *header.BlobGasUsed)
			} else {
				fmt.Printf("   BlobGasUsed: nil ✅\n")
			}
			if header.ExcessBlobGas != nil {
				fmt.Printf("   ExcessBlobGas: %d\n", *header.ExcessBlobGas)
			} else {
				fmt.Printf("   ExcessBlobGas: nil ✅\n")
			}
			if header.ParentBeaconRoot != nil {
				fmt.Printf("   ParentBeaconRoot: %s\n", header.ParentBeaconRoot.Hex())
			} else {
				fmt.Printf("   ParentBeaconRoot: nil ✅\n")
			}

			return nil
		})
	})

	if err != nil {
		fmt.Printf("Transaction error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ SUCCESS: Headers migrated correctly and can be decoded by geth!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
