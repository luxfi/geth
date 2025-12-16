package main

import (
	"encoding/hex"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/rlp"
	"golang.org/x/crypto/sha3"
)

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func main() {
	// SubnetEVM namespace
	namespace, _ := hex.DecodeString("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")

	// Block 1's state root
	block1StateRoot, _ := hex.DecodeString("a50232e29bc46561e0a2eb86f37ec3daf0477438370446fdab9fa822324f6063")

	// Open SubnetEVM database
	srcDB, err := pebble.Open("/Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb", &pebble.Options{ReadOnly: true})
	if err != nil {
		fmt.Printf("Error opening SubnetEVM db: %v\n", err)
		return
	}
	defer srcDB.Close()

	// Get root node for block 1 state
	srcKey := append(namespace, block1StateRoot...)
	rootNode, closer, err := srcDB.Get(srcKey)
	if err != nil {
		fmt.Printf("Error getting block 1 state root: %v\n", err)
		return
	}
	rootNodeCopy := make([]byte, len(rootNode))
	copy(rootNodeCopy, rootNode)
	closer.Close()

	fmt.Printf("=== Block 1 State Root ===\n")
	fmt.Printf("Root hash: %x\n", block1StateRoot)
	fmt.Printf("Root node (%d bytes): %x\n\n", len(rootNodeCopy), rootNodeCopy)

	// Decode the root node
	var decoded interface{}
	err = rlp.DecodeBytes(rootNodeCopy, &decoded)
	if err != nil {
		fmt.Printf("Error decoding root: %v\n", err)
		return
	}

	// Print structure
	printNode(srcDB, namespace, decoded, "", 0)

	// Also check genesis state root for comparison
	genesisStateRoot, _ := hex.DecodeString("2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80")
	genesisKey := append(namespace, genesisStateRoot...)
	genesisNode, closer2, err := srcDB.Get(genesisKey)
	if err != nil {
		fmt.Printf("\nError getting genesis state root: %v\n", err)
		return
	}
	genesisNodeCopy := make([]byte, len(genesisNode))
	copy(genesisNodeCopy, genesisNode)
	closer2.Close()

	fmt.Printf("\n=== Genesis State Root ===\n")
	fmt.Printf("Root hash: %x\n", genesisStateRoot)
	fmt.Printf("Root node (%d bytes): %x\n\n", len(genesisNodeCopy), genesisNodeCopy)

	var genesisDecoded interface{}
	rlp.DecodeBytes(genesisNodeCopy, &genesisDecoded)
	printNode(srcDB, namespace, genesisDecoded, "", 0)
}

func printNode(db *pebble.DB, namespace []byte, node interface{}, indent string, depth int) {
	if depth > 5 {
		fmt.Printf("%s[max depth reached]\n", indent)
		return
	}

	switch n := node.(type) {
	case []interface{}:
		if len(n) == 17 {
			// Branch node
			fmt.Printf("%sBranch node:\n", indent)
			for i, child := range n {
				if child == nil || (isBytes(child) && len(child.([]byte)) == 0) {
					continue
				}
				if i < 16 {
					fmt.Printf("%s  [%x]: ", indent, i)
					if bytes, ok := child.([]byte); ok && len(bytes) == 32 {
						// Hash reference - fetch and decode
						fmt.Printf("hash ref %x\n", bytes)
						childKey := append(namespace, bytes...)
						childNode, closer, err := db.Get(childKey)
						if err == nil {
							childCopy := make([]byte, len(childNode))
							copy(childCopy, childNode)
							closer.Close()
							var childDecoded interface{}
							rlp.DecodeBytes(childCopy, &childDecoded)
							printNode(db, namespace, childDecoded, indent+"    ", depth+1)
						}
					} else {
						// Inline node
						printNode(db, namespace, child, indent+"    ", depth+1)
					}
				} else {
					// Value slot
					if bytes, ok := child.([]byte); ok && len(bytes) > 0 {
						fmt.Printf("%s  [value]: %x\n", indent, bytes)
					}
				}
			}
		} else if len(n) == 2 {
			// Extension or Leaf node
			if pathBytes, ok := n[0].([]byte); ok {
				if len(pathBytes) > 0 {
					prefix := pathBytes[0] >> 4
					isLeaf := (prefix & 0x02) != 0

					// Extract path nibbles
					var nibbles string
					if prefix&0x01 != 0 {
						// Odd length - first nibble is data
						nibbles = fmt.Sprintf("%x", pathBytes[0]&0x0f)
						for _, b := range pathBytes[1:] {
							nibbles += fmt.Sprintf("%x%x", b>>4, b&0x0f)
						}
					} else {
						// Even length - skip first byte
						for _, b := range pathBytes[1:] {
							nibbles += fmt.Sprintf("%x%x", b>>4, b&0x0f)
						}
					}

					if isLeaf {
						fmt.Printf("%sLeaf node (path=%s):\n", indent, nibbles)
						if valueBytes, ok := n[1].([]byte); ok {
							// Decode account
							var account struct {
								Nonce    uint64
								Balance  []byte
								Root     []byte
								CodeHash []byte
							}
							if err := rlp.DecodeBytes(valueBytes, &account); err == nil {
								fmt.Printf("%s  Nonce: %d\n", indent, account.Nonce)
								fmt.Printf("%s  Balance: %x\n", indent, account.Balance)
								fmt.Printf("%s  Root: %x\n", indent, account.Root)
								fmt.Printf("%s  CodeHash: %x\n", indent, account.CodeHash)
							} else {
								fmt.Printf("%s  Value: %x\n", indent, valueBytes)
							}
						}
					} else {
						fmt.Printf("%sExtension node (path=%s):\n", indent, nibbles)
						if valueBytes, ok := n[1].([]byte); ok && len(valueBytes) == 32 {
							fmt.Printf("%s  -> hash: %x\n", indent, valueBytes)
							childKey := append(namespace, valueBytes...)
							childNode, closer, err := db.Get(childKey)
							if err == nil {
								childCopy := make([]byte, len(childNode))
								copy(childCopy, childNode)
								closer.Close()
								var childDecoded interface{}
								rlp.DecodeBytes(childCopy, &childDecoded)
								printNode(db, namespace, childDecoded, indent+"    ", depth+1)
							}
						} else {
							printNode(db, namespace, n[1], indent+"    ", depth+1)
						}
					}
				}
			}
		} else {
			fmt.Printf("%sUnknown node structure (len=%d)\n", indent, len(n))
		}
	case []byte:
		fmt.Printf("%sBytes: %x\n", indent, n)
	default:
		fmt.Printf("%sUnknown type: %T\n", indent, node)
	}
}

func isBytes(v interface{}) bool {
	_, ok := v.([]byte)
	return ok
}
