package provenance

import (
	"context"

	tmtypes "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
)

func (c *ProvenanceClient) GetBlockByHeight(ctx context.Context, height int64) (*tmtypes.GetBlockByHeightResponse, error) {
	res, err := (*c.TendermintClient()).GetBlockByHeight(ctx, &tmtypes.GetBlockByHeightRequest{
		Height: height,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get the latest block height
func (c *ProvenanceClient) GetLatestBlock(ctx context.Context) (*tmtypes.GetLatestBlockResponse, error) {
	res, err := (*c.TendermintClient()).GetLatestBlock(ctx, &tmtypes.GetLatestBlockRequest{})
	if err != nil {
		return nil, err
	}

	return res, nil
}
