package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/dcshock/prov-go/pkg/demoprime/contract"
	"github.com/dcshock/prov-go/pkg/provenance"
)

// DefaultRelativeTimeoutHeight is re-exported from demoprime/contract (Kotlin BaseContractClient.RELATIVE_TIMEOUT_HEIGHT).
const DefaultRelativeTimeoutHeight = contract.DefaultRelativeTimeoutHeight

// Client is the Go analogue of Kotlin PbPoolContractClient (blocking style), embedding BaseContractClient capabilities.
type Client struct {
	*contract.BaseClient
}

// NewClient builds a pool client. prov must be configured with a signer for execute methods.
func NewClient(prov *provenance.ProvenanceClient, contractAddr string) *Client {
	return &Client{BaseClient: contract.NewBaseClient(prov, contractAddr)}
}

func (c *Client) wasmQuery() wasmtypes.QueryClient {
	return wasmtypes.NewQueryClient(c.Prov.Grpc.Conn)
}

func (c *Client) querySmartContract(ctx context.Context, queryJSON []byte) ([]byte, error) {
	resp, err := c.wasmQuery().SmartContractState(ctx, &wasmtypes.QuerySmartContractStateRequest{
		Address:   c.ContractAddr,
		QueryData: queryJSON,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// --- Queries ---

func (c *Client) GetLenderStatus(ctx context.Context, address string) (*LenderStatusResponseV1, error) {
	bz, err := queryGetLenderStatus(address)
	if err != nil {
		return nil, err
	}
	raw, err := c.querySmartContract(ctx, bz)
	if err != nil {
		return nil, err
	}
	var out LenderStatusResponseV1
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetState(ctx context.Context) (*StateResponseV1, error) {
	bz, err := queryGetState()
	if err != nil {
		return nil, err
	}
	raw, err := c.querySmartContract(ctx, bz)
	if err != nil {
		return nil, err
	}
	var out StateResponseV1
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetReserve(ctx context.Context) (*ReserveResponseV1, error) {
	bz, err := queryGetReserve()
	if err != nil {
		return nil, err
	}
	raw, err := c.querySmartContract(ctx, bz)
	if err != nil {
		return nil, err
	}
	var out ReserveResponseV1
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBorrowerPosition(ctx context.Context, address string) (*BorrowerPositionResponseV1, error) {
	bz, err := queryGetBorrowerPosition(address)
	if err != nil {
		return nil, err
	}
	raw, err := c.querySmartContract(ctx, bz)
	if err != nil {
		return nil, err
	}
	var out BorrowerPositionResponseV1
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCollateralRequirements(ctx context.Context, borrower *string, collateralAssets []string, newLoanAmount *big.Int) (*CollateralRequirementsResponseV1, error) {
	bz, err := queryGetCollateralRequirements(borrower, collateralAssets, newLoanAmount)
	if err != nil {
		return nil, err
	}
	raw, err := c.querySmartContract(ctx, bz)
	if err != nil {
		return nil, err
	}
	var out CollateralRequirementsResponseV1
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) fetchBlock(ctx context.Context, height int64) (*tendermint.GetBlockByHeightResponse, error) {
	tc := c.Prov.TendermintClient()
	return (*tc).GetBlockByHeight(ctx, &tendermint.GetBlockByHeightRequest{Height: height})
}

func (c *Client) broadcastAndWait(ctx context.Context, msgs []sdk.Msg) (*sdk.TxResponse, *tendermint.GetBlockByHeightResponse, error) {
	if err := c.RequireSigner(); err != nil {
		return nil, nil, err
	}
	seq := c.Prov.NextSequence()
	txBz, err := c.Prov.SignTx(msgs, c.Prov.PrivKey.Bytes(), c.Prov.AccountNumber, seq, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("pool: sign tx: %w", err)
	}
	bcast, err := c.Prov.BroadcastTx(txBz)
	if err != nil {
		return nil, nil, fmt.Errorf("pool: broadcast: %w", err)
	}
	if bcast.TxResponse == nil || bcast.TxResponse.TxHash == "" {
		return nil, nil, fmt.Errorf("pool: empty broadcast response")
	}
	got, err := c.Prov.WaitOnTx(bcast.TxResponse.TxHash)
	if err != nil {
		return nil, nil, fmt.Errorf("pool: wait on tx: %w", err)
	}
	txr := got.TxResponse
	if txr == nil {
		return nil, nil, fmt.Errorf("pool: nil tx in get-tx response")
	}
	if txr.Code != 0 {
		return txr, nil, fmt.Errorf("pool: tx failed code=%d codespace=%s raw_log=%s", txr.Code, txr.Codespace, txr.RawLog)
	}
	block, err := c.fetchBlock(ctx, txr.Height)
	if err != nil {
		return txr, nil, fmt.Errorf("pool: get block %d: %w", txr.Height, err)
	}
	return txr, block, nil
}

func (c *Client) execute(ctx context.Context, sender string, msgJSON []byte, funds sdk.Coins) (*sdk.TxResponse, *tendermint.GetBlockByHeightResponse, error) {
	return c.executeOnContract(ctx, sender, c.ContractAddr, msgJSON, funds)
}

// executeOnContract broadcasts MsgExecuteContract on an arbitrary CosmWasm address (pool, CW20 repo, etc.).
func (c *Client) executeOnContract(ctx context.Context, sender, contractAddr string, msgJSON []byte, funds sdk.Coins) (*sdk.TxResponse, *tendermint.GetBlockByHeightResponse, error) {
	exec := &wasmtypes.MsgExecuteContract{
		Sender:   sender,
		Contract: contractAddr,
		Msg:      wasmtypes.RawContractMessage(msgJSON),
		Funds:    funds,
	}
	return c.broadcastAndWait(ctx, []sdk.Msg{exec})
}

func (c *Client) executeOnBehalfOf(ctx context.Context, granterAddress string, msgJSON []byte, funds sdk.Coins) (*sdk.TxResponse, *tendermint.GetBlockByHeightResponse, error) {
	if err := c.RequireSigner(); err != nil {
		return nil, nil, err
	}
	inner := &wasmtypes.MsgExecuteContract{
		Sender:   granterAddress,
		Contract: c.ContractAddr,
		Msg:      wasmtypes.RawContractMessage(msgJSON),
		Funds:    funds,
	}
	grantee := sdk.MustAccAddressFromBech32(c.Prov.Address)
	exec := authz.NewMsgExec(grantee, []sdk.Msg{inner})
	return c.broadcastAndWait(ctx, []sdk.Msg{&exec})
}

// --- Executes (direct) ---

func (c *Client) Lend(ctx context.Context, funds sdk.Coins) (*LendResponseV1, error) {
	msg, err := execLend()
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, funds)
	if err != nil {
		return nil, err
	}
	v, err := parseLend(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) Receive(ctx context.Context, msg Cw20ReceiveRequestV1) (Cw20ReceiveOutcome, error) {
	raw, err := execReceive(msg)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, raw, nil)
	if err != nil {
		return nil, err
	}
	return ParseCw20ReceiveResponse(txr, block)
}

func (c *Client) Borrow(ctx context.Context, amount *big.Int) (*BorrowResponseV1, error) {
	msg, err := execBorrow(amount)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseBorrow(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) Repay(ctx context.Context, funds sdk.Coins) (*RepayResponseV1, error) {
	msg, err := execRepay()
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, funds)
	if err != nil {
		return nil, err
	}
	v, err := parseRepay(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) AddCollateral(ctx context.Context, funds sdk.Coins) (*AddCollateralResponseV1, error) {
	msg, err := execAddCollateral()
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, funds)
	if err != nil {
		return nil, err
	}
	v, err := parseAddCollateral(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) RemoveCollateral(ctx context.Context, toRemove map[string]*big.Int) (*RemoveCollateralResponseV1, error) {
	msg, err := execRemoveCollateral(toRemove)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseRemoveCollateral(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) Liquidate(ctx context.Context, borrower string, collateralToSeize map[string]*big.Int, repayFunds sdk.Coins) (*LiquidateResponseV1, error) {
	msg, err := execLiquidate(borrower, collateralToSeize)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, repayFunds)
	if err != nil {
		return nil, err
	}
	v, err := parseLiquidate(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) UpdateSupportedCollateral(ctx context.Context, toRemove []string, toUpdate []CollateralAssetV1) (*UpdateSupportedCollateralResponseV1, error) {
	msg, err := execUpdateSupportedCollateral(toRemove, toUpdate)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseUpdateSupportedCollateral(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) SetOperationalState(ctx context.Context, state OperationalStateV1) (*SetOperationStateResponseV1, error) {
	msg, err := execSetOperationalState(state)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseSetOperationalState(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) SetBorrowRequiredAttrs(ctx context.Context, attrs []string) (*SetBorrowerRequiredAttrsResponseV1, error) {
	msg, err := execSetBorrowerRequiredAttrs(attrs)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseSetBorrowerRequiredAttrs(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) SetLenderRequiredAttrs(ctx context.Context, attrs []string) (*SetLenderRequiredAttrsResponseV1, error) {
	msg, err := execSetLenderRequiredAttrs(attrs)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseSetLenderRequiredAttrs(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) SetLenderRequireCommitOnExit(ctx context.Context, address string, require *bool) (*SetLenderRequireCommitOnExitResponseV1, error) {
	msg, err := execSetLenderRequireCommitOnExit(address, require)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseSetLenderRequireCommitOnExit(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) UpdateContractConfig(ctx context.Context, liquidationBonusRate, liquidationRate, marginRate *string, maxBorrowerCollateralTypes *int, minBorrow, minLend *big.Int, priceOracleAddress *string) (*UpdateContractConfigResponseV1, error) {
	msg, err := execUpdateContractConfig(liquidationBonusRate, liquidationRate, marginRate, maxBorrowerCollateralTypes, minBorrow, minLend, priceOracleAddress)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseUpdateContractConfig(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) UpdateRateParams(ctx context.Context, rateParams RateParamsV1) (*UpdateRateParamsResponseV1, error) {
	msg, err := execUpdateRateParams(rateParams)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseUpdateRateParams(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) Withdraw(ctx context.Context, lender string, amount *big.Int) (*WithdrawResponseV1, error) {
	msg, err := execWithdraw(lender, amount)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseWithdraw(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) WithdrawReserve(ctx context.Context, recipient *string) (*WithdrawReserveResponseV1, error) {
	msg, err := execWithdrawReserve(recipient)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.execute(ctx, c.Prov.Address, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseWithdrawReserve(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// --- Executes (authz / on behalf of) ---

func (c *Client) LendFor(ctx context.Context, granterAddress string, funds sdk.Coins) (*LendResponseV1, error) {
	msg, err := execLend()
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, funds)
	if err != nil {
		return nil, err
	}
	v, err := parseLend(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) ReceiveFor(ctx context.Context, granterAddress string, msg Cw20ReceiveRequestV1) (Cw20ReceiveOutcome, error) {
	raw, err := execReceive(msg)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, raw, nil)
	if err != nil {
		return nil, err
	}
	return ParseCw20ReceiveResponse(txr, block)
}

func (c *Client) BorrowFor(ctx context.Context, granterAddress string, amount *big.Int) (*BorrowResponseV1, error) {
	msg, err := execBorrow(amount)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseBorrow(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) RepayFor(ctx context.Context, granterAddress string, funds sdk.Coins) (*RepayResponseV1, error) {
	msg, err := execRepay()
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, funds)
	if err != nil {
		return nil, err
	}
	v, err := parseRepay(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) AddCollateralFor(ctx context.Context, granterAddress string, funds sdk.Coins) (*AddCollateralResponseV1, error) {
	msg, err := execAddCollateral()
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, funds)
	if err != nil {
		return nil, err
	}
	v, err := parseAddCollateral(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) RemoveCollateralFor(ctx context.Context, granterAddress string, toRemove map[string]*big.Int) (*RemoveCollateralResponseV1, error) {
	msg, err := execRemoveCollateral(toRemove)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseRemoveCollateral(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) WithdrawFor(ctx context.Context, granterAddress, lender string, amount *big.Int) (*WithdrawResponseV1, error) {
	msg, err := execWithdraw(lender, amount)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseWithdraw(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *Client) WithdrawReserveFor(ctx context.Context, granterAddress string, recipient *string) (*WithdrawReserveResponseV1, error) {
	msg, err := execWithdrawReserve(recipient)
	if err != nil {
		return nil, err
	}
	txr, block, err := c.executeOnBehalfOf(ctx, granterAddress, msg, nil)
	if err != nil {
		return nil, err
	}
	v, err := parseWithdrawReserve(txr, block)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
