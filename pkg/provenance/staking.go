package provenance

import (
	"context"

	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	stakingtypes "cosmossdk.io/api/cosmos/staking/v1beta1"
	"google.golang.org/grpc"
)

// GetStakingValidators retrieves all validators and returns them as a slice.
// It handles pagination automatically and will return all validators across multiple pages.
// The function respects context cancellation and will return ctx.Err() if the context
// is cancelled before completion.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - opts: Optional gRPC call options (e.g., grpc.WaitForReady, custom timeouts, etc.)
//
// Returns:
//   - []*stakingtypes.Validator: A slice containing all validators
//   - error: Returns an error if the query fails or context is cancelled
func (c *ProvenanceClient) GetStakingValidators(ctx context.Context, opts ...grpc.CallOption) ([]*stakingtypes.Validator, error) {
	validatorsChan, errChan := c.GetStakingValidatorsStream(ctx, opts...)

	validators := []*stakingtypes.Validator{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case validator, ok := <-validatorsChan:
			if !ok {
				return validators, nil
			}
			validators = append(validators, validator)
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		}
	}
}

// GetStakingValidatorsStream retrieves validators and streams them through channels.
// This function is useful for processing large numbers of validators incrementally without
// loading them all into memory at once. It handles pagination automatically and sends
// validators as they are retrieved from the blockchain.
//
// The function returns two channels:
//   - validatorsChan: Receives validator values as they are retrieved. The channel is closed
//     when all validators have been sent or an error occurs.
//   - errChan: Receives any errors that occur during retrieval. If an error is sent,
//     the validatorsChan will be closed and no more validators will be sent.
//
// The function respects context cancellation. If the context is cancelled, ctx.Err()
// will be sent on errChan and both channels will be closed.
//
// The caller must read from both channels until they are closed to prevent goroutine leaks.
// If the context is cancelled or an error occurs, the caller should stop reading from
// validatorsChan and read the error from errChan.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - opts: Optional gRPC call options (e.g., grpc.WaitForReady, custom timeouts, etc.)
//
// Returns:
//   - chan *stakingtypes.Validator: Channel that receives validator values. Closed when complete or on error.
//   - chan error: Channel that receives errors. Closed when the goroutine exits.
func (c *ProvenanceClient) GetStakingValidatorsStream(ctx context.Context, opts ...grpc.CallOption) (chan *stakingtypes.Validator, chan error) {
	pageBufferSize := uint64(100) // Match the page size of the client request.

	validatorsChan := make(chan *stakingtypes.Validator, pageBufferSize)
	errChan := make(chan error, 1) // Buffer of 1 to prevent blocking the goroutine.

	go func() {
		defer close(validatorsChan)
		defer close(errChan)

		nextKey := []byte(nil)
		for {
			res, err := (*c.StakingClient()).Validators(ctx, &stakingtypes.QueryValidatorsRequest{
				Pagination: &queryv1beta1.PageRequest{
					Key:        nextKey,
					Limit:      pageBufferSize,
					CountTotal: false,
				},
			}, opts...)

			if err != nil {
				if ctx.Err() != nil {
					errChan <- ctx.Err()
					return
				}
				errChan <- err
				return
			}

			for _, validator := range res.Validators {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case validatorsChan <- validator:
				}
			}

			if len(res.Pagination.NextKey) == 0 {
				break
			}
			nextKey = res.Pagination.NextKey
		}
	}()

	return validatorsChan, errChan
}

// GetDelegationsByDelegator retrieves all delegations for the given delegator address and returns them as a slice.
// It handles pagination automatically and will return all delegations across multiple pages.
// The function respects context cancellation and will return ctx.Err() if the context
// is cancelled before completion.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - delegatorAddress: The blockchain address of the delegator to query delegations for
//   - opts: Optional gRPC call options (e.g., grpc.WaitForReady, custom timeouts, etc.)
//
// Returns:
//   - []*stakingtypes.DelegationResponse: A slice containing all delegation responses for the delegator
//   - error: Returns an error if the query fails or context is cancelled
func (c *ProvenanceClient) GetDelegationsByDelegator(ctx context.Context, delegatorAddress string, opts ...grpc.CallOption) ([]*stakingtypes.DelegationResponse, error) {
	delegationsChan, errChan := c.GetDelegationsByDelegatorStream(ctx, delegatorAddress, opts...)

	delegations := []*stakingtypes.DelegationResponse{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case delegation, ok := <-delegationsChan:
			if !ok {
				return delegations, nil
			}
			delegations = append(delegations, delegation)
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		}
	}
}

// GetDelegationsByDelegatorStream retrieves delegations for the given delegator address and streams them through channels.
// This function is useful for processing large numbers of delegations incrementally without
// loading them all into memory at once. It handles pagination automatically and sends
// delegations as they are retrieved from the blockchain.
//
// The function returns two channels:
//   - delegationsChan: Receives delegation response values as they are retrieved. The channel is closed
//     when all delegations have been sent or an error occurs.
//   - errChan: Receives any errors that occur during retrieval. If an error is sent,
//     the delegationsChan will be closed and no more delegations will be sent.
//
// The function respects context cancellation. If the context is cancelled, ctx.Err()
// will be sent on errChan and both channels will be closed.
//
// The caller must read from both channels until they are closed to prevent goroutine leaks.
// If the context is cancelled or an error occurs, the caller should stop reading from
// delegationsChan and read the error from errChan.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - delegatorAddress: The blockchain address of the delegator to query delegations for
//   - opts: Optional gRPC call options (e.g., grpc.WaitForReady, custom timeouts, etc.)
//
// Returns:
//   - chan *stakingtypes.DelegationResponse: Channel that receives delegation response values. Closed when complete or on error.
//   - chan error: Channel that receives errors. Closed when the goroutine exits.
func (c *ProvenanceClient) GetDelegationsByDelegatorStream(ctx context.Context, delegatorAddress string, opts ...grpc.CallOption) (chan *stakingtypes.DelegationResponse, chan error) {
	pageBufferSize := uint64(100) // Match the page size of the client request.

	delegationsChan := make(chan *stakingtypes.DelegationResponse, pageBufferSize)
	errChan := make(chan error, 1) // Buffer of 1 to prevent blocking the goroutine.

	go func() {
		defer close(delegationsChan)
		defer close(errChan)

		nextKey := []byte(nil)
		for {
			res, err := (*c.StakingClient()).DelegatorDelegations(ctx, &stakingtypes.QueryDelegatorDelegationsRequest{
				DelegatorAddr: delegatorAddress,
				Pagination: &queryv1beta1.PageRequest{
					Key:        nextKey,
					Limit:      pageBufferSize,
					CountTotal: false,
				},
			}, opts...)

			if err != nil {
				if ctx.Err() != nil {
					errChan <- ctx.Err()
					return
				}
				errChan <- err
				return
			}

			for _, delegation := range res.DelegationResponses {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case delegationsChan <- delegation:
				}
			}

			if len(res.Pagination.NextKey) == 0 {
				break
			}
			nextKey = res.Pagination.NextKey
		}
	}()

	return delegationsChan, errChan
}
