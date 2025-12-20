package provenance

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (c *ProvenanceClient) GetBalance(address string) ([]sdk.Coin, error) {
	bankClient := banktypes.NewQueryClient(c.Grpc.Conn)

	nextKey := []byte(nil)
	balances := []sdk.Coin{}
	for {
		bankRes, err := bankClient.AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{
			Address: address,
			Pagination: &query.PageRequest{
				Key:        nextKey,
				Limit:      100, // adjust as needed
				CountTotal: false,
			},
		})
		if err != nil {
			return nil, err
		}

		for _, balance := range bankRes.Balances {
			balances = append(balances, balance)
		}

		if len(bankRes.Pagination.NextKey) == 0 {
			break
		}
		nextKey = bankRes.Pagination.NextKey
	}

	return balances, nil
}
