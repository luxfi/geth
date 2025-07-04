// (c) 2021-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"github.com/luxfi/coreth/core/state/snapshot"
	"github.com/luxfi/coreth/core/types"
	"github.com/ethereum/go-ethereum/common"
)

type BlockProvider interface {
	GetBlock(common.Hash, uint64) *types.Block
}

type SnapshotProvider interface {
	Snapshots() *snapshot.Tree
}

type SyncDataProvider interface {
	BlockProvider
	SnapshotProvider
}
