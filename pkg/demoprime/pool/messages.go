// Package pool provides a Go client for the democratized-prime pool_v2 CosmWasm contract,
// analogous to lending-services PbPoolContractClient.
package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// ExecuteRequestV1 is the JSON body for MsgExecuteContract (one variant set).
// Field names match the Kotlin ExecuteRequestV1 / pool contract schema.
type ExecuteRequestV1 struct {
	Lend                       json.RawMessage `json:"lend,omitempty"`
	Receive                    json.RawMessage `json:"receive,omitempty"`
	Borrow                     json.RawMessage `json:"borrow,omitempty"`
	Repay                      json.RawMessage `json:"repay,omitempty"`
	AddCollateral              json.RawMessage `json:"add_collateral,omitempty"`
	RemoveCollateral           json.RawMessage `json:"remove_collateral,omitempty"`
	Liquidate                  json.RawMessage `json:"liquidate,omitempty"`
	SetOperationalState        json.RawMessage `json:"set_operational_state,omitempty"`
	SetLenderRequiredAttrs     json.RawMessage `json:"set_lender_required_attrs,omitempty"`
	SetBorrowerRequiredAttrs   json.RawMessage `json:"set_borrower_required_attrs,omitempty"`
	SetLenderRequireCommitExit json.RawMessage `json:"set_lender_require_commit_on_exit,omitempty"`
	UpdateContractConfig       json.RawMessage `json:"update_contract_config,omitempty"`
	UpdateSupportedCollateral  json.RawMessage `json:"update_supported_collateral,omitempty"`
	UpdateRateParams           json.RawMessage `json:"update_rate_params,omitempty"`
	Withdraw                   json.RawMessage `json:"withdraw,omitempty"`
	WithdrawReserve            json.RawMessage `json:"withdraw_reserve,omitempty"`
}

// QueryRequestV1 is the JSON payload for SmartContractState queries.
type QueryRequestV1 struct {
	GetLenderStatus           json.RawMessage `json:"get_lender_status,omitempty"`
	GetState                  json.RawMessage `json:"get_state,omitempty"`
	GetReserve                json.RawMessage `json:"get_reserve,omitempty"`
	GetBorrowerPosition       json.RawMessage `json:"get_borrower_position,omitempty"`
	GetCollateralRequirements json.RawMessage `json:"get_collateral_requirements,omitempty"`
}

// --- Execute sub-messages (JSON fragments) ---

type borrowRequest struct {
	Amount string `json:"amount"`
}

type removeCollateralRequest struct {
	ToRemove map[string]string `json:"to_remove"`
}

type liquidateRequest struct {
	Borrower          string            `json:"borrower"`
	CollateralToSeize map[string]string `json:"collateral_to_seize"`
}

type updateSupportedCollateralRequest struct {
	ToRemove []string           `json:"to_remove"`
	ToUpdate []CollateralAssetV1 `json:"to_update"`
}

type setOperationalStateRequest struct {
	State OperationalStateV1 `json:"state"`
}

type setBorrowerRequiredAttrsRequest struct {
	BorrowerRequiredAttrs []string `json:"borrower_required_attrs"`
}

type setLenderRequiredAttrsRequest struct {
	LenderRequiredAttrs []string `json:"lender_required_attrs"`
}

type setLenderRequireCommitOnExitRequest struct {
	Address string `json:"address"`
	Require *bool  `json:"require,omitempty"`
}

type updateContractConfigRequest struct {
	LiquidationBonusRate       *string `json:"liquidation_bonus_rate,omitempty"`
	LiquidationRate            *string `json:"liquidation_rate,omitempty"`
	MarginRate                 *string `json:"margin_rate,omitempty"`
	MaxBorrowerCollateralTypes *int    `json:"max_borrower_collateral_types,omitempty"`
	MinBorrow                  *string `json:"min_borrow,omitempty"`
	MinLend                    *string `json:"min_lend,omitempty"`
	PriceOracleAddress         *string `json:"price_oracle_address,omitempty"`
}

type updateRateParamsRequest struct {
	RateParams RateParamsV1 `json:"rate_params"`
}

type withdrawRequest struct {
	Lender      string  `json:"lender"`
	Amount      *string `json:"amount,omitempty"`
	CommitFunds *bool   `json:"commit_funds,omitempty"`
}

type withdrawReserveRequest struct {
	Recipient *string `json:"recipient,omitempty"`
}

