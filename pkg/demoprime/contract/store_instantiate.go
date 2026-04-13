package contract

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"strconv"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GzipWasm optionally gzip-compresses bytecode for MsgStoreCode (Kotlin StoreMsgProvider compress flag).
func GzipWasm(bytecode []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(bytecode); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StoreWasm uploads WASM bytecode and returns the new code id (Kotlin storeWasm).
func (b *BaseClient) StoreWasm(ctx context.Context, bytecode []byte, compress bool) (uint64, error) {
	if err := b.RequireSigner(); err != nil {
		return 0, err
	}
	payload := bytecode
	if compress {
		var err error
		payload, err = GzipWasm(bytecode)
		if err != nil {
			return 0, fmt.Errorf("demoprime/contract: gzip wasm: %w", err)
		}
	}
	msg := &wasmtypes.MsgStoreCode{
		Sender:       b.PrimarySignerAddress(),
		WASMByteCode: payload,
	}
	txr, err := b.BroadcastMsgs(ctx, []sdk.Msg{msg})
	if err != nil {
		return 0, err
	}
	raw, ok := TxResponseFirstAttribute(txr, "code_id")
	if !ok {
		return 0, fmt.Errorf("demoprime/contract: no code_id in tx response")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("demoprime/contract: parse code_id %q: %w", raw, err)
	}
	return id, nil
}

// InstantiateContract creates a contract instance and returns its address (Kotlin PbClient.instantiateContract).
// Admin, if empty, defaults to the sender (primary signer).
func (b *BaseClient) InstantiateContract(ctx context.Context, codeID uint64, initMsgJSON []byte, label string, admin string, funds sdk.Coins) (contractAddress string, err error) {
	if err := b.RequireSigner(); err != nil {
		return "", err
	}
	sender := b.PrimarySignerAddress()
	if admin == "" {
		admin = sender
	}
	msg := &wasmtypes.MsgInstantiateContract{
		Sender: sender,
		Admin:  admin,
		CodeID: codeID,
		Label:  label,
		Msg:    wasmtypes.RawContractMessage(initMsgJSON),
		Funds:  funds,
	}
	txr, err := b.BroadcastMsgs(ctx, []sdk.Msg{msg})
	if err != nil {
		return "", err
	}
	addr, ok := TxResponseFirstAttribute(txr, "_contract_address")
	if !ok {
		return "", fmt.Errorf("demoprime/contract: no _contract_address in tx response")
	}
	return addr, nil
}
