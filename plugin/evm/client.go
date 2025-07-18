// (c) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/slog"

	"github.com/luxfi/node/api"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/crypto/secp256k1"
	"github.com/luxfi/node/utils/formatting"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/utils/json"
	"github.com/luxfi/node/utils/rpc"
	"github.com/luxfi/geth/plugin/evm/atomic"
	"github.com/luxfi/geth/plugin/evm/config"
)

// Interface compliance
var _ Client = (*evmClient)(nil)

// GetAtomicTxStatusReply defines the GetAtomicTxStatus replies returned from the API
type GetAtomicTxStatusReply struct {
	Status      atomic.Status `json:"status"`
	BlockHeight *json.Uint64  `json:"blockHeight,omitempty"`
}

// ExportKeyArgs are arguments for ExportKey
type ExportKeyArgs struct {
	api.UserPass
	Address string `json:"address"`
}

// ExportKeyReply is the response for ExportKey
type ExportKeyReply struct {
	PrivateKey    *secp256k1.PrivateKey `json:"privateKey"`
	PrivateKeyHex string                `json:"privateKeyHex"`
}

// ImportKeyArgs are arguments for ImportKey
type ImportKeyArgs struct {
	api.UserPass
	PrivateKey *secp256k1.PrivateKey `json:"privateKey"`
}

// ImportArgs are arguments for passing into Import requests
type ImportArgs struct {
	api.UserPass
	// Addresses that can be used to sign the import
	// If empty, addresses that can sign will be discovered
	From []string `json:"from"`

	// Chain the funds are coming from
	SourceChain string `json:"sourceChain"`

	// The address to import funds to
	To common.Address `json:"to"`

	// Basename of the address to import funds to
	BaseFee *big.Int `json:"baseFee"`
}

// ExportLUXArgs are arguments for ExportLUX
type ExportLUXArgs struct {
	api.UserPass
	// Amount to send
	Amount json.Uint64 `json:"amount"`

	// Chain the funds are going to
	TargetChain string `json:"targetChain"`

	// Address receiving the funds
	To string `json:"to"`

	// Memo field for the export tx
	Memo string `json:"memo"`
}

// ExportArgs are the arguments to Export
type ExportArgs struct {
	ExportLUXArgs
	// AssetID of the tokens to export - defaults to LUX
	AssetID string `json:"assetID"`
}

// SetLogLevelArgs defines the arguments for setting log level
type SetLogLevelArgs struct {
	Level string `json:"level"`
}

// ParseEthAddress parses a string into a common.Address
func ParseEthAddress(addrStr string) (common.Address, error) {
	if !common.IsHexAddress(addrStr) {
		return common.Address{}, fmt.Errorf("invalid address: %s", addrStr)
	}
	return common.HexToAddress(addrStr), nil
}

// Client interface for interacting with EVM [chain]
type Client interface {
	IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error)
	GetAtomicTxStatus(ctx context.Context, txID ids.ID, options ...rpc.Option) (atomic.Status, error)
	GetAtomicTx(ctx context.Context, txID ids.ID, options ...rpc.Option) ([]byte, error)
	GetAtomicUTXOs(ctx context.Context, addrs []ids.ShortID, sourceChain string, limit uint32, startAddress ids.ShortID, startUTXOID ids.ID, options ...rpc.Option) ([][]byte, ids.ShortID, ids.ID, error)
	ExportKey(ctx context.Context, userPass api.UserPass, addr common.Address, options ...rpc.Option) (*secp256k1.PrivateKey, string, error)
	ImportKey(ctx context.Context, userPass api.UserPass, privateKey *secp256k1.PrivateKey, options ...rpc.Option) (common.Address, error)
	Import(ctx context.Context, userPass api.UserPass, to common.Address, sourceChain string, options ...rpc.Option) (ids.ID, error)
	ExportLUX(ctx context.Context, userPass api.UserPass, amount uint64, to ids.ShortID, targetChain string, options ...rpc.Option) (ids.ID, error)
	Export(ctx context.Context, userPass api.UserPass, amount uint64, to ids.ShortID, targetChain string, assetID string, options ...rpc.Option) (ids.ID, error)
	StartCPUProfiler(ctx context.Context, options ...rpc.Option) error
	StopCPUProfiler(ctx context.Context, options ...rpc.Option) error
	MemoryProfile(ctx context.Context, options ...rpc.Option) error
	LockProfile(ctx context.Context, options ...rpc.Option) error
	SetLogLevel(ctx context.Context, level slog.Level, options ...rpc.Option) error
	GetVMConfig(ctx context.Context, options ...rpc.Option) (*config.Config, error)
}

