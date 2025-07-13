package miner

import (
	"math/big"

	"github.com/luxfi/coreth/core/txpool"
	"github.com/luxfi/coreth/core/types"
	"github.com/ava-labs/libevm/common"
)

type TransactionsByPriceAndNonce = transactionsByPriceAndNonce

func NewTransactionsByPriceAndNonce(signer types.Signer, txs map[common.Address][]*txpool.LazyTransaction, baseFee *big.Int) *TransactionsByPriceAndNonce {
	return newTransactionsByPriceAndNonce(signer, txs, baseFee)
}
