package provenance

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// GetBalances retrieves all balances for the given address and returns them as a slice.
// It handles pagination automatically and will return all balances across multiple pages.
// The function respects context cancellation and will return ctx.Err() if the context
// is cancelled before completion.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - address: The blockchain address to query balances for
//
// Returns:
//   - []sdk.Coin: A slice containing all balances for the address
//   - error: Returns an error if the query fails or context is cancelled
func (c *ProvenanceClient) GetBalances(ctx context.Context, address string) ([]sdk.Coin, error) {
	balancesChan, errChan := c.GetBalancesStream(ctx, address)

	balances := []sdk.Coin{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case balance, ok := <-balancesChan:
			if !ok {
				return balances, nil
			}
			balances = append(balances, balance)
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		}
	}
}

// GetBalancesStream retrieves balances for the given address and streams them through channels.
// This function is useful for processing large numbers of balances incrementally without
// loading them all into memory at once. It handles pagination automatically and sends
// balances as they are retrieved from the blockchain.
//
// The function returns two channels:
//   - balancesChan: Receives balance values as they are retrieved. The channel is closed
//     when all balances have been sent or an error occurs.
//   - errChan: Receives any errors that occur during retrieval. If an error is sent,
//     the balancesChan will be closed and no more balances will be sent.
//
// The function respects context cancellation. If the context is cancelled, ctx.Err()
// will be sent on errChan and both channels will be closed.
//
// The caller must read from both channels until they are closed to prevent goroutine leaks.
// If the context is cancelled or an error occurs, the caller should stop reading from
// balancesChan and read the error from errChan.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - address: The blockchain address to query balances for
//
// Returns:
//   - chan sdk.Coin: Channel that receives balance values. Closed when complete or on error.
//   - chan error: Channel that receives errors. Closed when the goroutine exits.
func (c *ProvenanceClient) GetBalancesStream(ctx context.Context, address string) (chan sdk.Coin, chan error) {
	pageBufferSize := uint64(50) // Match the page size of the client request.

	balancesChan := make(chan sdk.Coin, pageBufferSize)
	errChan := make(chan error, 1) // Buffer of 1 to prevent blocking the goroutine.

	go func() {
		defer close(balancesChan)
		defer close(errChan)

		nextKey := []byte(nil)
		for {
			bankRes, err := (*c.BankClient()).AllBalances(ctx, &banktypes.QueryAllBalancesRequest{
				Address: address,
				Pagination: &query.PageRequest{
					Key:        nextKey,
					Limit:      pageBufferSize,
					CountTotal: false,
				},
			})

			if err != nil {
				if ctx.Err() != nil {
					errChan <- ctx.Err()
					return
				}
				errChan <- err
				return
			}

			for _, balance := range bankRes.Balances {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case balancesChan <- balance:
				}
			}

			if len(bankRes.Pagination.NextKey) == 0 {
				break
			}
			nextKey = bankRes.Pagination.NextKey
		}
	}()

	return balancesChan, errChan
}
