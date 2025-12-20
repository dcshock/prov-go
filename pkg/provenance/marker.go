package provenance

import (
	"context"
	"fmt"
	"strings"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	marker "github.com/provenance-io/provenance/x/marker/types"
	metadata "github.com/provenance-io/provenance/x/metadata/keeper"
	"github.com/provenance-io/provenance/x/metadata/types"
)

type AccountValue struct {
	Address string
	Total   sdk.Coin
	NFTs    []NFTAccount
}

type NFTAccount struct {
	Denom              string
	NAV                *sdk.Coin
	UUID               string
	UpdatedBlockHeight uint64
}

type MarkerValueResult struct {
	MarkerId string
	Value    sdk.Coin
	Error    error
}

func (c *ProvenanceClient) GetMarker(denomOrAddress string) (*marker.MarkerAccountI, error) {
	res, err := (*c.MarkerClient()).Marker(context.Background(), &marker.QueryMarkerRequest{
		Id: denomOrAddress,
	})

	if err != nil {
		return nil, err
	}

	var acct marker.MarkerAccountI

	err = c.Cdc.UnpackAny(res.Marker, &acct)
	if err != nil {
		return nil, fmt.Errorf("error unpacking marker account: %w", err)
	}

	return &acct, nil
}

func (c *ProvenanceClient) GetMarkerAddress(denom string) (*string, error) {
	acct, err := c.GetMarker(denom)
	if acct == nil || err != nil {
		return nil, fmt.Errorf("no cached value for %s", denom)
	}

	// Get the address of the marker account that we'll use to query the balances
	markerAddress := (*acct).GetAddress().String()
	return &markerAddress, nil
}

// GetAccountValue gets the value of an account by address/denom
func (c *ProvenanceClient) GetAccountValue(addressOrDenom string) (*AccountValue, error) {
	// WaitGroup and Channels to handle results from the NAV queries
	wg := sync.WaitGroup{}
	totalChan := make(chan *AccountValue, 1)
	resultsChan := make(chan *NFTAccount, 1)

	// total routine to sum up the results from the results channel
	go func() {
		// read the results from the results channel
		total := AccountValue{
			Address: addressOrDenom,
			Total:   sdk.NewInt64Coin("usd", 0),
			NFTs:    []NFTAccount{},
		}
		for result := range resultsChan {
			total.Total = total.Total.Add(*result.NAV)
			total.NFTs = append(total.NFTs, *result)
		}
		totalChan <- &total

		close(totalChan)
	}()

	nextKey := []byte(nil)
	for {
		bankRes, err := (*c.BankClient()).AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{
			Address: addressOrDenom,
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
			if strings.HasPrefix(balance.Denom, "nft") {
				parts := strings.Split(balance.Denom, "/")
				if len(parts) == 2 {
					scopeId := parts[1]

					wg.Add(1)
					c.Pool.Submit(func() {
						defer wg.Done()

						var nav *types.NetAssetValue
						nav, err = c.GetNAV(scopeId)
						if err != nil {
							// Panic here since we don't want to report any results if there is an error
							panic(err)
						}

						// No NAV? Just exit...
						if nav == nil {
							nav = &types.NetAssetValue{
								Price:              sdk.NewInt64Coin("usd", 0),
								UpdatedBlockHeight: 0,
							}
						}

						metadataAddress, err := metadata.ParseScopeID(scopeId)
						if err != nil {
							panic(err)
						}

						uuid, err := metadataAddress.ScopeUUID()
						if err != nil {
							panic(err)
						}

						nftAccount := NFTAccount{
							Denom:              scopeId,
							NAV:                &nav.Price,
							UUID:               uuid.String(),
							UpdatedBlockHeight: nav.UpdatedBlockHeight,
						}

						if nav.Price.Denom == "usd" {
							resultsChan <- &nftAccount
						}
					})
				}
			}
		}

		if len(bankRes.Pagination.NextKey) == 0 {
			break
		}
		nextKey = bankRes.Pagination.NextKey
	}

	// wait for all the results to be added to the results channel
	wg.Wait()
	close(resultsChan)

	// Get the total from the total channel
	total := <-totalChan
	return total, nil
}