// CollateralAssetV1 matches pool supported-collateral entries.
type CollateralAssetV1 struct {
	ID      string  `json:"id"`
	Haircut *string `json:"h,omitempty"`
}

// RateParamsV1 is the kink rate model payload.
type RateParamsV1 struct {
	TargetRate       string `json:"tr"`
	MinRate          string `json:"minr"`
	MaxRate          string `json:"maxr"`
	KinkUtilization  string `json:"kink"`
	ReserveFactor    string `json:"rf"`
	SecondsPerYear   int64  `json:"spy"`
}

// OperationalStateV1 values: active, frozen, paused.
type OperationalStateV1 string

const (
	OperationalActive  OperationalStateV1 = "active"
	OperationalFrozen  OperationalStateV1 = "frozen"
	OperationalPaused  OperationalStateV1 = "paused"
)

// PoolReceiveRequestMsg is the inner JSON for CW20 receive callbacks (before base64).
type PoolReceiveRequestMsg struct {
	Withdraw       *PoolWithdrawPayloadV1       `json:"withdraw,omitempty"`
	WithdrawExact  *PoolWithdrawExactPayloadV1  `json:"withdraw_exact,omitempty"`
	Transfer       *PoolTransferPayloadV1       `json:"transfer,omitempty"`
	TransferExact  *PoolTransferExactPayloadV1  `json:"transfer_exact,omitempty"`
}

type PoolWithdrawPayloadV1 struct {
	Amount      string `json:"amount"`
	CommitFunds *bool  `json:"commit_funds,omitempty"`
}

type PoolWithdrawExactPayloadV1 struct {
	CommitFunds *bool `json:"commit_funds,omitempty"`
}

type PoolTransferPayloadV1 struct {
	Recipient string `json:"recipient"`
	Amount    string `json:"amount"`
}

type PoolTransferExactPayloadV1 struct {
	Recipient string `json:"recipient"`
}

// Cw20ReceiveRequestV1 is the JSON for execute.receive.
type Cw20ReceiveRequestV1 struct {
	Sender string `json:"sender"`
	Amount string `json:"amount"`
	Msg    string `json:"msg"` // base64-encoded JSON of PoolReceiveRequestMsg
}

// NewCw20ReceiveRequest builds a receive payload with msg as standard base64(JSON(inner)).
func NewCw20ReceiveRequest(sender string, amount *big.Int, inner PoolReceiveRequestMsg) (Cw20ReceiveRequestV1, error) {
	innerBz, err := json.Marshal(inner)
	if err != nil {
		return Cw20ReceiveRequestV1{}, err
	}
	return Cw20ReceiveRequestV1{
		Sender: sender,
		Amount: amount.String(),
		Msg:    base64.StdEncoding.EncodeToString(innerBz),
	}, nil
}

// --- Query sub-messages ---

type getLenderStatusQuery struct {
	Address string `json:"address"`
}

type getBorrowerPositionQuery struct {
	Address string `json:"address"`
}

type getCollateralRequirementsQuery struct {
	Borrower         *string  `json:"borrower,omitempty"`
	CollateralAssets []string `json:"collateral_assets"`
	NewLoanAmount    string   `json:"new_loan_amount"`
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func bigString(n *big.Int) string {
	if n == nil {
		return "0"
	}
	return n.String()
}

// BuildExecute marshals a single-variant ExecuteRequestV1.
func buildExecute(patch func(*ExecuteRequestV1)) ([]byte, error) {
	var e ExecuteRequestV1
	patch(&e)
	return json.Marshal(e)
}

func execLend() ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) { e.Lend = emptyObjectJSON() })
}

func execRepay() ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) { e.Repay = emptyObjectJSON() })
}

func execAddCollateral() ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) { e.AddCollateral = emptyObjectJSON() })
}

func execReceive(r Cw20ReceiveRequestV1) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) { e.Receive = mustJSON(r) })
}

func execBorrow(amount *big.Int) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.Borrow = mustJSON(borrowRequest{Amount: bigString(amount)})
	})
}

func execRemoveCollateral(toRemove map[string]*big.Int) ([]byte, error) {
	m := make(map[string]string, len(toRemove))
	for k, v := range toRemove {
		m[k] = bigString(v)
	}
	return buildExecute(func(e *ExecuteRequestV1) {
		e.RemoveCollateral = mustJSON(removeCollateralRequest{ToRemove: m})
	})
}