// evmClient implementation for interacting with EVM [chain]
type evmClient struct {
	requester      rpc.EndpointRequester
	adminRequester rpc.EndpointRequester
}

// ConfigReply is the response from admin.getVMConfig
type ConfigReply struct {
	Config *config.Config `json:"config"`
}

// NewClient returns a Client for interacting with EVM [chain]
func NewClient(uri, chain string) Client {
	return &evmClient{
		requester:      rpc.NewEndpointRequester(fmt.Sprintf("%s/ext/bc/%s/lux", uri, chain)),
		adminRequester: rpc.NewEndpointRequester(fmt.Sprintf("%s/ext/bc/%s/admin", uri, chain)),
	}
}

// NewCChainClient returns a Client for interacting with the C Chain
func NewCChainClient(uri string) Client {
	return NewClient(uri, "C")
}

// IssueTx issues a transaction to a node and returns the TxID
func (c *evmClient) IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error) {
	res := &api.JSONTxID{}
	txStr, err := formatting.Encode(formatting.Hex, txBytes)
	if err != nil {
		return res.TxID, fmt.Errorf("problem hex encoding bytes: %w", err)
	}
	err = c.requester.SendRequest(ctx, "lux.issueTx", &api.FormattedTx{
		Tx:       txStr,
		Encoding: formatting.Hex,
	}, res, options...)
	return res.TxID, err
}

// GetAtomicTxStatus returns the status of [txID]
func (c *evmClient) GetAtomicTxStatus(ctx context.Context, txID ids.ID, options ...rpc.Option) (atomic.Status, error) {
	res := &GetAtomicTxStatusReply{}
	err := c.requester.SendRequest(ctx, "lux.getAtomicTxStatus", &api.JSONTxID{
		TxID: txID,
	}, res, options...)
	return res.Status, err
}

// GetAtomicTx returns the byte representation of [txID]
func (c *evmClient) GetAtomicTx(ctx context.Context, txID ids.ID, options ...rpc.Option) ([]byte, error) {
	res := &api.FormattedTx{}
	err := c.requester.SendRequest(ctx, "lux.getAtomicTx", &api.GetTxArgs{
		TxID:     txID,
		Encoding: formatting.Hex,
	}, res, options...)
	if err != nil {
		return nil, err
	}

	return formatting.Decode(formatting.Hex, res.Tx)
}

// GetAtomicUTXOs returns the byte representation of the atomic UTXOs controlled by [addresses]
// from [sourceChain]
func (c *evmClient) GetAtomicUTXOs(ctx context.Context, addrs []ids.ShortID, sourceChain string, limit uint32, startAddress ids.ShortID, startUTXOID ids.ID, options ...rpc.Option) ([][]byte, ids.ShortID, ids.ID, error) {
	res := &api.GetUTXOsReply{}
	err := c.requester.SendRequest(ctx, "lux.getUTXOs", &api.GetUTXOsArgs{
		Addresses:   ids.ShortIDsToStrings(addrs),
		SourceChain: sourceChain,
		Limit:       json.Uint32(limit),
		StartIndex: api.Index{
			Address: startAddress.String(),
			UTXO:    startUTXOID.String(),
		},
		Encoding: formatting.Hex,
	}, res, options...)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}

	utxos := make([][]byte, len(res.UTXOs))
	for i, utxo := range res.UTXOs {
		utxoBytes, err := formatting.Decode(res.Encoding, utxo)
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, err
		}
		utxos[i] = utxoBytes
	}
	endAddr, err := address.ParseToID(res.EndIndex.Address)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}
	endUTXOID, err := ids.FromString(res.EndIndex.UTXO)
	return utxos, endAddr, endUTXOID, err
}

