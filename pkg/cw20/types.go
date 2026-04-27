// Package cw20 provides CosmWasm CW20 smart-query helpers (cw20-base / cw-plus style),
// matching common provenanced invocations such as token_info, minter, and balance.
package cw20

// TokenInfoResponse is the usual CW20 TokenInfo query response shape.
type TokenInfoResponse struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	TotalSupply string `json:"total_supply"`
}

// MinterResponse is returned by CW20 contracts that implement the mintable extension ({"minter":{}}).
type MinterResponse struct {
	Minter string `json:"minter"`
	Cap    string `json:"cap,omitempty"`
}

// BalanceResponse is the CW20 Balance query response ({"balance":{"address":"..."}}).
type BalanceResponse struct {
	Balance string `json:"balance"`
}
