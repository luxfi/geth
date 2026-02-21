// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Genesis file with a mostly-empty chain configuration (ensure missing fields work)
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001338",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config": {
				"terminalTotalDifficulty": 0
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000001338",
	},
	// Genesis file with specific chain configurations
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001339",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
				"homesteadBlock"                : 42,
				"daoForkBlock"                  : 141,
				"daoForkSupport"                : true,
				"terminalTotalDifficulty": 0
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000001339",
	},
}

// Tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	t.Parallel()
	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := t.TempDir()

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := os.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		geth := runGeth(t, "--networkid", "1337", "--syncmode=full", "--cache", "16",
			"--datadir", datadir, "--maxpeers", "0", "--port", "0", "--authrpc.port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}

// TestCustomBackend tests that the backend selection and detection works properly.
// Tests badger backend only (always available). Pebble tests are in genesis_test_pebble.go.
func TestCustomBackend(t *testing.T) {
	t.Parallel()
	// Test backends, but only on 64-bit platforms
	if strconv.IntSize != 64 {
		t.Skip("Custom backends are only available on 64-bit platform")
	}
	genesis := `{
		"alloc"      : {},
		"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001338",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config": {
				"terminalTotalDifficulty": 0
			}
		}`
	type backendTest struct {
		initArgs   []string
		initExpect string
		execArgs   []string
		execExpect string
	}
	testfunc := func(t *testing.T, tt backendTest) error {
		// Create a temporary data directory to use and inspect later
		datadir := t.TempDir()

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := os.WriteFile(json, []byte(genesis), 0600); err != nil {
			return fmt.Errorf("failed to write genesis file: %v", err)
		}
		{ // Init
			args := append(tt.initArgs, "--datadir", datadir, "init", json)
			geth := runGeth(t, args...)
			geth.ExpectRegexp(tt.initExpect)
			geth.ExpectExit()
		}
		{ // Exec + query
			args := append(tt.execArgs, "--networkid", "1337", "--syncmode=full", "--cache", "16",
				"--datadir", datadir, "--maxpeers", "0", "--port", "0", "--authrpc.port", "0",
				"--nodiscover", "--nat", "none", "--ipcdisable",
				"--exec", "eth.getBlock(0).nonce", "console")
			geth := runGeth(t, args...)
			geth.ExpectRegexp(tt.execExpect)
			geth.ExpectExit()
		}
		return nil
	}
	// Test badger backend (always available)
	for i, tt := range []backendTest{
		{ // When not specified, it should default to badger
			execArgs:   []string{"--db.engine", "badger"},
			execExpect: "0x0000000000001338",
		},
		{ // Explicit badger
			initArgs:   []string{"--db.engine", "badger"},
			execArgs:   []string{"--db.engine", "badger"},
			execExpect: "0x0000000000001338",
		},
		{ // Explicit badger first, then autodiscover
			initArgs:   []string{"--db.engine", "badger"},
			execExpect: "0x0000000000001338",
		},
		{ // Reject invalid backend choice
			initArgs:   []string{"--db.engine", "mssql"},
			initExpect: `Fatal: Invalid choice for db.engine 'mssql', allowed 'leveldb', 'pebble', or 'badger'`,
			// Since the init fails, this will return the (default) mainnet genesis
			// block nonce
			execExpect: `0x0000000000000042`,
		},
	} {
		if err := testfunc(t, tt); err != nil {
			t.Fatalf("test %d: %v", i, err)
		}
	}
}
