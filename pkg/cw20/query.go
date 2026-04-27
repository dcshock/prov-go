package cw20

import (
	"context"
	"encoding/json"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"google.golang.org/grpc"
)

// QueryClient runs wasm smart queries against a single CW20 contract address.
type QueryClient struct {
	wasm wasmtypes.QueryClient
	addr string
}

// NewQueryClient builds a CW20 query client. conn is typically provenance.ProvenanceClient.Grpc.Conn.
func NewQueryClient(conn grpc.ClientConnInterface, cw20ContractAddr string) *QueryClient {
	return &QueryClient{
		wasm: wasmtypes.NewQueryClient(conn),
		addr: cw20ContractAddr,
	}
}

func (c *QueryClient) smartQuery(ctx context.Context, queryJSON []byte) ([]byte, error) {
	if c.addr == "" {
		return nil, fmt.Errorf("cw20: empty contract address")
	}
	resp, err := c.wasm.SmartContractState(ctx, &wasmtypes.QuerySmartContractStateRequest{
		Address:   c.addr,
		QueryData: queryJSON,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// TokenInfo runs {"token_info":{}}.
func (c *QueryClient) TokenInfo(ctx context.Context) (*TokenInfoResponse, error) {
	q, err := json.Marshal(map[string]json.RawMessage{"token_info": json.RawMessage(`{}`)})
	if err != nil {
		return nil, err
	}
	raw, err := c.smartQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	var out TokenInfoResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("cw20: token_info unmarshal: %w", err)
	}
	return &out, nil
}

// Minter runs {"minter":{}} (mintable extension; contract may error if unsupported).
func (c *QueryClient) Minter(ctx context.Context) (*MinterResponse, error) {
	q, err := json.Marshal(map[string]json.RawMessage{"minter": json.RawMessage(`{}`)})
	if err != nil {
		return nil, err
	}
	raw, err := c.smartQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	var out MinterResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("cw20: minter unmarshal: %w", err)
	}
	return &out, nil
}

// Balance runs {"balance":{"address":"<addr>"}}.
func (c *QueryClient) Balance(ctx context.Context, holderBech32 string) (*BalanceResponse, error) {
	if holderBech32 == "" {
		return nil, fmt.Errorf("cw20: empty holder address")
	}
	q, err := json.Marshal(map[string]any{
		"balance": map[string]string{"address": holderBech32},
	})
	if err != nil {
		return nil, err
	}
	raw, err := c.smartQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	var out BalanceResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("cw20: balance unmarshal: %w", err)
	}
	return &out, nil
}
