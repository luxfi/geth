// (c) 2019-2021, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"context"
	"fmt"

	"github.com/luxfi/node/ids"
)

var _ RequestHandler = NoopRequestHandler{}

// GossipHandler handles inbound gossip messages
type GossipHandler interface {
	HandleAtomicTx(nodeID ids.NodeID, msg AtomicTxGossip) error
	HandleEthTxs(nodeID ids.NodeID, msg EthTxsGossip) error
}

// RequestHandler interface handles incoming requests from peers
// Must have methods in format of handleType(context.Context, ids.NodeID, uint32, request Type) error
// so that the Request object of relevant Type can invoke its respective handle method
// on this struct.
// Also see GossipHandler for implementation style.
type RequestHandler interface {
	HandleStateTrieLeafsRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, leafsRequest LeafsRequest) ([]byte, error)
	HandleAtomicTrieLeafsRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, leafsRequest LeafsRequest) ([]byte, error)
	HandleBlockRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, request BlockRequest) ([]byte, error)
	HandleCodeRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, codeRequest CodeRequest) ([]byte, error)
	HandleMessageSignatureRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, signatureRequest MessageSignatureRequest) ([]byte, error)
	HandleBlockSignatureRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, signatureRequest BlockSignatureRequest) ([]byte, error)
}

// CrossChainRequestHandler interface handles incoming cross-chain requests
type CrossChainRequestHandler interface {
	HandleEthCallRequest(ctx context.Context, requestingChainID ids.ID, requestID uint32, ethCallRequest EthCallRequest) ([]byte, error)
}

// CrossChainRequest is an interface for cross-chain requests
type CrossChainRequest interface {
	fmt.Stringer
	Handle(ctx context.Context, requestingChainID ids.ID, requestID uint32, handler CrossChainRequestHandler) ([]byte, error)
}

// ResponseHandler handles response for a sent request
// Only one of OnResponse or OnFailure is called for a given requestID, not both
type ResponseHandler interface {
	// OnResponse is invoked when the peer responded to a request
	OnResponse(response []byte) error
	// OnFailure is invoked when there was a failure in processing a request
	OnFailure() error
}

type NoopRequestHandler struct{}

func (NoopRequestHandler) HandleStateTrieLeafsRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, leafsRequest LeafsRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleAtomicTrieLeafsRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, leafsRequest LeafsRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleBlockRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, request BlockRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleCodeRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, codeRequest CodeRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleMessageSignatureRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, signatureRequest MessageSignatureRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleBlockSignatureRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, signatureRequest BlockSignatureRequest) ([]byte, error) {
	return nil, nil
}
