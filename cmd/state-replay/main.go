// State replay tool for rebuilding EVM state from migrated blocks
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/consensus"
	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/rpc"
	"github.com/luxfi/geth/trie"
	"github.com/holiman/uint256"
)

var (
	dbPath     = flag.String("db", "/home/z/.luxd/chainData/C/db/badgerdb/ethdb", "Database path")
	startBlock = flag.Uint64("start", 1, "Start block number")
	endBlock   = flag.Uint64("end", 0, "End block number (0 for all)")
	testAddr   = flag.String("addr", "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714", "Address to check balance")
	dryRun     = flag.Bool("dry", false, "Dry run without writing state")
	verbose    = flag.Bool("v", false, "Verbose output")
)

// Simple consensus engine for replay
type replayEngine struct{}

func (e *replayEngine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (e *replayEngine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

func (e *replayEngine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	for range headers {
		results <- nil
	}
	return abort, results
}

func (e *replayEngine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return nil
}

func (e *replayEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

func (e *replayEngine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body) {
}

func (e *replayEngine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil)), nil
}

func (e *replayEngine) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	results <- block
	return nil
}

func (e *replayEngine) SealHash(header *types.Header) common.Hash {
	return header.Hash()
}

func (e *replayEngine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

func (e *replayEngine) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return nil
}

func (e *replayEngine) Close() error {
	return nil
}

func main() {
	flag.Parse()

	log.Printf("Opening database at %s", *dbPath)

	// Open the database directly
	db, err := rawdb.OpenDatabase(&rawdb.DatabaseConfig{
		Directory: *dbPath,
		ReadOnly:  *dryRun,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get chain config
	chainConfig := &params.ChainConfig{
		ChainID:                 big.NewInt(96369),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		ShanghaiTime:            new(uint64),
		CancunTime:              new(uint64),
	}

	// Initialize blockchain
	genesis := &core.Genesis{
		Config:    chainConfig,
		Difficulty: big.NewInt(1),
		GasLimit:  30000000,
		Timestamp: 1640000000,
	}

	// Create blockchain without state initialization
	bcConfig := core.DefaultConfig()
	bcConfig.SnapshotLimit = 0 // Disable snapshots for replay

	blockchain, err := core.NewBlockChain(db, genesis, &replayEngine{}, bcConfig)
	if err != nil {
		log.Fatalf("Failed to create blockchain: %v", err)
	}
	defer blockchain.Stop()

	// Get current state
	stateDB, err := blockchain.State()
	if err != nil {
		log.Printf("Warning: Could not get current state: %v", err)
	}

	// Check initial balance
	if stateDB != nil && *testAddr != "" {
		addr := common.HexToAddress(*testAddr)
		balance := stateDB.GetBalance(addr)
		balanceBig := balance.ToBig()
		log.Printf("Current balance of %s: %s LUX", *testAddr, new(big.Int).Div(balanceBig, big.NewInt(1e18)).String())
	}

	// Get head block
	head := blockchain.CurrentBlock()
	log.Printf("Current head: block %d (hash: %x)", head.Number.Uint64(), head.Hash())

	// Determine range
	end := *endBlock
	if end == 0 || end > head.Number.Uint64() {
		end = head.Number.Uint64()
	}

	log.Printf("Replaying blocks from %d to %d", *startBlock, end)

	// Create processor
	processor := core.NewStateProcessor(chainConfig, blockchain.HeaderChain())

	// Process blocks
	for num := *startBlock; num <= end; num++ {
		block := blockchain.GetBlockByNumber(num)
		if block == nil {
			log.Printf("Block %d: not found, skipping", num)
			continue
		}

		// Get parent state
		var parentState *state.StateDB
		if num > 0 {
			parent := blockchain.GetBlockByNumber(num - 1)
			if parent != nil {
				parentState, err = blockchain.StateAt(parent.Root())
				if err != nil {
					log.Printf("Block %d: Could not get parent state: %v", num, err)
					continue
				}
			}
		} else {
			parentState, err = blockchain.StateAt(block.Root())
			if err != nil {
				log.Printf("Block %d: Could not get genesis state: %v", num, err)
				continue
			}
		}

		if parentState == nil {
			log.Printf("Block %d: No parent state available, skipping", num)
			continue
		}

		// Process the block
		header := block.Header()
		receipts, _, usedGas, err := processor.Process(block, parentState, vm.Config{})
		if err != nil {
			log.Printf("Block %d: Failed to process: %v", num, err)
			continue
		}

		// Finalize the block
		processor.Finalize(blockchain, header, parentState, block.Body())

		// Commit state
		newRoot, err := parentState.Commit(num, true)
		if err != nil {
			log.Printf("Block %d: Failed to commit state: %v", num, err)
			continue
		}

		if *verbose || num%100 == 0 {
			log.Printf("Block %d: Processed %d txs, gas used: %d, new root: %x",
				num, len(block.Transactions()), usedGas, newRoot)

			// Check test balance periodically
			if *testAddr != "" && num%1000 == 0 {
				testState, err := blockchain.StateAt(newRoot)
				if err == nil {
					addr := common.HexToAddress(*testAddr)
					balance := testState.GetBalance(addr)
					balanceBig := balance.ToBig()
					log.Printf("  Balance at block %d: %s LUX", num, new(big.Int).Div(balanceBig, big.NewInt(1e18)).String())
				}
			}
		}

		// Update blockchain head if not dry run
		if !*dryRun {
			rawdb.WriteHeadBlockHash(db, block.Hash())
			rawdb.WriteHeadHeaderHash(db, block.Hash())
		}
	}

	// Final state check
	finalState, err := blockchain.State()
	if err == nil && *testAddr != "" {
		addr := common.HexToAddress(*testAddr)
		balance := finalState.GetBalance(addr)
		balanceBig := balance.ToBig()
		log.Printf("\nFinal balance of %s: %s LUX", *testAddr, new(big.Int).Div(balanceBig, big.NewInt(1e18)).String())
	}

	log.Printf("\nReplay complete!")
}