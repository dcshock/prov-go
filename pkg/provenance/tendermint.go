package provenance

import (
	"context"

	tmtypes "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
)

func (c *ProvenanceClient) GetBlockByHeight(height int64) (*tmtypes.GetBlockByHeightResponse, error) {
	res, err := (*c.TendermintClient()).GetBlockByHeight(context.Background(), &tmtypes.GetBlockByHeightRequest{
		Height: height,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
