// (c) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/coreth/plugin/evm/message"
)

// GossipHandler handles incoming gossip messages
type GossipHandler struct {
	log logging.Logger
	vm  *VM
}

func NewGossipHandler(vm *VM, log logging.Logger) *GossipHandler {
	return &GossipHandler{
		log: log,
		vm:  vm,
	}
}

func (h *GossipHandler) HandleAtomicTx(nodeID ids.NodeID, msg message.AtomicTxGossip) error {
	h.log.Debug("Received AtomicTx gossiped from peer", "peerID", nodeID, "txHash", msg.Tx.Hash())
	
	if msg.Tx == nil {
		h.log.Debug("Dropping AtomicTx message with empty tx")
		return nil
	}

	// Add to mempool
	if err := h.vm.AddRemoteTxs([]*Tx{{Tx: msg.Tx.SignedTx}}); err != nil {
		h.log.Trace("AppGossip: failed to add remote tx to mempool", "err", err, "txHash", msg.Tx.Hash())
	}
	return nil
}

func (h *GossipHandler) HandleEthTxs(nodeID ids.NodeID, msg message.EthTxsGossip) error {
	h.log.Debug("Received EthTxs gossiped from peer", "peerID", nodeID, "size", len(msg.Txs))
	
	if err := h.vm.AddRemoteTxsToMempool(msg.Txs); err != nil {
		h.log.Trace("AppGossip: failed to add remote txs", "err", err)
	}
	return nil
}