func execLiquidate(borrower string, collateralToSeize map[string]*big.Int) ([]byte, error) {
	m := make(map[string]string, len(collateralToSeize))
	for k, v := range collateralToSeize {
		m[k] = bigString(v)
	}
	return buildExecute(func(e *ExecuteRequestV1) {
		e.Liquidate = mustJSON(liquidateRequest{Borrower: borrower, CollateralToSeize: m})
	})
}

func execUpdateSupportedCollateral(toRemove []string, toUpdate []CollateralAssetV1) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.UpdateSupportedCollateral = mustJSON(updateSupportedCollateralRequest{ToRemove: toRemove, ToUpdate: toUpdate})
	})
}

func execSetOperationalState(state OperationalStateV1) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.SetOperationalState = mustJSON(setOperationalStateRequest{State: state})
	})
}

func execSetBorrowerRequiredAttrs(attrs []string) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.SetBorrowerRequiredAttrs = mustJSON(setBorrowerRequiredAttrsRequest{BorrowerRequiredAttrs: attrs})
	})
}

func execSetLenderRequiredAttrs(attrs []string) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.SetLenderRequiredAttrs = mustJSON(setLenderRequiredAttrsRequest{LenderRequiredAttrs: attrs})
	})
}

func execSetLenderRequireCommitOnExit(address string, require *bool) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.SetLenderRequireCommitExit = mustJSON(setLenderRequireCommitOnExitRequest{Address: address, Require: require})
	})
}

func ptrPlainString(s string) *string { return &s }

func execUpdateContractConfig(liquidationBonusRate, liquidationRate, marginRate *string, maxBorrowerCollateralTypes *int, minBorrow, minLend *big.Int, priceOracleAddress *string) ([]byte, error) {
	var minB, minL *string
	if minBorrow != nil {
		minB = ptrPlainString(minBorrow.String())
	}
	if minLend != nil {
		minL = ptrPlainString(minLend.String())
	}
	return buildExecute(func(e *ExecuteRequestV1) {
		e.UpdateContractConfig = mustJSON(updateContractConfigRequest{
			LiquidationBonusRate:       liquidationBonusRate,
			LiquidationRate:            liquidationRate,
			MarginRate:                 marginRate,
			MaxBorrowerCollateralTypes: maxBorrowerCollateralTypes,
			MinBorrow:                  minB,
			MinLend:                    minL,
			PriceOracleAddress:         priceOracleAddress,
		})
	})
}

func execUpdateRateParams(rp RateParamsV1) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.UpdateRateParams = mustJSON(updateRateParamsRequest{RateParams: rp})
	})
}

func execWithdraw(lender string, amount *big.Int) ([]byte, error) {
	var amt *string
	if amount != nil {
		amt = ptrPlainString(amount.String())
	}
	return buildExecute(func(e *ExecuteRequestV1) {
		e.Withdraw = mustJSON(withdrawRequest{Lender: lender, Amount: amt})
	})
}

func execWithdrawReserve(recipient *string) ([]byte, error) {
	return buildExecute(func(e *ExecuteRequestV1) {
		e.WithdrawReserve = mustJSON(withdrawReserveRequest{Recipient: recipient})
	})
}

func emptyObjectJSON() json.RawMessage { return json.RawMessage(`{}`) }

func queryGetLenderStatus(address string) ([]byte, error) {
	q := QueryRequestV1{GetLenderStatus: mustJSON(getLenderStatusQuery{Address: address})}
	return json.Marshal(q)
}

func queryGetState() ([]byte, error) {
	q := QueryRequestV1{GetState: emptyObjectJSON()}
	return json.Marshal(q)
}

func queryGetReserve() ([]byte, error) {
	q := QueryRequestV1{GetReserve: emptyObjectJSON()}
	return json.Marshal(q)
}

func queryGetBorrowerPosition(address string) ([]byte, error) {
	q := QueryRequestV1{GetBorrowerPosition: mustJSON(getBorrowerPositionQuery{Address: address})}
	return json.Marshal(q)
}

func queryGetCollateralRequirements(borrower *string, collateralAssets []string, newLoanAmount *big.Int) ([]byte, error) {
	if newLoanAmount == nil {
		return nil, fmt.Errorf("newLoanAmount required")
	}
	q := QueryRequestV1{
		GetCollateralRequirements: mustJSON(getCollateralRequirementsQuery{
			Borrower:         borrower,
			CollateralAssets: collateralAssets,
			NewLoanAmount:    newLoanAmount.String(),
		}),
	}
	return json.Marshal(q)
}
