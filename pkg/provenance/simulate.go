package provenance

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"google.golang.org/grpc"
)

// SimulateTx returns the estimated gas for the transaction
func SimulateTx(grpcConn *grpc.ClientConn, txConfig client.TxConfig, txBuilder client.TxBuilder) (uint64, error) {
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return 0, fmt.Errorf("failed to encode tx: %w", err)
	}

	txClient := txtypes.NewServiceClient(grpcConn)

	resp, err := txClient.Simulate(context.Background(), &txtypes.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return 0, fmt.Errorf("simulate tx failed: %w", err)
	}

	return resp.GasInfo.GasUsed, nil
}
