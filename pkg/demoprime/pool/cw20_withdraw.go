package pool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// buildCw20SendWithdrawExecuteJSON builds the execute message for the repo CW20 contract:
// {"send":{"contract":"<pool>","amount":"<base units>","msg":"<base64({"withdraw":{"amount":"..."}})>"}}
// This matches the documented provenanced flow for pool withdraw via CW20 send.
func buildCw20SendWithdrawExecuteJSON(poolContract string, withdrawAmount *big.Int) ([]byte, error) {
	if poolContract == "" {
		return nil, fmt.Errorf("pool: empty pool contract for cw20 send")
	}
	if withdrawAmount == nil || withdrawAmount.Sign() <= 0 {
		return nil, fmt.Errorf("pool: withdraw amount must be positive")
	}
	amtStr := withdrawAmount.String()
	inner, err := json.Marshal(map[string]any{
		"withdraw": map[string]string{"amount": amtStr},
	})
	if err != nil {
		return nil, fmt.Errorf("pool: marshal inner withdraw: %w", err)
	}
	outer := map[string]any{
		"send": map[string]string{
			"contract": poolContract,
			"amount":   amtStr,
			"msg":      base64.StdEncoding.EncodeToString(inner),
		},
	}
	return json.Marshal(outer)
}

// WithdrawViaRepoCw20Send withdraws by executing CW20 send on repoCw20Contract, forwarding a withdraw hook to this
// Client's pool contract (c.ContractAddr). Matches: provenanced tx wasm execute $REPO '{"send":{...}}'.
func (c *Client) WithdrawViaRepoCw20Send(ctx context.Context, repoCw20Contract string, withdrawAmount *big.Int) (*WithdrawResponseV1, error) {
	if err := c.RequireSigner(); err != nil {
		return nil, err
	}
	msg, err := buildCw20SendWithdrawExecuteJSON(c.ContractAddr, withdrawAmount)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnContract(ctx, c.Prov.Address, repoCw20Contract, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseWithdraw(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
