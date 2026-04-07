// Package contract provides shared CosmWasm / authz helpers ported from Kotlin BaseContractClient.
package contract

import (
	"context"
	"fmt"
	"time"

	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/dcshock/prov-go/pkg/provenance"
)

// DefaultRelativeTimeoutHeight matches Kotlin BaseContractClient.RELATIVE_TIMEOUT_HEIGHT.
const DefaultRelativeTimeoutHeight int64 = 10

// BaseClient holds shared state for a CosmWasm contract on Provenance (Kotlin BaseContractClient / BaseContractQueryClient).
type BaseClient struct {
	Prov                  *provenance.ProvenanceClient
	ContractAddr          string
	RelativeTimeoutHeight int64
}

// NewBaseClient returns a base client for contractAddr. Relative timeout defaults to DefaultRelativeTimeoutHeight.
func NewBaseClient(prov *provenance.ProvenanceClient, contractAddr string) *BaseClient {
	return &BaseClient{
		Prov:                  prov,
		ContractAddr:          contractAddr,
		RelativeTimeoutHeight: DefaultRelativeTimeoutHeight,
	}
}

// ContractAddress returns the contract bech32 address.
func (b *BaseClient) ContractAddress() string { return b.ContractAddr }

// PrimarySignerAddress is the configured signing account (Kotlin primarySignerAddress).
func (b *BaseClient) PrimarySignerAddress() string {
	if b.Prov == nil {
		return ""
	}
	return b.Prov.Address
}

// RequireSigner returns an error if the provenance client has no private key.
func (b *BaseClient) RequireSigner() error {
	if b.Prov == nil || b.Prov.PrivKey == nil {
		return fmt.Errorf("demoprime/contract: provenance client has no signer")
	}
	return nil
}

func (b *BaseClient) authzQuery() authztypes.QueryClient {
	return authztypes.NewQueryClient(b.Prov.Grpc.Conn)
}

// CurrentHeight returns the latest committed block height (Kotlin pbcHandler.currentHeight).
func (b *BaseClient) CurrentHeight(ctx context.Context) (int64, error) {
	tc := b.Prov.TendermintClient()
	resp, err := (*tc).GetLatestBlock(ctx, &tendermint.GetLatestBlockRequest{})
	if err != nil {
		return 0, err
	}
	if resp.SdkBlock == nil || resp.SdkBlock.Header == nil {
		return 0, fmt.Errorf("demoprime/contract: empty GetLatestBlock sdk_block")
	}
	return resp.SdkBlock.Header.Height, nil
}

// CalculateTimeoutHeight returns currentHeight + RelativeTimeoutHeight (Kotlin calculateTimeoutHeight).
func (b *BaseClient) CalculateTimeoutHeight(ctx context.Context) (int64, error) {
	h, err := b.CurrentHeight(ctx)
	if err != nil {
		return 0, err
	}
	if b.RelativeTimeoutHeight < 0 {
		return 0, fmt.Errorf("demoprime/contract: RelativeTimeoutHeight cannot be negative")
	}
	return h + b.RelativeTimeoutHeight, nil
}

// BroadcastMsgs signs (via provenance.SignTx), broadcasts, waits for inclusion, and returns the tx response.
// Fails if TxResponse.Code != 0.
func (b *BaseClient) BroadcastMsgs(ctx context.Context, msgs []sdk.Msg) (*sdk.TxResponse, error) {
	if err := b.RequireSigner(); err != nil {
		return nil, err
	}
	seq := b.Prov.NextSequence()
	txBz, err := b.Prov.SignTx(msgs, b.Prov.PrivKey.Bytes(), b.Prov.AccountNumber, seq, 0)
	if err != nil {
		return nil, fmt.Errorf("demoprime/contract: sign tx: %w", err)
	}
	bcast, err := b.Prov.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("demoprime/contract: broadcast: %w", err)
	}
	if bcast.TxResponse == nil || bcast.TxResponse.TxHash == "" {
		return nil, fmt.Errorf("demoprime/contract: empty broadcast response")
	}
	got, err := b.Prov.WaitOnTx(bcast.TxResponse.TxHash)
	if err != nil {
		return nil, fmt.Errorf("demoprime/contract: wait on tx: %w", err)
	}
	txr := got.TxResponse
	if txr == nil {
		return nil, fmt.Errorf("demoprime/contract: nil tx in get-tx response")
	}
	if txr.Code != 0 {
		return txr, fmt.Errorf("demoprime/contract: tx failed code=%d codespace=%s raw_log=%s", txr.Code, txr.Codespace, txr.RawLog)
	}
	return txr, nil
}

// GrantClock is used when building authz.Grant via authz.NewGrant (expiration must be after block time).
func grantBlockTime() time.Time { return time.Now().UTC() }
