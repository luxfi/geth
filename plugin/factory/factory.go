// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package factory

import (
	"github.com/luxfi/consensus/engine/chain/block"
	"github.com/luxfi/geth/plugin/evm"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
)

var (
	// ID this VM should be referenced by
	ID = ids.ID{'e', 'v', 'm'}
)

type Factory struct{}

func (*Factory) New(logger log.Logger) (interface{}, error) {
	return &evm.VM{}, nil
}

func NewPluginVM() block.ChainVM {
	return &evm.VM{IsPlugin: true}
}
