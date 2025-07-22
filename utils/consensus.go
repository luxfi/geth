// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"errors"

	"github.com/luxfi/node/api/metrics"
	"github.com/luxfi/node/consensus"
	"github.com/luxfi/node/consensus/validators"
	"github.com/luxfi/node/consensus/validators/validatorstest"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/crypto/bls/signer/localsigner"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms/platformvm/warp"
)

var (
	testCChainID = ids.ID{'c', 'c', 'h', 'a', 'i', 'n', 't', 'e', 's', 't'}
	testXChainID = ids.ID{'t', 'e', 's', 't', 'x'}
	testChainID  = ids.ID{'t', 'e', 's', 't', 'c', 'h', 'a', 'i', 'n'}
)

func TestConsensusContext() *consensus.Context {
	signer, err := localsigner.New()
	if err != nil {
		panic(err)
	}
	pk := signer.PublicKey()
	networkID := constants.UnitTestID
	chainID := testChainID

	ctx := &consensus.Context{
		NetworkID:      networkID,
		SubnetID:       ids.Empty,
		ChainID:        chainID,
		NodeID:         ids.GenerateTestNodeID(),
		XChainID:       testXChainID,
		CChainID:       testCChainID,
		PublicKey:      pk,
		WarpSigner:     warp.NewSigner(signer, networkID, chainID),
		Log:            logging.NoLog{},
		BCLookup:       ids.NewAliaser(),
		Metrics:        metrics.NewPrefixGatherer(),
		ChainDataDir:   "",
		ValidatorState: NewTestValidatorState(),
	}

	return ctx
}

func NewTestValidatorState() *validatorstest.State {
	return &validatorstest.State{
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetSubnetIDF: func(_ context.Context, chainID ids.ID) (ids.ID, error) {
			subnetID, ok := map[ids.ID]ids.ID{
				constants.PlatformChainID: constants.PrimaryNetworkID,
				testXChainID:              constants.PrimaryNetworkID,
				testCChainID:              constants.PrimaryNetworkID,
			}[chainID]
			if !ok {
				return ids.Empty, errors.New("unknown chain")
			}
			return subnetID, nil
		},
		GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			return map[ids.NodeID]*validators.GetValidatorOutput{}, nil
		},
	}
}
