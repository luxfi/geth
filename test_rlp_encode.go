// +build ignore

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

func main() {
	type HdrOptional struct {
		Number  *big.Int
		BaseFee *big.Int     `rlp:"optional"`
		Ext     *common.Hash `rlp:"optional"`
	}
	type HdrNilString struct {
		Number  *big.Int
		BaseFee *big.Int     `rlp:"optional"`
		Ext     *common.Hash `rlp:"nilString"`
	}
	type HdrNil struct {
		Number  *big.Int
		BaseFee *big.Int     `rlp:"optional"`
		Ext     *common.Hash `rlp:"nil"`
	}
	
	hoNil := HdrOptional{Number: big.NewInt(1), BaseFee: big.NewInt(25000000000), Ext: nil}
	hnsNil := HdrNilString{Number: big.NewInt(1), BaseFee: big.NewInt(25000000000), Ext: nil}
	hnNil := HdrNil{Number: big.NewInt(1), BaseFee: big.NewInt(25000000000), Ext: nil}
	
	encHoNil, err1 := rlp.EncodeToBytes(hoNil)
	encHnsNil, err2 := rlp.EncodeToBytes(hnsNil)
	encHnNil, err3 := rlp.EncodeToBytes(hnNil)
	
	fmt.Printf("HdrOptional Ext=nil: %s (len=%d, err=%v)\n", hex.EncodeToString(encHoNil), len(encHoNil), err1)
	fmt.Printf("HdrNilString Ext=nil: %s (len=%d, err=%v)\n", hex.EncodeToString(encHnsNil), len(encHnsNil), err2)
	fmt.Printf("HdrNil Ext=nil:      %s (len=%d, err=%v)\n", hex.EncodeToString(encHnNil), len(encHnNil), err3)
	
	// Decode back
	var fields []rlp.RawValue
	rlp.DecodeBytes(encHoNil, &fields)
	fmt.Printf("HdrOptional fields: %d\n", len(fields))
	
	rlp.DecodeBytes(encHnsNil, &fields)
	fmt.Printf("HdrNilString fields: %d\n", len(fields))
	
	rlp.DecodeBytes(encHnNil, &fields)
	fmt.Printf("HdrNil fields: %d\n", len(fields))
}
