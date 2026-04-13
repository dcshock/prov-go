package contract

import (
	"context"
	"fmt"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// ContractGrantDetail is one contract + filter + limit pair for ContractExecutionAuthorization (Kotlin ContractGrantDetail).
type ContractGrantDetail struct {
	Contract string
	Filter   *codectypes.Any
	Limit    *codectypes.Any
}

// GrantDetailThisContract is a ContractGrantDetail for this client's contract address (Kotlin ContractGrantDetail(contractAddress(), filter, limit)).
func (b *BaseClient) GrantDetailThisContract(filter, limit *codectypes.Any) ContractGrantDetail {
	return ContractGrantDetail{Contract: b.ContractAddr, Filter: filter, Limit: limit}
}

// BuildGrantContractExecutionMsg builds MsgGrant with granter granting grantee a ContractExecutionAuthorization for this contract (Kotlin getGrantContractExecutionMsg; grantee defaults to primary signer when using the *ForPrimarySigner variant).
//
// Use grantee = PrimarySignerAddress() when the granter is the user and the operator is the grantee.
func (b *BaseClient) BuildGrantContractExecutionMsg(granter, grantee string, expiration time.Time, details []ContractGrantDetail) (*authz.MsgGrant, error) {
	grants := make([]wasmtypes.ContractGrant, len(details))
	for i, d := range details {
		grants[i] = wasmtypes.ContractGrant{
			Contract: d.Contract,
			Filter:   d.Filter,
			Limit:    d.Limit,
		}
	}
	auth := &wasmtypes.ContractExecutionAuthorization{Grants: grants}
	exp := expiration
	g, err := authz.NewGrant(grantBlockTime(), auth, &exp)
	if err != nil {
		return nil, fmt.Errorf("demoprime/contract: NewGrant: %w", err)
	}
	return &authz.MsgGrant{
		Granter: granter,
		Grantee: grantee,
		Grant:   g,
	}, nil
}

// BuildGrantContractExecutionMsgForPrimarySigner is Kotlin getGrantContractExecutionMsg: granter is the user account, grantee is PrimarySignerAddress().
func (b *BaseClient) BuildGrantContractExecutionMsgForPrimarySigner(granter string, expiration time.Time, details []ContractGrantDetail) (*authz.MsgGrant, error) {
	return b.BuildGrantContractExecutionMsg(granter, b.PrimarySignerAddress(), expiration, details)
}

// GrantContractExecution broadcasts MsgGrant where the primary signer is granter and grantee receives execution rights (Kotlin grantContractExecution).
func (b *BaseClient) GrantContractExecution(ctx context.Context, grantee string, expiration time.Time, details []ContractGrantDetail) error {
	msg, err := b.BuildGrantContractExecutionMsg(b.PrimarySignerAddress(), grantee, expiration, details)
	if err != nil {
		return err
	}
	_, err = b.BroadcastMsgs(ctx, []sdk.Msg{msg})
	return err
}

// BuildGrantContractCommitFundsMsg builds MsgGrant: granter authorizes contract (grantee) to submit MsgCommitFunds on their behalf (Kotlin getGrantContractCommitFundsMsg).
func (b *BaseClient) BuildGrantContractCommitFundsMsg(granter string, expiration time.Time) (*authz.MsgGrant, error) {
	gen := authz.NewGenericAuthorization(msgTypeURLCommitFunds)
	exp := expiration
	g, err := authz.NewGrant(grantBlockTime(), gen, &exp)
	if err != nil {
		return nil, fmt.Errorf("demoprime/contract: NewGrant: %w", err)
	}
	return &authz.MsgGrant{
		Granter: granter,
		Grantee: b.ContractAddr,
		Grant:   g,
	}, nil
}

// GrantContractCommitFunds broadcasts commit-funds authz with primary signer as granter (Kotlin grantContractCommitFunds).
func (b *BaseClient) GrantContractCommitFunds(ctx context.Context, expiration time.Time) error {
	msg, err := b.BuildGrantContractCommitFundsMsg(b.PrimarySignerAddress(), expiration)
	if err != nil {
		return err
	}
	_, err = b.BroadcastMsgs(ctx, []sdk.Msg{msg})
	return err
}
