// (c) 2021-2022, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/geth/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
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
