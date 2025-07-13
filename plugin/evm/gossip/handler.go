// (c) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gossip

import (
	"context"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/utils/logging"
)

var _ p2p.Handler = (*txGossipHandler)(nil)

type txGossipHandler struct {
	log logging.Logger
}

func NewTxGossipHandler(log logging.Logger) *txGossipHandler {
	return &txGossipHandler{
		log: log,
	}
}

func (t *txGossipHandler) AppGossip(ctx context.Context, nodeID ids.NodeID, gossipBytes []byte) {
	// TODO: Implement gossip handling
	t.log.Debug("received app gossip", "nodeID", nodeID, "size", len(gossipBytes))
}

func (t *txGossipHandler) AppRequest(ctx context.Context, nodeID ids.NodeID, deadline uint64, requestBytes []byte) ([]byte, error) {
	// TODO: Implement request handling
	t.log.Debug("received app request", "nodeID", nodeID, "size", len(requestBytes))
	return nil, nil
}