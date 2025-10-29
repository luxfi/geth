// Copyright (C) 2019-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/luxfi/log"
	"github.com/luxfi/node/utils/ulimit"
	"github.com/luxfi/node/vms/rpcchainvm"

	"github.com/luxfi/geth/node"
)

// CorethVM wraps geth node for luxd plugin
type CorethVM struct{}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("CorethVM/1.0.0")
		os.Exit(0)
	}

	if err := ulimit.Set(ulimit.DefaultFDLimit, log.Root()); err != nil {
		fmt.Printf("failed to set fd limit: %s\n", err)
		os.Exit(1)
	}

	// TODO: Implement VM interface for geth
	// For now, create a placeholder that will fail gracefully
	vm := &CorethVM{}
	rpcchainvm.Serve(context.Background(), vm)
}
