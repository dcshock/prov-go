package contract

import (
	"context"
	"strings"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/dcshock/prov-go/pkg/provenance"
)

const msgTypeURLCommitFunds = "/provenance.exchange.v1.MsgCommitFundsRequest"

func isAuthzNotFound(err error) bool {
	if err == nil {
		return false
	}
	// gRPC / SDK variants
	if strings.Contains(strings.ToLower(err.Error()), "authorization not found") {
		return true
	}
	return false
}

func grantStillValid(g *authztypes.Grant, now time.Time) bool {
	if g == nil || g.Expiration == nil {
		return true
	}
	return !g.Expiration.Before(now)
}

func allGrants(ctx context.Context, c authztypes.QueryClient, req *authztypes.QueryGrantsRequest) ([]*authztypes.Grant, error) {
	var out []*authztypes.Grant
	var key []byte
	for {
		req.Pagination = &query.PageRequest{Key: key, Limit: 100}
		resp, err := c.Grants(ctx, req)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Grants...)
		if resp.Pagination == nil || len(resp.Pagination.NextKey) == 0 {
			break
		}
		key = resp.Pagination.NextKey
	}
	return out, nil
}

// ContractCommitmentGrants lists non-expired authz grants where grantee is this contract and authorization is MsgCommitFunds (Kotlin contractCommitmentGrants).
func (b *BaseClient) ContractCommitmentGrants(ctx context.Context, granter string) ([]*authztypes.Grant, error) {
	req := &authztypes.QueryGrantsRequest{
		Granter:    granter,
		Grantee:    b.ContractAddr,
		MsgTypeUrl: msgTypeURLCommitFunds,
	}
	grants, err := allGrants(ctx, b.authzQuery(), req)
	if err != nil {
		if isAuthzNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	now := time.Now().UTC()
	var kept []*authztypes.Grant
	for _, g := range grants {
		if grantStillValid(g, now) {
			kept = append(kept, g)
		}
	}
	return kept, nil
}

// ContractExecutionGrants returns wasm ContractGrant entries from ContractExecutionAuthorization between granter and grantee for this contract (Kotlin contractExecutionGrants).
func (b *BaseClient) ContractExecutionGrants(ctx context.Context, granter, grantee string) ([]wasmtypes.ContractGrant, error) {
	req := &authztypes.QueryGrantsRequest{
		Granter: granter,
		Grantee: grantee,
	}
	grants, err := allGrants(ctx, b.authzQuery(), req)
	if err != nil {
		return nil, err
	}
	cdc := provenance.Codec()
	now := time.Now().UTC()
	var out []wasmtypes.ContractGrant
	for _, g := range grants {
		if g == nil || !grantStillValid(g, now) {
			continue
		}
		var auth wasmtypes.ContractExecutionAuthorization
		if err := cdc.UnpackAny(g.Authorization, &auth); err != nil {
			continue
		}
		for _, cg := range auth.Grants {
			if cg.Contract == b.ContractAddr {
				out = append(out, cg)
			}
		}
	}
	return out, nil
}

// HasContractExecutionGrant reports whether grantee may execute this contract on behalf of granter (Kotlin hasContractExecutionGrant).
func (b *BaseClient) HasContractExecutionGrant(ctx context.Context, granter, grantee string) (bool, error) {
	list, err := b.ContractExecutionGrants(ctx, granter, grantee)
	if err != nil {
		return false, err
	}
	return len(list) > 0, nil
}

// HasContractCommitmentGrant reports whether this contract may commit funds for granter (Kotlin hasContractCommitmentGrant).
func (b *BaseClient) HasContractCommitmentGrant(ctx context.Context, granter string) (bool, error) {
	list, err := b.ContractCommitmentGrants(ctx, granter)
	if err != nil {
		return false, err
	}
	return len(list) > 0, nil
}
