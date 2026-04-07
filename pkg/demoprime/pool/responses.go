package pool

import (
	"encoding/json"
	"fmt"
	"time"

	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Wasm event attribute keys (matches Kotlin WasmEvent / Responses).
const (
	AttrContractAddress = "_contract_address"
	AttrActionName      = "action"
)

// Attribute action values (pool_v2).
const (
	ActionAddCollateral              = "add_collateral"
	ActionBorrow                     = "borrow"
	ActionLend                       = "lend"
	ActionLiquidate                  = "liquidate"
	ActionRemoveCollateral           = "remove_collateral"
	ActionRepay                      = "repay"
	ActionSetBorrowerRequiredAttrs   = "set_borrower_required_attrs"
	ActionSetLenderRequireCommitExit = "set_lender_require_commit_on_exit"
	ActionSetLenderRequiredAttrs     = "set_lender_required_attrs"
	ActionSetOperationalState      = "set_operational_state"
	ActionTransfer                 = "transfer"
	ActionTransferExact            = "transfer_exact"
	ActionUpdateContractConfig     = "update_contract_config"
	ActionUpdateRateParams         = "update_rate_params"
	ActionUpdateSupportedCollateral = "update_supported_collateral"
	ActionWithdraw                 = "withdraw"
	ActionWithdrawExact            = "withdraw_exact"
	ActionWithdrawReserve          = "withdraw_reserve"
)

// TxMetadata locates a wasm event inside a transaction (Kotlin TxMetadata).
type TxMetadata struct {
	TxHash        string
	TxIndex       int
	TxEventIndex  int
}

// BlockInfo is height and block time for executed txs.
type BlockInfo struct {
	BlockHeight int64
	BlockTime   time.Time
}

// WasmTxEventResponse is embedded metadata on execute responses.
type WasmTxEventResponse struct {
	BlockHeight   int64
	BlockTime     time.Time
	TxHash        string
	TxIndex       int
	TxEventIndex  int
	ContractAddr  string
}

func blockInfoFromBlock(resp *tendermint.GetBlockByHeightResponse) (BlockInfo, error) {
	if resp == nil || resp.SdkBlock == nil || resp.SdkBlock.Header == nil {
		return BlockInfo{}, fmt.Errorf("pool: empty block response")
	}
	h := resp.SdkBlock.Header
	ts := h.Time.AsTime()
	return BlockInfo{BlockHeight: h.Height, BlockTime: ts}, nil
}

func txMetadataFrom(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse, action string) (TxMetadata, map[string]string, BlockInfo, error) {
	if tx == nil {
		return TxMetadata{}, nil, BlockInfo{}, fmt.Errorf("pool: nil tx response")
	}
	bi, err := blockInfoFromBlock(block)
	if err != nil {
		return TxMetadata{}, nil, BlockInfo{}, err
	}
	idx, attrs, err := findWasmActionEvent(tx, action)
	if err != nil {
		return TxMetadata{}, nil, bi, err
	}
	txIx, err := txIndexInBlock(block, tx.TxHash)
	if err != nil {
		return TxMetadata{}, nil, bi, err
	}
	return TxMetadata{
		TxHash:       tx.TxHash,
		TxIndex:      txIx,
		TxEventIndex: idx,
	}, attrs, bi, nil
}

// --- Execute response types ---

type AddCollateralResponseV1 struct {
	WasmTxEventResponse
	Borrower       string
	CollateralJSON string
}

type BorrowResponseV1 struct {
	WasmTxEventResponse
	Borrower      string
	Amount        string
	ScaledAmount  string
}

type LendResponseV1 struct {
	WasmTxEventResponse
	Lender       string
	Amount       string
	ScaledAmount string
}

type LiquidateResponseV1 struct {
	WasmTxEventResponse
	Liquidator     string
	Borrower       string
	Amount         string
	ScaledAmount   string
	CollateralJSON string
}

type RepayResponseV1 struct {
	WasmTxEventResponse
	Borrower     string
	Amount       string
	ScaledAmount string
}

type RemoveCollateralResponseV1 struct {
	WasmTxEventResponse
	Borrower       string
	CollateralJSON string
}

type SetBorrowerRequiredAttrsResponseV1 struct {
	WasmTxEventResponse
	BorrowerRequiredAttrsJSON string
}

type SetLenderRequiredAttrsResponseV1 struct {
	WasmTxEventResponse
	LenderRequiredAttrsJSON string
}

type SetLenderRequireCommitOnExitResponseV1 struct {
	WasmTxEventResponse
	Address string
	Require string // "true", "false", or "default"
}

type SetOperationStateResponseV1 struct {
	WasmTxEventResponse
	State string
}

type UpdateContractConfigResponseV1 struct {
	WasmTxEventResponse
}

type UpdateRateParamsResponseV1 struct {
	WasmTxEventResponse
}

type UpdateSupportedCollateralResponseV1 struct {
	WasmTxEventResponse
	CollateralUpdatedJSON *string
	CollateralRemovedJSON *string
}

type WithdrawResponseV1 struct {
	WasmTxEventResponse
	Lender       string
	Amount       string
	ScaledAmount string
}

type Cw20WithdrawExactResponseV1 struct {
	WasmTxEventResponse
	Lender       string
	Amount       string
	ScaledAmount string
}

type Cw20TransferResponseV1 struct {
	WasmTxEventResponse
	Lender       string
	Recipient    string
	Amount       string
	ScaledAmount string
}

type Cw20TransferExactResponseV1 struct {
	WasmTxEventResponse
	Lender       string
	Recipient    string
	Amount       string
	ScaledAmount string
}

type WithdrawReserveResponseV1 struct {
	WasmTxEventResponse
	Recipient string
	Amount    string
}

// Cw20ReceiveOutcome is one of the receive callback result shapes.
type Cw20ReceiveOutcome interface {
	cw20Outcome()
}

func (WithdrawResponseV1) cw20Outcome()              {}
func (Cw20WithdrawExactResponseV1) cw20Outcome()      {}
func (Cw20TransferResponseV1) cw20Outcome()           {}
func (Cw20TransferExactResponseV1) cw20Outcome()      {}

func parseCollateralMapJSON(s string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func parseAddCollateral(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (AddCollateralResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionAddCollateral)
	if err != nil {
		return AddCollateralResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return AddCollateralResponseV1{
		WasmTxEventResponse: w,
		Borrower:            attrs["borrower"],
		CollateralJSON:      attrs["collateral_json"],
	}, nil
}

func parseBorrow(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (BorrowResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionBorrow)
	if err != nil {
		return BorrowResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return BorrowResponseV1{
		WasmTxEventResponse: w,
		Borrower:            attrs["borrower"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseLend(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (LendResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionLend)
	if err != nil {
		return LendResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return LendResponseV1{
		WasmTxEventResponse: w,
		Lender:              attrs["lender"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseLiquidate(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (LiquidateResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionLiquidate)
	if err != nil {
		return LiquidateResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return LiquidateResponseV1{
		WasmTxEventResponse: w,
		Liquidator:          attrs["liquidator"],
		Borrower:            attrs["borrower"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
		CollateralJSON:      attrs["collateral_json"],
	}, nil
}

func parseRepay(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (RepayResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionRepay)
	if err != nil {
		return RepayResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return RepayResponseV1{
		WasmTxEventResponse: w,
		Borrower:            attrs["borrower"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseRemoveCollateral(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (RemoveCollateralResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionRemoveCollateral)
	if err != nil {
		return RemoveCollateralResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return RemoveCollateralResponseV1{
		WasmTxEventResponse: w,
		Borrower:            attrs["borrower"],
		CollateralJSON:      attrs["collateral_json"],
	}, nil
}

func parseSetBorrowerRequiredAttrs(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (SetBorrowerRequiredAttrsResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionSetBorrowerRequiredAttrs)
	if err != nil {
		return SetBorrowerRequiredAttrsResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return SetBorrowerRequiredAttrsResponseV1{
		WasmTxEventResponse:       w,
		BorrowerRequiredAttrsJSON: attrs["borrower_required_attrs_json"],
	}, nil
}

func parseSetLenderRequiredAttrs(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (SetLenderRequiredAttrsResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionSetLenderRequiredAttrs)
	if err != nil {
		return SetLenderRequiredAttrsResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return SetLenderRequiredAttrsResponseV1{
		WasmTxEventResponse:     w,
		LenderRequiredAttrsJSON: attrs["lender_required_attrs_json"],
	}, nil
}

func parseSetLenderRequireCommitOnExit(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (SetLenderRequireCommitOnExitResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionSetLenderRequireCommitExit)
	if err != nil {
		return SetLenderRequireCommitOnExitResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return SetLenderRequireCommitOnExitResponseV1{
		WasmTxEventResponse: w,
		Address:             attrs["address"],
		Require:             attrs["require"],
	}, nil
}

func parseSetOperationalState(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (SetOperationStateResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionSetOperationalState)
	if err != nil {
		return SetOperationStateResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return SetOperationStateResponseV1{
		WasmTxEventResponse: w,
		State:               attrs["state"],
	}, nil
}

func parseUpdateContractConfig(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (UpdateContractConfigResponseV1, error) {
	txMeta, attrs, bi, err := txMetadataFrom(tx, block, ActionUpdateContractConfig)
	if err != nil {
		return UpdateContractConfigResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: txMeta.TxHash, TxIndex: txMeta.TxIndex, TxEventIndex: txMeta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return UpdateContractConfigResponseV1{WasmTxEventResponse: w}, nil
}

func parseUpdateRateParams(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (UpdateRateParamsResponseV1, error) {
	txMeta, attrs, bi, err := txMetadataFrom(tx, block, ActionUpdateRateParams)
	if err != nil {
		return UpdateRateParamsResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: txMeta.TxHash, TxIndex: txMeta.TxIndex, TxEventIndex: txMeta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return UpdateRateParamsResponseV1{WasmTxEventResponse: w}, nil
}

func parseUpdateSupportedCollateral(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (UpdateSupportedCollateralResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionUpdateSupportedCollateral)
	if err != nil {
		return UpdateSupportedCollateralResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	u := attrs["supported_collateral_updated_json"]
	r := attrs["supported_collateral_removed_json"]
	var up, rm *string
	if u != "" {
		up = &u
	}
	if r != "" {
		rm = &r
	}
	return UpdateSupportedCollateralResponseV1{
		WasmTxEventResponse:   w,
		CollateralUpdatedJSON: up,
		CollateralRemovedJSON: rm,
	}, nil
}

func parseWithdraw(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (WithdrawResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionWithdraw)
	if err != nil {
		return WithdrawResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return WithdrawResponseV1{
		WasmTxEventResponse: w,
		Lender:              attrs["lender"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseWithdrawExact(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (Cw20WithdrawExactResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionWithdrawExact)
	if err != nil {
		return Cw20WithdrawExactResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return Cw20WithdrawExactResponseV1{
		WasmTxEventResponse: w,
		Lender:              attrs["lender"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseTransfer(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (Cw20TransferResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionTransfer)
	if err != nil {
		return Cw20TransferResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return Cw20TransferResponseV1{
		WasmTxEventResponse: w,
		Lender:              attrs["lender"],
		Recipient:           attrs["recipient"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseTransferExact(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (Cw20TransferExactResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionTransferExact)
	if err != nil {
		return Cw20TransferExactResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return Cw20TransferExactResponseV1{
		WasmTxEventResponse: w,
		Lender:              attrs["lender"],
		Recipient:           attrs["recipient"],
		Amount:              attrs["amount"],
		ScaledAmount:        attrs["scaled_amount"],
	}, nil
}

func parseWithdrawReserve(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (WithdrawReserveResponseV1, error) {
	meta, attrs, bi, err := txMetadataFrom(tx, block, ActionWithdrawReserve)
	if err != nil {
		return WithdrawReserveResponseV1{}, err
	}
	w := WasmTxEventResponse{
		BlockHeight: bi.BlockHeight, BlockTime: bi.BlockTime,
		TxHash: meta.TxHash, TxIndex: meta.TxIndex, TxEventIndex: meta.TxEventIndex,
		ContractAddr: attrs[AttrContractAddress],
	}
	return WithdrawReserveResponseV1{
		WasmTxEventResponse: w,
		Recipient:           attrs["recipient"],
		Amount:              attrs["amount"],
	}, nil
}

// ParseCw20ReceiveResponse matches Kotlin toCw20ReceiveResponse (withdraw / withdraw_exact / transfer / transfer_exact).
func ParseCw20ReceiveResponse(tx *sdk.TxResponse, block *tendermint.GetBlockByHeightResponse) (Cw20ReceiveOutcome, error) {
	if tx == nil {
		return nil, fmt.Errorf("pool: nil tx response")
	}
	if v, err := parseWithdraw(tx, block); err == nil {
		return v, nil
	}
	if v, err := parseWithdrawExact(tx, block); err == nil {
		return v, nil
	}
	if v, err := parseTransfer(tx, block); err == nil {
		return v, nil
	}
	if v, err := parseTransferExact(tx, block); err == nil {
		return v, nil
	}
	return nil, fmt.Errorf("pool: cw20 receive: no matching withdraw/transfer action")
}
