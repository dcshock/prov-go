package pool

import (
	"encoding/hex"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	tmhash "github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
)

func abciEventAttrs(ev abci.Event) map[string]string {
	m := make(map[string]string, len(ev.Attributes))
	for _, a := range ev.Attributes {
		m[a.Key] = a.Value
	}
	return m
}

func stringEventAttrs(ev sdk.StringEvent) map[string]string {
	m := make(map[string]string, len(ev.Attributes))
	for _, a := range ev.Attributes {
		m[a.Key] = a.Value
	}
	return m
}

// flatEventAttrs returns attribute maps in the same aggregate order as typical TxResponse indexing:
// top-level Events first, then Events nested under Logs (message events).
func flatEventAttrs(tx *sdk.TxResponse) []map[string]string {
	if tx == nil {
		return nil
	}
	var out []map[string]string
	for _, ev := range tx.Events {
		out = append(out, abciEventAttrs(ev))
	}
	for _, log := range tx.Logs {
		for _, ev := range log.Events {
			out = append(out, stringEventAttrs(ev))
		}
	}
	return out
}

func findWasmActionEvent(tx *sdk.TxResponse, action string) (int, map[string]string, error) {
	list := flatEventAttrs(tx)
	for i, attrs := range list {
		if attrs[AttrActionName] == action {
			return i, attrs, nil
		}
	}
	return -1, nil, fmt.Errorf("pool: no event with action %q", action)
}

func txIndexInBlock(block *tendermint.GetBlockByHeightResponse, txHash string) (int, error) {
	if block == nil || block.SdkBlock == nil || block.SdkBlock.Data == nil {
		return -1, fmt.Errorf("pool: block has no tx data")
	}
	txs := block.SdkBlock.Data.Txs
	want := strings.ToUpper(strings.TrimSpace(txHash))
	for i, raw := range txs {
		h := strings.ToUpper(hex.EncodeToString(tmhash.Sum(raw)))
		if h == want {
			return i, nil
		}
	}
	return -1, fmt.Errorf("pool: tx hash %s not found in block txs", txHash)
}
