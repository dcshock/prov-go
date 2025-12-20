package provenance

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	registry "github.com/provenance-io/provenance/x/registry/types"
)

func (c *ProvenanceClient) RegistryBulkUpdate(entries []registry.RegistryEntry) (*tx.BroadcastTxResponse, error) {
	msg := &registry.MsgRegistryBulkUpdate{
		Signer:  c.Address,
		Entries: entries,
	}

	txBz, err := c.Grpc.SignTx([]sdk.Msg{msg}, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), 0)
	if err != nil {
		return nil, fmt.Errorf("error creating tx: %w", err)
	}

	resp, err := c.Grpc.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("error broadcasting transaction: %w", err)
	}

	return resp, nil
}
