package pool

// Denom matches pool contract denom JSON (short keys).
type Denom struct {
	Name      string `json:"n"`
	Precision int    `json:"p"`
}

// AssetRequirementV1 is a collateral line in query responses.
type AssetRequirementV1 struct {
	AssetID string `json:"asset_id"`
	Amount  string `json:"amount"`
}

// PoolContractStateV1 is the nested "contract" object in get_state.
type PoolContractStateV1 struct {
	Admin                    string               `json:"admin"`
	ContractName             string               `json:"c_n"`
	Description              string               `json:"d"`
	RepoTokenCw20Address     string               `json:"atca"`
	LendingDenom             Denom                `json:"ld"`
	RateParams               RateParamsV1         `json:"rp"`
	LenderRequiredAttrs      []string             `json:"lra"`
	BorrowRequiredAttrs      []string             `json:"bra"`
	PriceOracleAddress       string               `json:"poa"`
	MaxBorrowerCollateralTypes int                `json:"max_borrower_collateral_types"`
	MarginRate               string               `json:"mr"`
	LiquidationRate          string               `json:"lr"`
	LiquidationBonusRate     string               `json:"lbr"`
	MinLend                  string               `json:"min_lend"`
	MinBorrow                string               `json:"min_borrow"`
	SupportedCollateralAssets []CollateralAssetV1 `json:"sca"`
	OperationalState         OperationalStateV1   `json:"op"`
}

// ReserveStateResponseV1 is reserve totals in queries.
type ReserveStateResponseV1 struct {
	LiquidityIndex      string `json:"liquidity_index"`
	BorrowIndex         string `json:"borrow_index"`
	LastUpdatedAt       string `json:"last_updated_at"`
	TotalScaledLiquidity string `json:"total_scaled_liquidity"`
	TotalScaledBorrow    string `json:"total_scaled_borrow"`
	AccruedReserve       string `json:"accrued_reserve"`
	TotalLiquidity       string `json:"total_liquidity"`
	TotalBorrow          string `json:"total_borrow"`
}

// StateResponseV1 is get_state query output.
type StateResponseV1 struct {
	Contract            PoolContractStateV1   `json:"contract"`
	Reserve             ReserveStateResponseV1 `json:"reserve"`
	SupportedCollateral []CollateralAssetV1   `json:"supported_collateral"`
	TotalCollateralHeld []AssetRequirementV1  `json:"total_collateral_held"`
}

// LenderStatusResponseV1 is get_lender_status output.
type LenderStatusResponseV1 struct {
	RequireCommitOnExit bool `json:"require_commit_on_exit"`
}

// ReserveResponseV1 is get_reserve output.
type ReserveResponseV1 struct {
	Reserve           ReserveStateResponseV1 `json:"reserve"`
	CurrentBorrowerRate string               `json:"current_borrower_rate"`
	CurrentLenderRate   string               `json:"current_lender_rate"`
	Utilization         string               `json:"utilization"`
}

// BorrowerPositionResponseV1 is get_borrower_position output.
type BorrowerPositionResponseV1 struct {
	Address               string                 `json:"address"`
	ScaledBorrow          string                 `json:"scaled_borrow"`
	UnderlyingDebt        string                 `json:"underlying_debt"`
	UnderlyingDebtDisplay string                 `json:"underlying_debt_display"`
	LendingDenom          Denom                  `json:"lending_denom"`
	Collateral            []AssetRequirementV1   `json:"collateral"`
	CollateralValueUsd    string                 `json:"collateral_value_usd"`
	LoanToValue           string                 `json:"loan_to_value"`
	Health                string                 `json:"health"`
	HealthUnknownReason   *string                `json:"health_unknown_reason,omitempty"`
}

// CollateralRequirementsResponseV1 is get_collateral_requirements output.
type CollateralRequirementsResponseV1 struct {
	RequiredCollateralValueUsd   string               `json:"required_collateral_value_usd"`
	AdditionalCollateralValueUsd string               `json:"additional_collateral_value_usd"`
	Required                     []AssetRequirementV1 `json:"required"`
}
