// Simple state replay tool for migrated blocks
package main

import (
	"flag"
	"log"
	"math/big"
	"path/filepath"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/triedb"
	"github.com/holiman/uint256"
)

var (
	dbPath     = flag.String("db", "/home/z/.luxd/chainData/C/db", "Database path")
	startBlock = flag.Uint64("start", 1, "Start block number")
	endBlock   = flag.Uint64("end", 100, "End block number")
	testAddr   = flag.String("addr", "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714", "Test address")
	verbose    = flag.Bool("v", false, "Verbose logging")
)

func openDB(path string) (ethdb.Database, error) {
	// Try opening as LevelDB
	return rawdb.NewLevelDBDatabase(path, 256, 1024, "", false)
}

func main() {
	flag.Parse()

	// Open database
	log.Printf("Opening database at %s", *dbPath)
	db, err := openDB(*dbPath)
	if err != nil {
		// Try badgerdb path
		badgerPath := filepath.Join(*dbPath, "badgerdb", "ethdb")
		log.Printf("LevelDB failed, trying BadgerDB at %s", badgerPath)
		db, err = openDB(badgerPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
	}
	defer db.Close()

	// Chain config
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(96369),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
	}

	// Check if we have blocks
	headHash := rawdb.ReadHeadBlockHash(db)
	if headHash == (common.Hash{}) {
		log.Fatal("No head block found in database")
	}

	headNumber := rawdb.ReadHeaderNumber(db, headHash)
	if headNumber == nil {
		log.Fatal("Head block number not found")
	}

	log.Printf("Head block: %d (hash: %x)", *headNumber, headHash)

	// Initialize state database
	trieDB := triedb.NewDatabase(db, &triedb.Config{
		Preimages: true,
	})
	stateDB := state.NewDatabase(trieDB, nil)

	// Get genesis state
	genesisHeader := rawdb.ReadHeader(db, rawdb.ReadCanonicalHash(db, 0), 0)
	if genesisHeader == nil {
		log.Fatal("Genesis header not found")
	}

	// Initialize or load state
	var currentState *state.StateDB
	currentState, err = state.New(genesisHeader.Root, stateDB)
	if err != nil {
		log.Printf("Could not load genesis state (root: %x), creating new", genesisHeader.Root)
		currentState, _ = state.New(common.Hash{}, stateDB)

		// Set up initial balance for test address
		if *testAddr != "" {
			addr := common.HexToAddress(*testAddr)
			// Set a reasonable initial balance (not 2T)
			initialBalance := new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)) // 1M LUX
			currentState.SetBalance(addr, uint256.MustFromBig(initialBalance), 0)
			log.Printf("Set initial balance for %s: 1,000,000 LUX", *testAddr)
		}
	}

	// Check current balance
	if *testAddr != "" {
		addr := common.HexToAddress(*testAddr)
		balance := currentState.GetBalance(addr)
		log.Printf("Current balance of %s: %s LUX", *testAddr,
			new(big.Int).Div(balance.ToBig(), big.NewInt(1e18)).String())
	}

	// Determine block range
	end := *endBlock
	if end == 0 || end > *headNumber {
		end = *headNumber
	}

	log.Printf("Processing blocks %d to %d", *startBlock, end)

	// Process blocks
	for blockNum := *startBlock; blockNum <= end; blockNum++ {
		hash := rawdb.ReadCanonicalHash(db, blockNum)
		if hash == (common.Hash{}) {
			log.Printf("Block %d: hash not found", blockNum)
			continue
		}

		block := rawdb.ReadBlock(db, hash, blockNum)
		if block == nil {
			log.Printf("Block %d: not found", blockNum)
			continue
		}

		header := block.Header()
		txCount := len(block.Transactions())

		if *verbose || blockNum%100 == 0 {
			log.Printf("Block %d: %d transactions", blockNum, txCount)
		}

		// Process transactions
		if txCount > 0 {
			// Create EVM context
			context := core.NewEVMBlockContext(header, nil, &header.Coinbase)
			vmConfig := vm.Config{}

			for i, tx := range block.Transactions() {
				// Get sender
				signer := types.LatestSigner(chainConfig)
				from, err := types.Sender(signer, tx)
				if err != nil {
					if *verbose {
						log.Printf("  TX %d: Could not get sender: %v", i, err)
					}
					continue
				}

				// Create EVM
				txContext := core.NewEVMTxContext(from, tx.GasPrice())
				evm := vm.NewEVM(context, txContext, currentState, chainConfig, vmConfig)

				// Apply transaction
				msg := &core.Message{
					From:              from,
					To:                tx.To(),
					Value:             tx.Value(),
					GasLimit:          tx.Gas(),
					GasPrice:          tx.GasPrice(),
					GasFeeCap:         tx.GasFeeCap(),
					GasTipCap:         tx.GasTipCap(),
					Data:              tx.Data(),
					AccessList:        tx.AccessList(),
					SkipNonceChecks:   true,
					SkipFromEOACheck:  true,
				}

				currentState.SetTxContext(tx.Hash(), i)
				_, err = core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(block.GasLimit()))

				if err != nil && *verbose {
					log.Printf("  TX %d: Failed to apply: %v", i, err)
				}
			}
		}

		// Commit state periodically
		if blockNum%100 == 0 {
			root, err := currentState.Commit(blockNum, false)
			if err != nil {
				log.Printf("Failed to commit at block %d: %v", blockNum, err)
			} else if *verbose {
				log.Printf("Committed state at block %d, root: %x", blockNum, root)
			}

			// Check balance periodically
			if *testAddr != "" && blockNum%1000 == 0 {
				addr := common.HexToAddress(*testAddr)
				balance := currentState.GetBalance(addr)
				log.Printf("Balance at block %d: %s LUX", blockNum,
					new(big.Int).Div(balance.ToBig(), big.NewInt(1e18)).String())
			}

			// Recreate state to avoid memory issues
			currentState, err = state.New(root, stateDB)
			if err != nil {
				log.Printf("Failed to recreate state: %v", err)
				currentState, _ = state.New(common.Hash{}, stateDB)
			}
		}
	}

	// Final commit
	finalRoot, err := currentState.Commit(end, false)
	if err != nil {
		log.Printf("Failed final commit: %v", err)
	} else {
		log.Printf("Final state root: %x", finalRoot)

		// Update head if successful
		if hash := rawdb.ReadCanonicalHash(db, end); hash != (common.Hash{}) {
			rawdb.WriteHeadBlockHash(db, hash)
			rawdb.WriteHeadHeaderHash(db, hash)
		}
	}

	// Final balance check
	if *testAddr != "" {
		addr := common.HexToAddress(*testAddr)
		balance := currentState.GetBalance(addr)
		log.Printf("\nFinal balance of %s: %s LUX", *testAddr,
			new(big.Int).Div(balance.ToBig(), big.NewInt(1e18)).String())
	}

	log.Printf("\nState replay complete!")
}