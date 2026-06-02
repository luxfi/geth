// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// rlpEncodeImpl is the function the ZAP-native ancient store calls when
// the caller hands it a structured item. Split into its own file so
// zap_ancient.go does not pull rlp into a tight inner loop signature
// (the zapWriteOp interface is `interface{}`, not `rlp.Encoder`).
package rawdb

import "github.com/luxfi/geth/rlp"

func rlpEncodeImpl(item interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(item)
}
