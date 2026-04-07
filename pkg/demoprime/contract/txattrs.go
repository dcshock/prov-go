package contract

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TxResponseFirstAttribute scans tx.Events then Logs events for the first attribute with the given key.
func TxResponseFirstAttribute(tx *sdk.TxResponse, wantKey string) (value string, ok bool) {
	if tx == nil {
		return "", false
	}
	for _, ev := range tx.Events {
		if v, ok := abciEventFind(ev, wantKey); ok {
			return v, true
		}
	}
	for _, log := range tx.Logs {
		for _, se := range log.Events {
			for _, a := range se.Attributes {
				if a.Key == wantKey {
					return a.Value, true
				}
			}
		}
	}
	return "", false
}

func abciEventFind(ev abci.Event, wantKey string) (string, bool) {
	for _, a := range ev.Attributes {
		if a.Key == wantKey {
			return a.Value, true
		}
	}
	return "", false
}
