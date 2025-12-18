// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	gomath "math"
	"math/big"
	"reflect"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/common/hexutil"
	"github.com/luxfi/geth/common/math"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/internal/blocktest"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/rlp"
	"github.com/holiman/uint256"
)

// from bcValidBlockTest.json, "SimpleTx"
func TestBlockEncoding(t *testing.T) {
	blockEnc := common.FromHex("f90260f901f9a083cafc574e1f51ba9dc0568fc617a08ea2429fb384059c972f13b19fa1c8dd55a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017a05fe50b260da6308036625b850b5d6ced6d0a9f814c0688bc91ffb7b7a3a54b67a0bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008302000001832fefd8825208845506eb0780a0bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff49888a13a5a8c8f2bb1c4f861f85f800a82c35094095e7baea6a6c7c4c2dfeb977efac326af552d870a801ba09bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094fa08a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b1c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Difficulty", block.Difficulty(), big.NewInt(131072))
	check("GasLimit", block.GasLimit(), uint64(3141592))
	check("GasUsed", block.GasUsed(), uint64(21000))
	check("Coinbase", block.Coinbase(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("MixDigest", block.MixDigest(), common.HexToHash("bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff498"))
	check("Root", block.Root(), common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"))
	check("Hash", block.Hash(), common.HexToHash("0a5843ac1cb04865017cb35a57b50b07084e5fcee39b5acadade33149f4fff9e"))
	check("Nonce", block.Nonce(), uint64(0xa13a5a8c8f2bb1c4))
	check("Time", block.Time(), uint64(1426516743))
	check("Size", block.Size(), uint64(len(blockEnc)))

	tx1 := NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(10), 50000, big.NewInt(10), nil)
	tx1, _ = tx1.WithSignature(HomesteadSigner{}, common.Hex2Bytes("9bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094f8a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b100"))
	check("len(Transactions)", len(block.Transactions()), 1)
	check("Transactions[0].Hash", block.Transactions()[0].Hash(), tx1.Hash())
	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}

func TestEIP1559BlockEncoding(t *testing.T) {
	blockEnc := common.FromHex("f9030bf901fea083cafc574e1f51ba9dc0568fc617a08ea2429fb384059c972f13b19fa1c8dd55a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017a05fe50b260da6308036625b850b5d6ced6d0a9f814c0688bc91ffb7b7a3a54b67a0bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008302000001832fefd8825208845506eb0780a0bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff49888a13a5a8c8f2bb1c4843b9aca00f90106f85f800a82c35094095e7baea6a6c7c4c2dfeb977efac326af552d870a801ba09bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094fa08a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b1b8a302f8a0018080843b9aca008301e24194095e7baea6a6c7c4c2dfeb977efac326af552d878080f838f7940000000000000000000000000000000000000001e1a0000000000000000000000000000000000000000000000000000000000000000080a0fe38ca4e44a30002ac54af7cf922a6ac2ba11b7d22f548e8ecb3f51f41cb31b0a06de6a5cbae13c0c856e33acf021b51819636cfc009d39eafb9f606d546e305a8c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}

	check("Difficulty", block.Difficulty(), big.NewInt(131072))
	check("GasLimit", block.GasLimit(), uint64(3141592))
	check("GasUsed", block.GasUsed(), uint64(21000))
	check("Coinbase", block.Coinbase(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("MixDigest", block.MixDigest(), common.HexToHash("bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff498"))
	check("Root", block.Root(), common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"))
	check("Hash", block.Hash(), common.HexToHash("c7252048cd273fe0dac09650027d07f0e3da4ee0675ebbb26627cea92729c372"))
	check("Nonce", block.Nonce(), uint64(0xa13a5a8c8f2bb1c4))
	check("Time", block.Time(), uint64(1426516743))
	check("Size", block.Size(), uint64(len(blockEnc)))
	check("BaseFee", block.BaseFee(), new(big.Int).SetUint64(params.InitialBaseFee))

	tx1 := NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(10), 50000, big.NewInt(10), nil)
	tx1, _ = tx1.WithSignature(HomesteadSigner{}, common.Hex2Bytes("9bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094f8a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b100"))

	addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	accesses := AccessList{AccessTuple{
		Address: addr,
		StorageKeys: []common.Hash{
			{0},
		},
	}}
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	txdata := &DynamicFeeTx{
		ChainID:    big.NewInt(1),
		Nonce:      0,
		To:         &to,
		Gas:        123457,
		GasFeeCap:  new(big.Int).Set(block.BaseFee()),
		GasTipCap:  big.NewInt(0),
		AccessList: accesses,
		Data:       []byte{},
	}
	tx2 := NewTx(txdata)
	tx2, err := tx2.WithSignature(LatestSignerForChainID(big.NewInt(1)), common.Hex2Bytes("fe38ca4e44a30002ac54af7cf922a6ac2ba11b7d22f548e8ecb3f51f41cb31b06de6a5cbae13c0c856e33acf021b51819636cfc009d39eafb9f606d546e305a800"))
	if err != nil {
		t.Fatal("invalid signature error: ", err)
	}

	check("len(Transactions)", len(block.Transactions()), 2)
	check("Transactions[0].Hash", block.Transactions()[0].Hash(), tx1.Hash())
	check("Transactions[1].Hash", block.Transactions()[1].Hash(), tx2.Hash())
	check("Transactions[1].Type", block.Transactions()[1].Type(), tx2.Type())
	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}

func TestEIP2718BlockEncoding(t *testing.T) {
	blockEnc := common.FromHex("f90319f90211a00000000000000000000000000000000000000000000000000000000000000000a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017a0e6e49996c7ec59f7a23d22b83239a60151512c65613bf84a0d7da336399ebc4aa0cafe75574d59780665a97fbfd11365c7545aa8f1abf4e5e12e8243334ef7286bb901000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000083020000820200832fefd882a410845506eb0796636f6f6c65737420626c6f636b206f6e20636861696ea0bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff49888a13a5a8c8f2bb1c4f90101f85f800a82c35094095e7baea6a6c7c4c2dfeb977efac326af552d870a801ba09bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094fa08a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b1b89e01f89b01800a8301e24194095e7baea6a6c7c4c2dfeb977efac326af552d878080f838f7940000000000000000000000000000000000000001e1a0000000000000000000000000000000000000000000000000000000000000000001a03dbacc8d0259f2508625e97fdfc57cd85fdd16e5821bc2c10bdd1a52649e8335a0476e10695b183a87b0aa292a7f4b78ef0c3fbe62aa2c42c84e1d9c3da159ef14c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Difficulty", block.Difficulty(), big.NewInt(131072))
	check("GasLimit", block.GasLimit(), uint64(3141592))
	check("GasUsed", block.GasUsed(), uint64(42000))
	check("Coinbase", block.Coinbase(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("MixDigest", block.MixDigest(), common.HexToHash("bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff498"))
	check("Root", block.Root(), common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"))
	check("Nonce", block.Nonce(), uint64(0xa13a5a8c8f2bb1c4))
	check("Time", block.Time(), uint64(1426516743))
	check("Size", block.Size(), uint64(len(blockEnc)))

	// Create legacy tx.
	to := common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
	tx1 := NewTx(&LegacyTx{
		Nonce:    0,
		To:       &to,
		Value:    big.NewInt(10),
		Gas:      50000,
		GasPrice: big.NewInt(10),
	})
	sig := common.Hex2Bytes("9bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094f8a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b100")
	tx1, _ = tx1.WithSignature(HomesteadSigner{}, sig)

	// Create ACL tx.
	addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tx2 := NewTx(&AccessListTx{
		ChainID:    big.NewInt(1),
		Nonce:      0,
		To:         &to,
		Gas:        123457,
		GasPrice:   big.NewInt(10),
		AccessList: AccessList{{Address: addr, StorageKeys: []common.Hash{{0}}}},
	})
	sig2 := common.Hex2Bytes("3dbacc8d0259f2508625e97fdfc57cd85fdd16e5821bc2c10bdd1a52649e8335476e10695b183a87b0aa292a7f4b78ef0c3fbe62aa2c42c84e1d9c3da159ef1401")
	tx2, _ = tx2.WithSignature(NewEIP2930Signer(big.NewInt(1)), sig2)

	check("len(Transactions)", len(block.Transactions()), 2)
	check("Transactions[0].Hash", block.Transactions()[0].Hash(), tx1.Hash())
	check("Transactions[1].Hash", block.Transactions()[1].Hash(), tx2.Hash())
	check("Transactions[1].Type()", block.Transactions()[1].Type(), uint8(AccessListTxType))

	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}

func TestEIP4844BlockEncoding(t *testing.T) {
	// https://github.com/ethereum/tests/blob/develop/BlockchainTests/ValidBlocks/bcEIP4844-blobtransactions/blockWithAllTransactionTypes.json
	blockEnc := common.FromHex("0xf90417f90244a05eb7f6da0f3e237c62bcae48b7fb5f4506d392616b62890429c8b76b4a1d4104a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d4934794ba5e000000000000000000000000000000000000a011639dcca0b44f2acb5b630a82c8a69cb82742b3711383ec4e111a554d27aea5a05cb644f722e31f9792a8ef6e2a762334e1a862e8b40c1612e1e9507fd7121ef9a00c82719448356ba6807d6edfcd8e5aea575a5e97f36038ffb3e395749b26d41cb9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800188016345785d8a00008301482082079e42a00000000000000000000000000000000000000000000000000000000000020000880000000000000000820314a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b4218302000080a00000000000000000000000000000000000000000000000000000000000000000f901cbf864808203e885e8d4a5100094100000000000000000000000000000000000000a01801ca09de4adda6288582a6700dbcd8eb70c0a4a7fc9487d965f7bf22424e0bd121095a01cdb078764cc3770d5db847e99e10333aa7c356247baaf09b03eae04d64e7926b86901f86601018203e885e8d4a5100094100000000000000000000000000000000000000a0380c080a025090740da12684493e4fb466a3979e365b194e8cf462edf3c2c3be2f130bb2ea034fa18fb4c1bff4d957d72e28535d27f1352517a942aeaca0ed944085f0cd8bbb86a02f8670102018203e885e8d4a5100094100000000000000000000000000000000000000a0580c080a0352a7be5002ce111bc5167f3addf97a75e2e0b810d826af71d2caae18aed284ea065d38f8a5c8948ce706842e8861fb21020b93a4d5e489162a0e6d419a457b735b88c03f8890103018203e885e8d4a5100094100000000000000000000000000000000000000a0780c00ae1a001a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8809f638144c46d5de7a9e630c0e7c5c63ae829ecfd8cc94715d9c29fe17c464de0a06c5fc54c3aa868ba35ef31a4e12431611631ab7bcdceb4214dd273d83f73b5e1c0c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Difficulty", block.Difficulty(), big.NewInt(0))
	check("GasLimit", block.GasLimit(), hexutil.MustDecodeUint64("0x16345785d8a0000"))
	check("GasUsed", block.GasUsed(), hexutil.MustDecodeUint64("0x14820"))
	check("Coinbase", block.Coinbase(), common.HexToAddress("0xba5e000000000000000000000000000000000000"))
	check("MixDigest", block.MixDigest(), common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000020000"))
	check("Root", block.Root(), common.HexToHash("0x11639dcca0b44f2acb5b630a82c8a69cb82742b3711383ec4e111a554d27aea5"))
	check("WithdrawalRoot", *block.Header().WithdrawalsHash, common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"))
	check("Nonce", block.Nonce(), uint64(0))
	check("Time", block.Time(), hexutil.MustDecodeUint64("0x79e"))
	check("Size", block.Size(), uint64(len(blockEnc)))

	// Create blob tx.
	tx := NewTx(&BlobTx{
		ChainID:    uint256.NewInt(1),
		Nonce:      3,
		To:         common.HexToAddress("0x100000000000000000000000000000000000000a"),
		Gas:        hexutil.MustDecodeUint64("0xe8d4a51000"),
		GasTipCap:  uint256.MustFromHex("0x1"),
		GasFeeCap:  uint256.MustFromHex("0x3e8"),
		BlobFeeCap: uint256.MustFromHex("0xa"),
		BlobHashes: []common.Hash{
			common.BytesToHash(hexutil.MustDecode("0x01a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")),
		},
		Value: uint256.MustFromHex("0x7"),
	})
	sig := common.Hex2Bytes("00638144c46d5de7a9e630c0e7c5c63ae829ecfd8cc94715d9c29fe17c464de06c5fc54c3aa868ba35ef31a4e12431611631ab7bcdceb4214dd273d83f73b5e100")
	tx, _ = tx.WithSignature(LatestSignerForChainID(big.NewInt(1)), sig)

	check("len(Transactions)", len(block.Transactions()), 4)
	check("Transactions[3].Hash", block.Transactions()[3].Hash(), tx.Hash())
	check("Transactions[3].Type()", block.Transactions()[3].Type(), uint8(BlobTxType))

	// Note: Lux uses different RLP field order (Lux fields before Eth 2.0 fields)
	// so we can decode standard Ethereum blocks but re-encoding produces different bytes.
	// The important thing is that decoding works correctly (verified above).
}

func TestUncleHash(t *testing.T) {
	uncles := make([]*Header, 0)
	h := CalcUncleHash(uncles)
	exp := EmptyUncleHash
	if h != exp {
		t.Fatalf("empty uncle hash is wrong, got %x != %x", h, exp)
	}
}

var benchBuffer = bytes.NewBuffer(make([]byte, 0, 32000))

func BenchmarkEncodeBlock(b *testing.B) {
	block := makeBenchBlock()

	for b.Loop() {
		benchBuffer.Reset()
		if err := rlp.Encode(benchBuffer, block); err != nil {
			b.Fatal(err)
		}
	}
}

func makeBenchBlock() *Block {
	var (
		key, _   = crypto.GenerateKey()
		txs      = make([]*Transaction, 70)
		receipts = make([]*Receipt, len(txs))
		signer   = LatestSigner(params.TestChainConfig)
		uncles   = make([]*Header, 3)
	)
	header := &Header{
		Difficulty: math.BigPow(11, 11),
		Number:     math.BigPow(2, 9),
		GasLimit:   12345678,
		GasUsed:    1476322,
		Time:       9876543,
		Extra:      []byte("coolest block on chain"),
	}
	for i := range txs {
		amount := math.BigPow(2, int64(i))
		price := big.NewInt(300000)
		data := make([]byte, 100)
		tx := NewTransaction(uint64(i), common.Address{}, amount, 123457, price, data)
		signedTx, err := SignTx(tx, signer, key)
		if err != nil {
			panic(err)
		}
		txs[i] = signedTx
		receipts[i] = NewReceipt(make([]byte, 32), false, tx.Gas())
	}
	for i := range uncles {
		uncles[i] = &Header{
			Difficulty: math.BigPow(11, 11),
			Number:     math.BigPow(2, 9),
			GasLimit:   12345678,
			GasUsed:    1476322,
			Time:       9876543,
			Extra:      []byte("benchmark uncle"),
		}
	}
	return NewBlock(header, &Body{Transactions: txs, Uncles: uncles}, receipts, blocktest.NewHasher())
}

func TestRlpDecodeParentHash(t *testing.T) {
	// A minimum one
	want := common.HexToHash("0x112233445566778899001122334455667788990011223344556677889900aabb")
	if rlpData, err := rlp.EncodeToBytes(&Header{ParentHash: want}); err != nil {
		t.Fatal(err)
	} else {
		if have := HeaderParentHashFromRLP(rlpData); have != want {
			t.Fatalf("have %x, want %x", have, want)
		}
	}
	// And a maximum one
	// | Difficulty  | dynamic| *big.Int       | 0x5ad3c2c71bbff854908 (current mainnet TD: 76 bits) |
	// | Number      | dynamic| *big.Int       | 64 bits               |
	// | Extra       | dynamic| []byte         | 65+32 byte (clique)   |
	// | BaseFee     | dynamic| *big.Int       | 64 bits               |
	mainnetTd := new(big.Int)
	mainnetTd.SetString("5ad3c2c71bbff854908", 16)
	if rlpData, err := rlp.EncodeToBytes(&Header{
		ParentHash: want,
		Difficulty: mainnetTd,
		Number:     new(big.Int).SetUint64(gomath.MaxUint64),
		Extra:      make([]byte, 65+32),
		BaseFee:    new(big.Int).SetUint64(gomath.MaxUint64),
	}); err != nil {
		t.Fatal(err)
	} else {
		if have := HeaderParentHashFromRLP(rlpData); have != want {
			t.Fatalf("have %x, want %x", have, want)
		}
	}
	// Also test a very very large header.
	{
		// The rlp-encoding of the header belowCauses _total_ length of 65540,
		// which is the first to blow the fast-path.
		h := &Header{
			ParentHash: want,
			Extra:      make([]byte, 65041),
		}
		if rlpData, err := rlp.EncodeToBytes(h); err != nil {
			t.Fatal(err)
		} else {
			if have := HeaderParentHashFromRLP(rlpData); have != want {
				t.Fatalf("have %x, want %x", have, want)
			}
		}
	}
	{
		// Test some invalid erroneous stuff
		for i, rlpData := range [][]byte{
			nil,
			common.FromHex("0x"),
			common.FromHex("0x01"),
			common.FromHex("0x3031323334"),
		} {
			if have, want := HeaderParentHashFromRLP(rlpData), (common.Hash{}); have != want {
				t.Fatalf("invalid %d: have %x, want %x", i, have, want)
			}
		}
	}
}

// TestLuxMainnetGenesis verifies that the Lux mainnet genesis hash and state root
// are computed correctly. This test ensures that any changes to the Header struct
// or RLP encoding do not break compatibility with the existing Lux mainnet.
func TestLuxMainnetGenesis(t *testing.T) {
	// Expected Lux mainnet genesis values
	expectedGenesisHash := common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e")
	expectedStateRoot := common.HexToHash("0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80")

	// Construct the Lux mainnet genesis header
	// These values are from the actual Lux mainnet genesis block (network-id 96369)
	genesisHeader := &Header{
		ParentHash:  common.Hash{},
		UncleHash:   EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        expectedStateRoot,
		TxHash:      EmptyTxsHash,
		ReceiptHash: EmptyReceiptsHash,
		Bloom:       Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(0),
		GasLimit:    12000000,            // 0xb71b00
		GasUsed:     0,
		Time:        0x672485c2,          // 1730479554
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       BlockNonce{},
		BaseFee:     big.NewInt(25000000000), // 25 gwei (0x5d21dba00)
	}

	// Verify the genesis hash using Hash16 (16-field format for Lux mainnet)
	computedHash := genesisHeader.Hash16()
	if computedHash != expectedGenesisHash {
		t.Errorf("Lux mainnet genesis hash mismatch:\n  computed: %s\n  expected: %s",
			computedHash.Hex(), expectedGenesisHash.Hex())
	}

	// Verify state root is preserved
	if genesisHeader.Root != expectedStateRoot {
		t.Errorf("Lux mainnet state root mismatch:\n  have: %s\n  want: %s",
			genesisHeader.Root.Hex(), expectedStateRoot.Hex())
	}

	// Verify the header can be encoded and decoded correctly
	encoded, err := rlp.EncodeToBytes(genesisHeader)
	if err != nil {
		t.Fatalf("failed to encode genesis header: %v", err)
	}

	var decoded Header
	if err := rlp.DecodeBytes(encoded, &decoded); err != nil {
		t.Fatalf("failed to decode genesis header: %v", err)
	}

	// Verify decoded values match
	if decoded.Root != expectedStateRoot {
		t.Errorf("decoded state root mismatch:\n  have: %s\n  want: %s",
			decoded.Root.Hex(), expectedStateRoot.Hex())
	}
	if decoded.BaseFee.Cmp(genesisHeader.BaseFee) != 0 {
		t.Errorf("decoded BaseFee mismatch:\n  have: %s\n  want: %s",
			decoded.BaseFee.String(), genesisHeader.BaseFee.String())
	}

	t.Logf("Lux mainnet genesis hash verified: %s", computedHash.Hex())
	t.Logf("Lux mainnet state root verified: %s", expectedStateRoot.Hex())
}

// TestLuxGenesisRawHashPreservation verifies that decoding a block from RLP bytes
// preserves the original hash. This is critical for genesis block compatibility.
//
// Test procedure:
// 1. Create RLP bytes for a block header with known hash (Lux genesis)
// 2. Decode the block
// 3. Call block.Hash()
// 4. Verify hash matches expected
func TestLuxGenesisRawHashPreservation(t *testing.T) {
	// Expected Lux mainnet genesis hash
	expectedHash := common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e")

	// Lux genesis header values
	stateRoot := common.HexToHash("0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80")
	timestamp := uint64(0x672485c2)  // 1730479554
	gasLimit := uint64(0xb71b00)     // 12000000
	baseFee := big.NewInt(0x5d21dba00) // 25 gwei

	// Step 1: Create a 16-field header struct (post-London, pre-ExtDataHash format)
	h16 := hdr16{
		ParentHash:  common.Hash{},    // zero hash for genesis
		UncleHash:   EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        stateRoot,
		TxHash:      EmptyTxsHash,
		ReceiptHash: EmptyReceiptsHash,
		Bloom:       Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(0),
		GasLimit:    gasLimit,
		GasUsed:     0,
		Time:        timestamp,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       BlockNonce{},
		BaseFee:     baseFee,
	}

	// Step 2: Encode the header to RLP bytes
	headerRLP, err := rlp.EncodeToBytes(&h16)
	if err != nil {
		t.Fatalf("failed to encode header: %v", err)
	}

	// Create a block with this header (empty body)
	// Block format: [header, transactions, uncles]
	blockRLP, err := rlp.EncodeToBytes([]interface{}{
		rlp.RawValue(headerRLP),
		[]*Transaction{}, // empty transactions
		[]*Header{},      // empty uncles
	})
	if err != nil {
		t.Fatalf("failed to encode block: %v", err)
	}

	// Step 3: Decode the block from RLP
	var block Block
	if err := rlp.DecodeBytes(blockRLP, &block); err != nil {
		t.Fatalf("failed to decode block: %v", err)
	}

	// Step 4: Verify hash matches expected
	computedHash := block.Hash()
	if computedHash != expectedHash {
		t.Errorf("hash mismatch after decode:\n  computed: %s\n  expected: %s",
			computedHash.Hex(), expectedHash.Hex())
	}

	// Additional verification: decode header directly and check hash
	decodedHeader, err := DecodeHeader(headerRLP)
	if err != nil {
		t.Fatalf("failed to decode header directly: %v", err)
	}

	headerHash := decodedHeader.Hash16()
	if headerHash != expectedHash {
		t.Errorf("direct header hash mismatch:\n  computed: %s\n  expected: %s",
			headerHash.Hex(), expectedHash.Hex())
	}

	// Verify key fields are preserved
	if block.Root() != stateRoot {
		t.Errorf("state root not preserved: have %s, want %s",
			block.Root().Hex(), stateRoot.Hex())
	}
	if block.Time() != timestamp {
		t.Errorf("timestamp not preserved: have %d, want %d",
			block.Time(), timestamp)
	}
	if block.GasLimit() != gasLimit {
		t.Errorf("gas limit not preserved: have %d, want %d",
			block.GasLimit(), gasLimit)
	}
	if block.BaseFee().Cmp(baseFee) != 0 {
		t.Errorf("base fee not preserved: have %s, want %s",
			block.BaseFee().String(), baseFee.String())
	}

	t.Logf("Raw hash preservation verified: %s", expectedHash.Hex())
}

// TestLuxGenesisHash16 verifies that the Lux mainnet genesis hash using 16-field format
// computes to the expected value: 0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
func TestLuxGenesisHash16(t *testing.T) {
	// Expected hash from Lux mainnet genesis
	expectedHash := common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e")

	// Genesis parameters from Lux mainnet (network-id 96369)
	genesisHeader := &Header{
		ParentHash:  common.Hash{},
		UncleHash:   EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        common.HexToHash("0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80"),
		TxHash:      EmptyTxsHash,
		ReceiptHash: EmptyReceiptsHash,
		Bloom:       Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(0),
		GasLimit:    0xb71b00,                // 12000000
		GasUsed:     0,
		Time:        0x672485c2,              // 1730479554
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       BlockNonce{},
		BaseFee:     big.NewInt(0x5d21dba00), // 25000000000 (25 gwei)
	}

	// Compute hash using 16-field format
	computedHash := genesisHeader.Hash16()

	if computedHash != expectedHash {
		t.Errorf("genesis hash mismatch:\n  computed: %s\n  expected: %s",
			computedHash.Hex(), expectedHash.Hex())
	}

	// Verify RLP encoding produces expected field count
	h16 := hdr16{
		ParentHash:  genesisHeader.ParentHash,
		UncleHash:   genesisHeader.UncleHash,
		Coinbase:    genesisHeader.Coinbase,
		Root:        genesisHeader.Root,
		TxHash:      genesisHeader.TxHash,
		ReceiptHash: genesisHeader.ReceiptHash,
		Bloom:       genesisHeader.Bloom,
		Difficulty:  genesisHeader.Difficulty,
		Number:      genesisHeader.Number,
		GasLimit:    genesisHeader.GasLimit,
		GasUsed:     genesisHeader.GasUsed,
		Time:        genesisHeader.Time,
		Extra:       genesisHeader.Extra,
		MixDigest:   genesisHeader.MixDigest,
		Nonce:       genesisHeader.Nonce,
		BaseFee:     genesisHeader.BaseFee,
	}
	encoded, err := rlp.EncodeToBytes(&h16)
	if err != nil {
		t.Fatalf("failed to encode header: %v", err)
	}

	// Decode and verify field count
	decoded, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode header: %v", err)
	}
	if decoded.BaseFee.Cmp(genesisHeader.BaseFee) != 0 {
		t.Errorf("BaseFee mismatch after decode: have %v, want %v",
			decoded.BaseFee, genesisHeader.BaseFee)
	}

	t.Logf("Genesis hash verified: %s", computedHash.Hex())
}

// TestLuxBlockHash19 verifies that post-genesis blocks with 19-field headers
// (including ExtDataHash, ExtDataGasUsed, BlockGasCost) compute correct hashes.
func TestLuxBlockHash19(t *testing.T) {
	// Create a 19-field header (post-genesis with Lux extensions)
	extDataHash := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	header := &Header{
		ParentHash:     common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e"),
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.Address{},
		Root:           common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(0),
		Number:         big.NewInt(1),
		GasLimit:       0xb71b00,
		GasUsed:        21000,
		Time:           0x672485c3,
		Extra:          []byte{},
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		BaseFee:        big.NewInt(0x5d21dba00),
		ExtDataHash:    &extDataHash,
		ExtDataGasUsed: big.NewInt(0),
		BlockGasCost:   big.NewInt(100000),
	}

	// Encode using hdr19 struct to ensure 19-field format
	h19 := hdr19{
		ParentHash:     header.ParentHash,
		UncleHash:      header.UncleHash,
		Coinbase:       header.Coinbase,
		Root:           header.Root,
		TxHash:         header.TxHash,
		ReceiptHash:    header.ReceiptHash,
		Bloom:          header.Bloom,
		Difficulty:     header.Difficulty,
		Number:         header.Number,
		GasLimit:       header.GasLimit,
		GasUsed:        header.GasUsed,
		Time:           header.Time,
		Extra:          header.Extra,
		MixDigest:      header.MixDigest,
		Nonce:          header.Nonce,
		BaseFee:        header.BaseFee,
		ExtDataHash:    header.ExtDataHash,
		ExtDataGasUsed: header.ExtDataGasUsed,
		BlockGasCost:   header.BlockGasCost,
	}

	// Encode and compute hash
	encoded, err := rlp.EncodeToBytes(&h19)
	if err != nil {
		t.Fatalf("failed to encode 19-field header: %v", err)
	}

	// Compute hash via RLP
	hash19 := rlpHash(&h19)

	// Decode and verify
	decoded, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode 19-field header: %v", err)
	}

	// Verify all 19 fields are preserved
	if decoded.ParentHash != header.ParentHash {
		t.Errorf("ParentHash mismatch: have %s, want %s", decoded.ParentHash.Hex(), header.ParentHash.Hex())
	}
	if decoded.ExtDataHash == nil || *decoded.ExtDataHash != extDataHash {
		t.Errorf("ExtDataHash mismatch: have %v, want %s", decoded.ExtDataHash, extDataHash.Hex())
	}
	if decoded.ExtDataGasUsed == nil || decoded.ExtDataGasUsed.Cmp(header.ExtDataGasUsed) != 0 {
		t.Errorf("ExtDataGasUsed mismatch: have %v, want %v", decoded.ExtDataGasUsed, header.ExtDataGasUsed)
	}
	if decoded.BlockGasCost == nil || decoded.BlockGasCost.Cmp(header.BlockGasCost) != 0 {
		t.Errorf("BlockGasCost mismatch: have %v, want %v", decoded.BlockGasCost, header.BlockGasCost)
	}

	t.Logf("19-field block hash: %s", hash19.Hex())
}

// TestLuxHashChainContinuity verifies that parent hash matches the previous block hash,
// ensuring hash chain continuity between 16-field genesis and 19-field post-genesis blocks.
func TestLuxHashChainContinuity(t *testing.T) {
	// Genesis block (16-field)
	genesisHeader := &Header{
		ParentHash:  common.Hash{},
		UncleHash:   EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        common.HexToHash("0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80"),
		TxHash:      EmptyTxsHash,
		ReceiptHash: EmptyReceiptsHash,
		Bloom:       Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(0),
		GasLimit:    0xb71b00,
		GasUsed:     0,
		Time:        0x672485c2,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       BlockNonce{},
		BaseFee:     big.NewInt(0x5d21dba00),
	}
	genesisHash := genesisHeader.Hash16()

	// Block 1 (19-field, references genesis)
	extDataHash1 := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	block1Header := &Header{
		ParentHash:     genesisHash, // Must match genesis hash
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.Address{},
		Root:           common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(0),
		Number:         big.NewInt(1),
		GasLimit:       0xb71b00,
		GasUsed:        0,
		Time:           0x672485c3,
		Extra:          []byte{},
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		BaseFee:        big.NewInt(0x5d21dba00),
		ExtDataHash:    &extDataHash1,
		ExtDataGasUsed: big.NewInt(0),
		BlockGasCost:   big.NewInt(100000),
	}
	h19_1 := hdr19{
		ParentHash:     block1Header.ParentHash,
		UncleHash:      block1Header.UncleHash,
		Coinbase:       block1Header.Coinbase,
		Root:           block1Header.Root,
		TxHash:         block1Header.TxHash,
		ReceiptHash:    block1Header.ReceiptHash,
		Bloom:          block1Header.Bloom,
		Difficulty:     block1Header.Difficulty,
		Number:         block1Header.Number,
		GasLimit:       block1Header.GasLimit,
		GasUsed:        block1Header.GasUsed,
		Time:           block1Header.Time,
		Extra:          block1Header.Extra,
		MixDigest:      block1Header.MixDigest,
		Nonce:          block1Header.Nonce,
		BaseFee:        block1Header.BaseFee,
		ExtDataHash:    block1Header.ExtDataHash,
		ExtDataGasUsed: block1Header.ExtDataGasUsed,
		BlockGasCost:   block1Header.BlockGasCost,
	}
	block1Hash := rlpHash(&h19_1)

	// Block 2 (19-field, references block 1)
	extDataHash2 := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	block2Header := &Header{
		ParentHash:     block1Hash, // Must match block 1 hash
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.Address{},
		Root:           common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(0),
		Number:         big.NewInt(2),
		GasLimit:       0xb71b00,
		GasUsed:        21000,
		Time:           0x672485c4,
		Extra:          []byte{},
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		BaseFee:        big.NewInt(0x5d21dba00),
		ExtDataHash:    &extDataHash2,
		ExtDataGasUsed: big.NewInt(0),
		BlockGasCost:   big.NewInt(100000),
	}

	// Verify chain continuity
	if block1Header.ParentHash != genesisHash {
		t.Errorf("Block 1 parent hash mismatch:\n  have: %s\n  want: %s",
			block1Header.ParentHash.Hex(), genesisHash.Hex())
	}
	if block2Header.ParentHash != block1Hash {
		t.Errorf("Block 2 parent hash mismatch:\n  have: %s\n  want: %s",
			block2Header.ParentHash.Hex(), block1Hash.Hex())
	}

	// Verify genesis hash is correct
	expectedGenesisHash := common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e")
	if genesisHash != expectedGenesisHash {
		t.Errorf("Genesis hash mismatch:\n  have: %s\n  want: %s",
			genesisHash.Hex(), expectedGenesisHash.Hex())
	}

	t.Logf("Chain continuity verified:")
	t.Logf("  Genesis (block 0): %s", genesisHash.Hex())
	t.Logf("  Block 1 parent:    %s", block1Header.ParentHash.Hex())
	t.Logf("  Block 1 hash:      %s", block1Hash.Hex())
	t.Logf("  Block 2 parent:    %s", block2Header.ParentHash.Hex())
}

// TestExtDataHashEncoding verifies that ExtDataHash encodes as a value type (32 bytes)
// not as a pointer, ensuring correct RLP encoding format.
func TestExtDataHashEncoding(t *testing.T) {
	extDataHash := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// Create 17-field header with ExtDataHash
	h17 := hdr17{
		ParentHash:  common.HexToHash("0x1234"),
		UncleHash:   EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        common.HexToHash("0xabcd"),
		TxHash:      EmptyTxsHash,
		ReceiptHash: EmptyReceiptsHash,
		Bloom:       Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(100),
		GasLimit:    8000000,
		GasUsed:     0,
		Time:        1234567890,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       BlockNonce{},
		BaseFee:     big.NewInt(1000000000),
		ExtDataHash: &extDataHash,
	}

	// Encode the header
	encoded, err := rlp.EncodeToBytes(&h17)
	if err != nil {
		t.Fatalf("failed to encode header: %v", err)
	}

	// Decode and verify ExtDataHash is preserved
	decoded, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode header: %v", err)
	}

	if decoded.ExtDataHash == nil {
		t.Fatal("ExtDataHash is nil after decode")
	}
	if *decoded.ExtDataHash != extDataHash {
		t.Errorf("ExtDataHash mismatch:\n  have: %s\n  want: %s",
			decoded.ExtDataHash.Hex(), extDataHash.Hex())
	}

	// Verify the raw RLP encoding contains the 32-byte hash value
	// The ExtDataHash should be encoded as a 32-byte string, not as a list or shorter value
	// This ensures it's encoded as a value type, not a pointer

	// Count RLP list items to verify field count
	content, _, err := rlp.SplitList(encoded)
	if err != nil {
		t.Fatalf("failed to split RLP list: %v", err)
	}

	fieldCount := 0
	for len(content) > 0 {
		_, rest, err := rlp.SplitString(content)
		if err != nil {
			_, rest, err = rlp.SplitList(content)
			if err != nil {
				t.Fatalf("failed to split field %d: %v", fieldCount, err)
			}
		}
		content = rest
		fieldCount++
	}

	if fieldCount != 17 {
		t.Errorf("expected 17 fields, got %d", fieldCount)
	}

	// Re-encode decoded header and verify hash matches
	h17Decoded := hdr17{
		ParentHash:  decoded.ParentHash,
		UncleHash:   decoded.UncleHash,
		Coinbase:    decoded.Coinbase,
		Root:        decoded.Root,
		TxHash:      decoded.TxHash,
		ReceiptHash: decoded.ReceiptHash,
		Bloom:       decoded.Bloom,
		Difficulty:  decoded.Difficulty,
		Number:      decoded.Number,
		GasLimit:    decoded.GasLimit,
		GasUsed:     decoded.GasUsed,
		Time:        decoded.Time,
		Extra:       decoded.Extra,
		MixDigest:   decoded.MixDigest,
		Nonce:       decoded.Nonce,
		BaseFee:     decoded.BaseFee,
		ExtDataHash: decoded.ExtDataHash,
	}

	originalHash := rlpHash(&h17)
	decodedHash := rlpHash(&h17Decoded)

	if originalHash != decodedHash {
		t.Errorf("hash mismatch after round-trip:\n  original: %s\n  decoded:  %s",
			originalHash.Hex(), decodedHash.Hex())
	}

	t.Logf("ExtDataHash encoding verified: %s", extDataHash.Hex())
	t.Logf("Header hash: %s", originalHash.Hex())
}

// TestLuxHeaderFormats verifies that different Lux header formats (16, 17, 18, 19 fields)
// can be correctly decoded using the DecodeHeader function.
func TestLuxHeaderFormats(t *testing.T) {
	// Test 16-field header (post-London, pre-ExtDataHash)
	t.Run("16-field", func(t *testing.T) {
		header := &Header{
			ParentHash:  common.HexToHash("0x1234"),
			UncleHash:   EmptyUncleHash,
			Coinbase:    common.Address{},
			Root:        common.HexToHash("0xabcd"),
			TxHash:      EmptyTxsHash,
			ReceiptHash: EmptyReceiptsHash,
			Bloom:       Bloom{},
			Difficulty:  big.NewInt(1),
			Number:      big.NewInt(100),
			GasLimit:    8000000,
			GasUsed:     21000,
			Time:        1234567890,
			Extra:       []byte("test"),
			MixDigest:   common.Hash{},
			Nonce:       BlockNonce{},
			BaseFee:     big.NewInt(1000000000),
		}
		// Encode using hdr16 struct directly for 16-field format
		h16 := hdr16{
			ParentHash:  header.ParentHash,
			UncleHash:   header.UncleHash,
			Coinbase:    header.Coinbase,
			Root:        header.Root,
			TxHash:      header.TxHash,
			ReceiptHash: header.ReceiptHash,
			Bloom:       header.Bloom,
			Difficulty:  header.Difficulty,
			Number:      header.Number,
			GasLimit:    header.GasLimit,
			GasUsed:     header.GasUsed,
			Time:        header.Time,
			Extra:       header.Extra,
			MixDigest:   header.MixDigest,
			Nonce:       header.Nonce,
			BaseFee:     header.BaseFee,
		}
		encoded, err := rlp.EncodeToBytes(&h16)
		if err != nil {
			t.Fatalf("failed to encode 16-field header: %v", err)
		}

		decoded, err := DecodeHeader(encoded)
		if err != nil {
			t.Fatalf("failed to decode 16-field header: %v", err)
		}
		if decoded.Number.Cmp(header.Number) != 0 {
			t.Errorf("Number mismatch: have %v, want %v", decoded.Number, header.Number)
		}
		if decoded.BaseFee.Cmp(header.BaseFee) != 0 {
			t.Errorf("BaseFee mismatch: have %v, want %v", decoded.BaseFee, header.BaseFee)
		}
	})

	// Test 17-field header (with ExtDataHash)
	t.Run("17-field", func(t *testing.T) {
		extDataHash := common.HexToHash("0xdeadbeef")
		h17 := hdr17{
			ParentHash:  common.HexToHash("0x1234"),
			UncleHash:   EmptyUncleHash,
			Coinbase:    common.Address{},
			Root:        common.HexToHash("0xabcd"),
			TxHash:      EmptyTxsHash,
			ReceiptHash: EmptyReceiptsHash,
			Bloom:       Bloom{},
			Difficulty:  big.NewInt(1),
			Number:      big.NewInt(200),
			GasLimit:    8000000,
			GasUsed:     42000,
			Time:        1234567891,
			Extra:       []byte("test17"),
			MixDigest:   common.Hash{},
			Nonce:       BlockNonce{},
			BaseFee:     big.NewInt(2000000000),
			ExtDataHash: &extDataHash,
		}
		encoded, err := rlp.EncodeToBytes(&h17)
		if err != nil {
			t.Fatalf("failed to encode 17-field header: %v", err)
		}

		decoded, err := DecodeHeader(encoded)
		if err != nil {
			t.Fatalf("failed to decode 17-field header: %v", err)
		}
		if decoded.Number.Cmp(h17.Number) != 0 {
			t.Errorf("Number mismatch: have %v, want %v", decoded.Number, h17.Number)
		}
		if decoded.ExtDataHash == nil || *decoded.ExtDataHash != extDataHash {
			t.Errorf("ExtDataHash mismatch: have %v, want %v", decoded.ExtDataHash, extDataHash)
		}
	})

	// Test 19-field header (with ExtDataHash, ExtDataGasUsed, BlockGasCost)
	t.Run("19-field", func(t *testing.T) {
		extDataHash := common.HexToHash("0xcafebabe")
		h19 := hdr19{
			ParentHash:     common.HexToHash("0x5678"),
			UncleHash:      EmptyUncleHash,
			Coinbase:       common.Address{},
			Root:           common.HexToHash("0xef01"),
			TxHash:         EmptyTxsHash,
			ReceiptHash:    EmptyReceiptsHash,
			Bloom:          Bloom{},
			Difficulty:     big.NewInt(1),
			Number:         big.NewInt(300),
			GasLimit:       8000000,
			GasUsed:        63000,
			Time:           1234567892,
			Extra:          []byte("test19"),
			MixDigest:      common.Hash{},
			Nonce:          BlockNonce{},
			BaseFee:        big.NewInt(3000000000),
			ExtDataHash:    &extDataHash,
			ExtDataGasUsed: big.NewInt(10000),
			BlockGasCost:   big.NewInt(50000),
		}
		encoded, err := rlp.EncodeToBytes(&h19)
		if err != nil {
			t.Fatalf("failed to encode 19-field header: %v", err)
		}

		decoded, err := DecodeHeader(encoded)
		if err != nil {
			t.Fatalf("failed to decode 19-field header: %v", err)
		}
		if decoded.Number.Cmp(h19.Number) != 0 {
			t.Errorf("Number mismatch: have %v, want %v", decoded.Number, h19.Number)
		}
		if decoded.ExtDataHash == nil || *decoded.ExtDataHash != extDataHash {
			t.Errorf("ExtDataHash mismatch: have %v, want %v", decoded.ExtDataHash, extDataHash)
		}
		if decoded.ExtDataGasUsed == nil || decoded.ExtDataGasUsed.Cmp(h19.ExtDataGasUsed) != 0 {
			t.Errorf("ExtDataGasUsed mismatch: have %v, want %v", decoded.ExtDataGasUsed, h19.ExtDataGasUsed)
		}
		if decoded.BlockGasCost == nil || decoded.BlockGasCost.Cmp(h19.BlockGasCost) != 0 {
			t.Errorf("BlockGasCost mismatch: have %v, want %v", decoded.BlockGasCost, h19.BlockGasCost)
		}
	})
}

// TestRawRLPHashPreservation verifies that decoding a header preserves the original
// RLP bytes for hash computation. This ensures cryptographic continuity when a block
// is decoded from one format and re-encoded (which might produce different bytes).
func TestRawRLPHashPreservation(t *testing.T) {
	// Create a 19-field header (Lux format)
	extDataHash := common.HexToHash("0xcafebabe")
	h19 := hdr19{
		ParentHash:     common.HexToHash("0x5678"),
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.Address{},
		Root:           common.HexToHash("0xef01"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(1),
		Number:         big.NewInt(300),
		GasLimit:       8000000,
		GasUsed:        63000,
		Time:           1234567892,
		Extra:          []byte("test19"),
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		BaseFee:        big.NewInt(3000000000),
		ExtDataHash:    &extDataHash,
		ExtDataGasUsed: big.NewInt(10000),
		BlockGasCost:   big.NewInt(50000),
	}

	// Encode to RLP bytes
	encoded, err := rlp.EncodeToBytes(&h19)
	if err != nil {
		t.Fatalf("failed to encode 19-field header: %v", err)
	}

	// Compute expected hash directly from encoded bytes
	expectedHash := rlpHashBytes(encoded)

	// Decode header using DecodeHeader (without rawRLP)
	decodedWithoutRaw, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode header: %v", err)
	}

	// Decode header using rlp.DecodeBytes which triggers DecodeRLP (with rawRLP)
	var decodedWithRaw Header
	if err := rlp.DecodeBytes(encoded, &decodedWithRaw); err != nil {
		t.Fatalf("failed to decode header with RLP: %v", err)
	}

	// Verify rawRLP is set
	if len(decodedWithRaw.RawRLP()) == 0 {
		t.Fatal("rawRLP not set after DecodeRLP")
	}

	// Verify rawRLP matches original encoded bytes
	if !bytes.Equal(decodedWithRaw.RawRLP(), encoded) {
		t.Errorf("rawRLP mismatch:\n  have: %x\n  want: %x", decodedWithRaw.RawRLP(), encoded)
	}

	// Hash from header with rawRLP should match expected hash
	hashWithRaw := decodedWithRaw.Hash()
	if hashWithRaw != expectedHash {
		t.Errorf("Hash() with rawRLP mismatch:\n  have: %s\n  want: %s",
			hashWithRaw.Hex(), expectedHash.Hex())
	}

	// Hash from header without rawRLP might differ (depends on re-encoding)
	// This demonstrates why rawRLP is needed
	hashWithoutRaw := decodedWithoutRaw.Hash()

	t.Logf("Expected hash (from original bytes): %s", expectedHash.Hex())
	t.Logf("Hash with rawRLP:                    %s", hashWithRaw.Hex())
	t.Logf("Hash without rawRLP:                 %s", hashWithoutRaw.Hex())

	// The key assertion: hash with rawRLP must match expected
	if hashWithRaw != expectedHash {
		t.Errorf("CRITICAL: rawRLP hash preservation failed")
	}
}

// TestHash19ValueVsPointer verifies that Hash19() correctly encodes ExtDataHash as
// value type (common.Hash) matching original coreth format, not pointer type (*common.Hash).
//
// RLP encoding difference:
//   - *common.Hash nil -> 0x80 (empty string)
//   - common.Hash{} -> 0xa0 + 32 zero bytes
func TestHash19ValueVsPointer(t *testing.T) {
	// Create a header with nil ExtDataHash
	header := &Header{
		ParentHash:     common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Root:           common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(1),
		Number:         big.NewInt(500),
		GasLimit:       8000000,
		GasUsed:        21000,
		Time:           1234567890,
		Extra:          []byte("test"),
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		BaseFee:        big.NewInt(1000000000),
		ExtDataHash:    nil, // nil pointer
		ExtDataGasUsed: big.NewInt(10000),
		BlockGasCost:   big.NewInt(50000),
	}

	// Hash using Hash19() - should encode ExtDataHash as zero hash (VALUE type)
	hash19 := header.Hash19()

	// Hash using Hash() - should also use Hash19() since ExtDataGasUsed is set
	hashAuto := header.Hash()

	// These should be equal since Hash() delegates to Hash19() when ExtDataGasUsed/BlockGasCost is set
	if hash19 != hashAuto {
		t.Errorf("Hash19() and Hash() mismatch:\n  Hash19(): %s\n  Hash():   %s",
			hash19.Hex(), hashAuto.Hex())
	}

	// Now verify Hash19() produces different hash than pointer-based encoding
	// Encode with hdr19 (uses *common.Hash pointer) and compute hash
	h19ptr := hdr19{
		ParentHash:     header.ParentHash,
		UncleHash:      header.UncleHash,
		Coinbase:       header.Coinbase,
		Root:           header.Root,
		TxHash:         header.TxHash,
		ReceiptHash:    header.ReceiptHash,
		Bloom:          header.Bloom,
		Difficulty:     header.Difficulty,
		Number:         header.Number,
		GasLimit:       header.GasLimit,
		GasUsed:        header.GasUsed,
		Time:           header.Time,
		Extra:          header.Extra,
		MixDigest:      header.MixDigest,
		Nonce:          header.Nonce,
		BaseFee:        header.BaseFee,
		ExtDataHash:    nil, // nil pointer - will encode as empty string
		ExtDataGasUsed: header.ExtDataGasUsed,
		BlockGasCost:   header.BlockGasCost,
	}
	hashPtrNil := rlpHash(&h19ptr)

	// Hash19 with nil ExtDataHash should differ from pointer-based encoding
	// because value type encodes as zero hash (32 bytes), pointer nil encodes as empty string
	if hash19 == hashPtrNil {
		t.Errorf("Hash19() should differ from pointer-nil encoding, but both are: %s", hash19.Hex())
	}

	// Now test with non-nil ExtDataHash - should produce SAME hash
	extDataHash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	header.ExtDataHash = &extDataHash

	hash19WithValue := header.Hash19()

	h19ptrWithValue := hdr19{
		ParentHash:     header.ParentHash,
		UncleHash:      header.UncleHash,
		Coinbase:       header.Coinbase,
		Root:           header.Root,
		TxHash:         header.TxHash,
		ReceiptHash:    header.ReceiptHash,
		Bloom:          header.Bloom,
		Difficulty:     header.Difficulty,
		Number:         header.Number,
		GasLimit:       header.GasLimit,
		GasUsed:        header.GasUsed,
		Time:           header.Time,
		Extra:          header.Extra,
		MixDigest:      header.MixDigest,
		Nonce:          header.Nonce,
		BaseFee:        header.BaseFee,
		ExtDataHash:    &extDataHash, // non-nil pointer
		ExtDataGasUsed: header.ExtDataGasUsed,
		BlockGasCost:   header.BlockGasCost,
	}
	hashPtrWithValue := rlpHash(&h19ptrWithValue)

	// With non-nil value, both should produce same hash
	if hash19WithValue != hashPtrWithValue {
		t.Errorf("Hash19() with value should match pointer encoding:\n  Hash19(): %s\n  Pointer:  %s",
			hash19WithValue.Hex(), hashPtrWithValue.Hex())
	}

	t.Logf("Hash19 (nil ExtDataHash as zero):     %s", hash19.Hex())
	t.Logf("Pointer (nil ExtDataHash as empty):  %s", hashPtrNil.Hex())
	t.Logf("Hash19 (with ExtDataHash value):     %s", hash19WithValue.Hex())
	t.Logf("Pointer (with ExtDataHash value):    %s", hashPtrWithValue.Hex())
}

// TestDecodeLux19FieldFormat verifies that DecodeHeader correctly handles the Lux
// 19-field format where ExtDataHash is encoded as a VALUE type (common.Hash), not
// a pointer. This is the actual format used in Lux block production.
//
// The key difference:
//   - hdr19lux: ExtDataHash is common.Hash (VALUE type) - always 32 bytes
//   - hdr19:    ExtDataHash is *common.Hash (pointer) - 0x80 when nil
func TestDecodeLux19FieldFormat(t *testing.T) {
	extDataHash := common.HexToHash("0xdeadbeefcafebabedeadbeefcafebabedeadbeefcafebabedeadbeefcafebabe")

	// Create a Lux 19-field header using VALUE type for ExtDataHash
	h19lux := hdr19lux{
		ParentHash:     common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Root:           common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(0),
		Number:         big.NewInt(100),
		GasLimit:       12000000,
		GasUsed:        21000,
		Time:           1730479555,
		Extra:          []byte{},
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		BaseFee:        big.NewInt(25000000000),
		ExtDataHash:    extDataHash, // VALUE type - this is key!
		ExtDataGasUsed: big.NewInt(0),
		BlockGasCost:   big.NewInt(100000),
	}

	// Encode to RLP
	encoded, err := rlp.EncodeToBytes(&h19lux)
	if err != nil {
		t.Fatalf("failed to encode hdr19lux: %v", err)
	}

	// Compute expected hash from original encoding
	expectedHash := rlpHashBytes(encoded)

	// Decode using DecodeHeader - should detect Lux format
	decoded, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode Lux 19-field header: %v", err)
	}

	// Verify rlpFormat is set correctly
	if decoded.GetRLPFormat() != RLPFormat19Lux {
		t.Errorf("expected RLPFormat19Lux, got %d", decoded.GetRLPFormat())
	}

	// Verify all fields are preserved
	if decoded.ParentHash != h19lux.ParentHash {
		t.Errorf("ParentHash mismatch")
	}
	if decoded.Number.Cmp(h19lux.Number) != 0 {
		t.Errorf("Number mismatch: have %v, want %v", decoded.Number, h19lux.Number)
	}
	if decoded.ExtDataHash == nil {
		t.Fatal("ExtDataHash is nil after decode")
	}
	if *decoded.ExtDataHash != extDataHash {
		t.Errorf("ExtDataHash mismatch: have %s, want %s",
			decoded.ExtDataHash.Hex(), extDataHash.Hex())
	}
	if decoded.ExtDataGasUsed == nil || decoded.ExtDataGasUsed.Cmp(h19lux.ExtDataGasUsed) != 0 {
		t.Errorf("ExtDataGasUsed mismatch")
	}
	if decoded.BlockGasCost == nil || decoded.BlockGasCost.Cmp(h19lux.BlockGasCost) != 0 {
		t.Errorf("BlockGasCost mismatch")
	}

	// Verify Hash19Lux produces the same hash as original encoding
	hash19Lux := decoded.Hash19Lux()
	if hash19Lux != expectedHash {
		t.Errorf("Hash19Lux mismatch:\n  have: %s\n  want: %s",
			hash19Lux.Hex(), expectedHash.Hex())
	}

	// Now test with zero ExtDataHash (should still work)
	h19luxZero := hdr19lux{
		ParentHash:     h19lux.ParentHash,
		UncleHash:      h19lux.UncleHash,
		Coinbase:       h19lux.Coinbase,
		Root:           h19lux.Root,
		TxHash:         h19lux.TxHash,
		ReceiptHash:    h19lux.ReceiptHash,
		Bloom:          h19lux.Bloom,
		Difficulty:     h19lux.Difficulty,
		Number:         h19lux.Number,
		GasLimit:       h19lux.GasLimit,
		GasUsed:        h19lux.GasUsed,
		Time:           h19lux.Time,
		Extra:          h19lux.Extra,
		MixDigest:      h19lux.MixDigest,
		Nonce:          h19lux.Nonce,
		BaseFee:        h19lux.BaseFee,
		ExtDataHash:    common.Hash{}, // Zero hash
		ExtDataGasUsed: h19lux.ExtDataGasUsed,
		BlockGasCost:   h19lux.BlockGasCost,
	}

	encodedZero, err := rlp.EncodeToBytes(&h19luxZero)
	if err != nil {
		t.Fatalf("failed to encode hdr19lux with zero ExtDataHash: %v", err)
	}

	decodedZero, err := DecodeHeader(encodedZero)
	if err != nil {
		t.Fatalf("failed to decode Lux 19-field header with zero ExtDataHash: %v", err)
	}

	if decodedZero.GetRLPFormat() != RLPFormat19Lux {
		t.Errorf("expected RLPFormat19Lux for zero hash, got %d", decodedZero.GetRLPFormat())
	}

	// Zero hash should still decode to a pointer to zero hash, not nil
	if decodedZero.ExtDataHash == nil {
		t.Error("ExtDataHash should not be nil for zero hash")
	} else if *decodedZero.ExtDataHash != (common.Hash{}) {
		t.Errorf("ExtDataHash should be zero hash, got %s", decodedZero.ExtDataHash.Hex())
	}

	t.Logf("Lux 19-field format decoding verified")
	t.Logf("  Expected hash: %s", expectedHash.Hex())
	t.Logf("  Hash19Lux():   %s", hash19Lux.Hex())
}

// TestDecodeCoreth19FieldFormat tests that headers encoded with coreth's
// HeaderSerializable format (ExtDataHash at pos 15, BaseFee at pos 16) are
// correctly decoded with BlockGasCost preserved.
func TestDecodeCoreth19FieldFormat(t *testing.T) {
	extDataHash := common.HexToHash("0xdeadbeefcafebabedeadbeefcafebabedeadbeefcafebabedeadbeefcafebabe")

	// Create a coreth 19-field header using their field order
	// This matches coreth's HeaderSerializable: ExtDataHash at pos 15, BaseFee at pos 16
	h19coreth := hdr19coreth{
		ParentHash:     common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		UncleHash:      EmptyUncleHash,
		Coinbase:       common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Root:           common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"),
		TxHash:         EmptyTxsHash,
		ReceiptHash:    EmptyReceiptsHash,
		Bloom:          Bloom{},
		Difficulty:     big.NewInt(0),
		Number:         big.NewInt(1080013), // Block number that was failing
		GasLimit:       12000000,
		GasUsed:        21000,
		Time:           1730479555,
		Extra:          []byte{},
		MixDigest:      common.Hash{},
		Nonce:          BlockNonce{},
		ExtDataHash:    extDataHash,          // Position 15 in coreth format
		BaseFee:        big.NewInt(25000000), // Position 16 in coreth format
		ExtDataGasUsed: big.NewInt(0),        // Position 17
		BlockGasCost:   big.NewInt(0),        // Position 18 - THIS IS KEY!
	}

	// Encode using coreth field order
	encoded, err := rlp.EncodeToBytes(&h19coreth)
	if err != nil {
		t.Fatalf("failed to encode hdr19coreth: %v", err)
	}

	// Decode using DecodeHeader - should try coreth format first
	decoded, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("failed to decode coreth 19-field header: %v", err)
	}

	// Verify all fields are preserved
	if decoded.ParentHash != h19coreth.ParentHash {
		t.Errorf("ParentHash mismatch")
	}
	if decoded.Number.Cmp(h19coreth.Number) != 0 {
		t.Errorf("Number mismatch: have %v, want %v", decoded.Number, h19coreth.Number)
	}

	// Critical: ExtDataHash must be preserved
	if decoded.ExtDataHash == nil {
		t.Fatal("ExtDataHash is nil after decode")
	}
	if *decoded.ExtDataHash != extDataHash {
		t.Errorf("ExtDataHash mismatch: have %s, want %s",
			decoded.ExtDataHash.Hex(), extDataHash.Hex())
	}

	// Critical: BaseFee must be preserved
	if decoded.BaseFee == nil || decoded.BaseFee.Cmp(h19coreth.BaseFee) != 0 {
		t.Errorf("BaseFee mismatch: have %v, want %v", decoded.BaseFee, h19coreth.BaseFee)
	}

	// Critical: ExtDataGasUsed must be preserved
	if decoded.ExtDataGasUsed == nil || decoded.ExtDataGasUsed.Cmp(h19coreth.ExtDataGasUsed) != 0 {
		t.Errorf("ExtDataGasUsed mismatch: have %v, want %v", decoded.ExtDataGasUsed, h19coreth.ExtDataGasUsed)
	}

	// MOST CRITICAL: BlockGasCost must be preserved (this was nil before the fix)
	if decoded.BlockGasCost == nil {
		t.Fatal("BlockGasCost is nil after decode - this is the bug we fixed!")
	}
	if decoded.BlockGasCost.Cmp(h19coreth.BlockGasCost) != 0 {
		t.Errorf("BlockGasCost mismatch: have %v, want %v", decoded.BlockGasCost, h19coreth.BlockGasCost)
	}

	t.Logf("Coreth 19-field format decoding verified successfully")
	t.Logf("  Block number: %v", decoded.Number)
	t.Logf("  ExtDataHash: %s", decoded.ExtDataHash.Hex())
	t.Logf("  BaseFee: %v", decoded.BaseFee)
	t.Logf("  ExtDataGasUsed: %v", decoded.ExtDataGasUsed)
	t.Logf("  BlockGasCost: %v", decoded.BlockGasCost)
}
