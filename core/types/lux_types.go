package types

import (
	"math/big"
	
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/crypto"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// Body represents a block body with Lux extensions
type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
	Version      uint32
	ExtData      []byte
}

// NewBlockWithExtData creates a new block with extended data
// This is a Lux-specific function that maintains compatibility
func NewBlockWithExtData(header *Header, txs []*Transaction, uncles []*Header, 
	receipts []*Receipt, version uint32, extData []byte, commit bool) *Block {
	// For now, just create a standard block
	// The extended data would be handled separately in the actual implementation
	// Create ethereum body and use standard method
	ethBody := ethtypes.Body{
		Transactions: txs,
		Uncles:       uncles,
	}
	return NewBlockWithHeader(header).WithBody(ethBody)
}

// BlockWithExtData adds extended data to a block
// This is a helper function since we can't add methods to the aliased Block type
func BlockWithExtData(b *Block, version uint32, extData []byte) *Block {
	// In the actual implementation, this would create a new block with extended data
	// For now, just return the block unchanged
	return b
}

// ExtendedStateAccount is a Lux-specific extension of StateAccount
type ExtendedStateAccount struct {
	Nonce       uint64
	Balance     *uint256.Int
	Root        common.Hash
	CodeHash    []byte
	IsMultiCoin bool
}

// Copy creates a deep copy of StateAccount.
func (s ExtendedStateAccount) Copy() *ExtendedStateAccount {
	return &ExtendedStateAccount{
		Nonce:       s.Nonce,
		Balance:     new(uint256.Int).Set(s.Balance),
		Root:        s.Root,
		CodeHash:    common.CopyBytes(s.CodeHash),
		IsMultiCoin: s.IsMultiCoin,
	}
}

// StateAccount is our extended state account type
type StateAccount = ExtendedStateAccount

// ExtendedSlimAccount is a Lux-specific extension of SlimAccount  
type ExtendedSlimAccount struct {
	Nonce       uint64
	Balance     *uint256.Int
	Root        []byte
	CodeHash    []byte
	IsMultiCoin bool
}

// SlimAccount is our extended slim account type
type SlimAccount = ExtendedSlimAccount

// NewEmptyStateAccount creates an empty state account
func NewEmptyStateAccount() *StateAccount {
	return &ExtendedStateAccount{
		Balance:  new(uint256.Int),
		Root:     ethtypes.EmptyRootHash,
		CodeHash: ethtypes.EmptyCodeHash[:],
	}
}


// SlimAccountRLP converts an account to its RLP representation for snapshot storage
func SlimAccountRLP(acc StateAccount) []byte {
	// Convert our extended account to ethereum StateAccount
	ethAcc := ethtypes.StateAccount{
		Nonce:    acc.Nonce,
		Balance:  acc.Balance,
		Root:     acc.Root,
		CodeHash: acc.CodeHash,
	}
	return ethtypes.SlimAccountRLP(ethAcc)
}

// FlattenLogs flattens a slice of logs
func FlattenLogs(logs [][]*Log) []*Log {
	var flattened []*Log
	for _, subLogs := range logs {
		flattened = append(flattened, subLogs...)
	}
	return flattened
}

// BlockGasCost retrieves the gas cost of a block
func BlockGasCost(b *Block) *big.Int {
	// For now, return nil as we need to implement extended header support
	// This would typically access the extended header data
	return nil
}

// CreateBloom creates a bloom filter from receipts
func CreateBloom(receipts Receipts) Bloom {
	var bloom Bloom
	for _, r := range receipts {
		// Create bloom for each receipt and OR them together
		ethReceipt := (*ethtypes.Receipt)(r)
		receiptBloom := ethtypes.CreateBloom(ethReceipt)
		bloom = Bloom(OrBloom(ethtypes.Bloom(bloom), receiptBloom))
	}
	return bloom
}

// OrBloom combines two bloom filters
func OrBloom(a, b ethtypes.Bloom) ethtypes.Bloom {
	var result ethtypes.Bloom
	for i := 0; i < len(a); i++ {
		result[i] = a[i] | b[i]
	}
	return result
}

// MergeBloom merges multiple blooms into one
func MergeBloom(receipts Receipts) Bloom {
	return CreateBloom(receipts)
}

// BlockTimestamp returns the timestamp of a block
func BlockTimestamp(b *Block) uint64 {
	if b == nil || b.Header() == nil {
		return 0
	}
	return b.Header().Time
}

// BlockConfigContext wraps a block to implement ConfigurationBlockContext
type BlockConfigContext struct {
	*Block
}

// Timestamp implements ConfigurationBlockContext
func (b BlockConfigContext) Timestamp() uint64 {
	return BlockTimestamp(b.Block)
}

// NewBlockConfigContext creates a new BlockConfigContext
func NewBlockConfigContext(block *Block) BlockConfigContext {
	return BlockConfigContext{Block: block}
}

// CopyHeader creates a deep copy of a block header
func CopyHeader(h *Header) *Header {
	if h == nil {
		return nil
	}
	// Use the ethereum CopyHeader function
	return ethtypes.CopyHeader(h)
}

// ExtData returns the extra data field of a block (for compatibility)
func ExtData(b *Block) []byte {
	// For now, return empty data as blocks don't have extended data in the standard implementation
	return []byte{}
}

// TxDifference returns the difference between two transaction slices
func TxDifference(a, b Transactions) Transactions {
	keep := make(map[common.Hash]struct{})
	for _, tx := range b {
		keep[tx.Hash()] = struct{}{}
	}
	
	var diff Transactions
	for _, tx := range a {
		if _, ok := keep[tx.Hash()]; !ok {
			diff = append(diff, tx)
		}
	}
	return diff
}

// bytesBacked wraps types that can provide byte representations
type bytesBacked interface {
	Bytes() []byte
}

// bytesWrapper wraps a byte slice to implement bytesBacked
type bytesWrapper []byte

func (b bytesWrapper) Bytes() []byte {
	return []byte(b)
}

// BloomLookup checks if a bloom filter contains a particular pattern
func BloomLookup(bin Bloom, topic interface{}) bool {
	var topicBytes []byte
	
	switch t := topic.(type) {
	case common.Hash:
		topicBytes = t.Bytes()
	case common.Address:
		topicBytes = t.Bytes()
	case []byte:
		topicBytes = t
	default:
		return false
	}
	
	if len(topicBytes) == 0 {
		return false
	}
	
	// Use the ethereum BloomLookup with our wrapper
	return ethtypes.BloomLookup(ethtypes.Bloom(bin), bytesWrapper(topicBytes))
}

// CalcExtDataHash calculates the hash of extended data
func CalcExtDataHash(extData []byte) common.Hash {
	if len(extData) == 0 {
		return common.Hash{}
	}
	return crypto.Keccak256Hash(extData)
}