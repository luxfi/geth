package miner

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/txpool"
	"github.com/luxfi/geth/core/types"
)

type TransactionsByPriceAndNonce = transactionsByPriceAndNonce

func NewTransactionsByPriceAndNonce(signer types.Signer, txs map[common.Address][]*txpool.LazyTransaction, baseFee *big.Int) *TransactionsByPriceAndNonce {
	return newTransactionsByPriceAndNonce(signer, txs, baseFee)
}
