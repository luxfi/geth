// Simple state rebuilder for migrated blocks
package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb/badgerdb"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/triedb"
	"github.com/holiman/uint256"
)

func main() {
	dbPath := flag.String("db", "/home/z/.luxd/chainData/C/db/badgerdb/ethdb", "Database path")
	testAddr := flag.String("addr", "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714", "Test address")
	flag.Parse()

	// Open BadgerDB
	log.Printf("Opening database at %s", *dbPath)
	db, err := badgerdb.New(*dbPath, false)
	if err != nil {
		log.Fatalf("Failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Get head block info
	headHash := rawdb.ReadHeadBlockHash(db)
	if headHash == (common.Hash{}) {
		log.Fatal("No head block found")
	}

	headNumber := rawdb.ReadHeaderNumber(db, headHash)
	if headNumber == nil {
		log.Fatal("No head number found")
	}
	log.Printf("Head block: %d", *headNumber)

	// Initialize state DB
	trieDB := triedb.NewDatabase(db, &triedb.Config{
		Preimages: true,
	})
	stateDB := state.NewDatabase(trieDB, nil)

	// Try to load existing state or create new
	genesis := rawdb.ReadHeader(db, rawdb.ReadCanonicalHash(db, 0), 0)
	if genesis == nil {
		log.Fatal("Genesis not found")
	}

	currentState, err := state.New(genesis.Root, stateDB)
	if err != nil {
		log.Printf("Creating new state (genesis root %x not found)", genesis.Root)
		currentState, _ = state.New(common.Hash{}, stateDB)
	}

	// Process first 100 blocks to rebuild state
	chainConfig := params.MainnetChainConfig
	chainConfig.ChainID = big.NewInt(96369)

	processed := 0
	for blockNum := uint64(1); blockNum <= 100 && blockNum <= *headNumber; blockNum++ {
		hash := rawdb.ReadCanonicalHash(db, blockNum)
		if hash == (common.Hash{}) {
			continue
		}

		block := rawdb.ReadBlock(db, hash, blockNum)
		if block == nil {
			continue
		}

		// Process transactions
		for _, tx := range block.Transactions() {
			signer := types.LatestSigner(chainConfig)
			from, err := types.Sender(signer, tx)
			if err != nil {
				continue
			}

			// Update sender balance (deduct tx cost)
			if tx.Value() != nil && tx.Value().Sign() > 0 {
				fromBalance := currentState.GetBalance(from)
				if fromBalance.Cmp(uint256.MustFromBig(tx.Value())) >= 0 {
					currentState.SubBalance(from, uint256.MustFromBig(tx.Value()), 0)
					if tx.To() != nil {
						currentState.AddBalance(*tx.To(), uint256.MustFromBig(tx.Value()), 0)
					}
				}
			}
		}

		processed++
		if blockNum%10 == 0 {
			log.Printf("Processed block %d (%d txs)", blockNum, len(block.Transactions()))
		}
	}

	// Commit state
	root, err := currentState.Commit(uint64(processed), true, false)
	if err != nil {
		log.Printf("Failed to commit: %v", err)
	} else {
		log.Printf("State committed, new root: %x", root)
	}

	// Check test address balance
	if *testAddr != "" {
		addr := common.HexToAddress(*testAddr)
		balance := currentState.GetBalance(addr)
		balanceInLux := new(big.Int).Div(balance.ToBig(), big.NewInt(1e18))
		log.Printf("\nAddress %s balance: %s LUX", *testAddr, balanceInLux.String())

		if balanceInLux.Cmp(big.NewInt(2000000000000)) == 0 {
			log.Printf("⚠️  Balance still shows genesis amount (2T LUX)")
			log.Printf("State replay needs deeper integration with C-Chain VM")
		} else {
			log.Printf("✅ Balance updated from genesis!")
		}
	}

	fmt.Printf("\nProcessed %d blocks\n", processed)
}