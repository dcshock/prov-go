package provenance

import (
	"context"

	nodetypes "cosmossdk.io/api/cosmos/base/node/v1beta1"
)

func (c *ProvenanceClient) GetNodeInfo(ctx context.Context) (*nodetypes.StatusResponse, error) {
	res, err := (*c.NodeClient()).Status(ctx, &nodetypes.StatusRequest{})
	if err != nil {
		return nil, err
	}
	return res, nil
}
