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

// Get node config
func (c *ProvenanceClient) GetNodeConfig(ctx context.Context) (*nodetypes.ConfigResponse, error) {
	res, err := (*c.NodeClient()).Config(ctx, &nodetypes.ConfigRequest{})
	if err != nil {
		return nil, err
	}
	return res, nil
}
