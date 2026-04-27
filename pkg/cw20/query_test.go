package cw20

import (
	"encoding/json"
	"testing"
)

func TestQueryPayloadsMatchCLI(t *testing.T) {
	t.Parallel()
	ti, err := json.Marshal(map[string]json.RawMessage{"token_info": json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if string(ti) != `{"token_info":{}}` {
		t.Fatalf("token_info: %s", ti)
	}
	m, err := json.Marshal(map[string]json.RawMessage{"minter": json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if string(m) != `{"minter":{}}` {
		t.Fatalf("minter: %s", m)
	}
	b, err := json.Marshal(map[string]any{"balance": map[string]string{"address": "tp1abc"}})
	if err != nil {
		t.Fatal(err)
	}
	var balWrap struct {
		Balance struct {
			Address string `json:"address"`
		} `json:"balance"`
	}
	if err := json.Unmarshal(b, &balWrap); err != nil {
		t.Fatal(err)
	}
	if balWrap.Balance.Address != "tp1abc" {
		t.Fatalf("balance address: %s", string(b))
	}
}
