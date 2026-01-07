package provenance

import (
	"context"

	stakingtypes "cosmossdk.io/api/cosmos/staking/v1beta1"
)

func (c *ProvenanceClient) GetStakingValidators(ctx context.Context) (*stakingtypes.QueryValidatorsResponse, error) {
	res, err := (*c.StakingClient()).Validators(ctx, &stakingtypes.QueryValidatorsRequest{})
	if err != nil {
		return nil, err
	}
	return res, nil
}
