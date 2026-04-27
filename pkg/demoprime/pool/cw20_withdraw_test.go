package pool

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
)

func TestBuildCw20SendWithdrawExecuteJSON(t *testing.T) {
	t.Parallel()
	pool := "tp1pool0000000000000000000000000000000000"
	amt := big.NewInt(10000000)
	raw, err := buildCw20SendWithdrawExecuteJSON(pool, amt)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatal(err)
	}
	sendRaw, ok := top["send"]
	if !ok {
		t.Fatalf("missing send: %s", string(raw))
	}
	var send struct {
		Contract string `json:"contract"`
		Amount   string `json:"amount"`
		Msg      string `json:"msg"`
	}
	if err := json.Unmarshal(sendRaw, &send); err != nil {
		t.Fatal(err)
	}
	if send.Contract != pool || send.Amount != "10000000" {
		t.Fatalf("send contract/amount: %+v", send)
	}
	innerB, err := base64.StdEncoding.DecodeString(send.Msg)
	if err != nil {
		t.Fatal(err)
	}
	var inner map[string]map[string]string
	if err := json.Unmarshal(innerB, &inner); err != nil {
		t.Fatal(err)
	}
	if inner["withdraw"]["amount"] != "10000000" {
		t.Fatalf("inner withdraw: %s", string(innerB))
	}
}
