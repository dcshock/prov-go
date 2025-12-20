package provenance

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (c *ProvenanceClient) GetBalances(address string) ([]sdk.Coin, error) {
	balancesChan, errChan := c.GetBalancesStream(address)

	balances := []sdk.Coin{}
	for {
		select {
		case balance, ok := <-balancesChan:
			if !ok {
				return balances, nil
			} else {
				balances = append(balances, *balance)
			}
		case err := <-errChan:
			return nil, err
		}
	}
}

func (c *ProvenanceClient) GetBalancesStream(address string) (chan *sdk.Coin, chan error) {
	balancesChan := make(chan *sdk.Coin)
	errChan := make(chan error)

	go func() {
		defer close(balancesChan)
		defer close(errChan)

		nextKey := []byte(nil)
		for {
			bankRes, err := (*c.BankClient()).AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{
				Address: address,
				Pagination: &query.PageRequest{
					Key:        nextKey,
					Limit:      50,
					CountTotal: false,
				},
			})

			if err != nil {
				errChan <- err
				return
			}

			for _, balance := range bankRes.Balances {
				balancesChan <- &balance
			}

			if len(bankRes.Pagination.NextKey) == 0 {
				break
			}
			nextKey = bankRes.Pagination.NextKey
		}
	}()

	return balancesChan, errChan
}