// ExportKey returns the private key corresponding to [addr] controlled by [user]
// in both Lux standard format and hex format
func (c *evmClient) ExportKey(ctx context.Context, user api.UserPass, addr common.Address, options ...rpc.Option) (*secp256k1.PrivateKey, string, error) {
	res := &ExportKeyReply{}
	err := c.requester.SendRequest(ctx, "lux.exportKey", &ExportKeyArgs{
		UserPass: user,
		Address:  addr.Hex(),
	}, res, options...)
	return res.PrivateKey, res.PrivateKeyHex, err
}

// ImportKey imports [privateKey] to [user]
func (c *evmClient) ImportKey(ctx context.Context, user api.UserPass, privateKey *secp256k1.PrivateKey, options ...rpc.Option) (common.Address, error) {
	res := &api.JSONAddress{}
	err := c.requester.SendRequest(ctx, "lux.importKey", &ImportKeyArgs{
		UserPass:   user,
		PrivateKey: privateKey,
	}, res, options...)
	if err != nil {
		return common.Address{}, err
	}
	return ParseEthAddress(res.Address)
}

// Import sends an import transaction to import funds from [sourceChain] and
// returns the ID of the newly created transaction
func (c *evmClient) Import(ctx context.Context, user api.UserPass, to common.Address, sourceChain string, options ...rpc.Option) (ids.ID, error) {
	res := &api.JSONTxID{}
	err := c.requester.SendRequest(ctx, "lux.import", &ImportArgs{
		UserPass:    user,
		To:          to,
		SourceChain: sourceChain,
	}, res, options...)
	return res.TxID, err
}

// ExportLUX sends LUX from this chain to the address specified by [to].
// Returns the ID of the newly created atomic transaction
func (c *evmClient) ExportLUX(
	ctx context.Context,
	user api.UserPass,
	amount uint64,
	to ids.ShortID,
	targetChain string,
	options ...rpc.Option,
) (ids.ID, error) {
	return c.Export(ctx, user, amount, to, targetChain, "LUX", options...)
}

// Export sends an asset from this chain to the P/C-Chain.
// After this tx is accepted, the LUX must be imported to the P/C-chain with an importTx.
// Returns the ID of the newly created atomic transaction
func (c *evmClient) Export(
	ctx context.Context,
	user api.UserPass,
	amount uint64,
	to ids.ShortID,
	targetChain string,
	assetID string,
	options ...rpc.Option,
) (ids.ID, error) {
	res := &api.JSONTxID{}
	err := c.requester.SendRequest(ctx, "lux.export", &ExportArgs{
		ExportLUXArgs: ExportLUXArgs{
			UserPass:    user,
			Amount:      json.Uint64(amount),
			TargetChain: targetChain,
			To:          to.String(),
		},
		AssetID: assetID,
	}, res, options...)
	return res.TxID, err
}

func (c *evmClient) StartCPUProfiler(ctx context.Context, options ...rpc.Option) error {
	return c.adminRequester.SendRequest(ctx, "admin.startCPUProfiler", struct{}{}, &api.EmptyReply{}, options...)
}

func (c *evmClient) StopCPUProfiler(ctx context.Context, options ...rpc.Option) error {
	return c.adminRequester.SendRequest(ctx, "admin.stopCPUProfiler", struct{}{}, &api.EmptyReply{}, options...)
}

func (c *evmClient) MemoryProfile(ctx context.Context, options ...rpc.Option) error {
	return c.adminRequester.SendRequest(ctx, "admin.memoryProfile", struct{}{}, &api.EmptyReply{}, options...)
}

func (c *evmClient) LockProfile(ctx context.Context, options ...rpc.Option) error {
	return c.adminRequester.SendRequest(ctx, "admin.lockProfile", struct{}{}, &api.EmptyReply{}, options...)
}

// SetLogLevel dynamically sets the log level for the C Chain
func (c *evmClient) SetLogLevel(ctx context.Context, level slog.Level, options ...rpc.Option) error {
	return c.adminRequester.SendRequest(ctx, "admin.setLogLevel", &SetLogLevelArgs{
		Level: level.String(),
	}, &api.EmptyReply{}, options...)
}

// GetVMConfig returns the current config of the VM
func (c *evmClient) GetVMConfig(ctx context.Context, options ...rpc.Option) (*config.Config, error) {
	res := &ConfigReply{}
	err := c.adminRequester.SendRequest(ctx, "admin.getVMConfig", struct{}{}, res, options...)
	return res.Config, err
}
