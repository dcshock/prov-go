package provenance

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx"
	attrtypes "github.com/provenance-io/provenance/x/attribute/types"
)

type Attribute struct {
	Name      string
	Acct      string
	JsonValue string
}

func (c *ProvenanceClient) AddAttributes(attrs []Attribute) (chan *tx.BroadcastTxResponse, chan error) {
	// Go routine to add attributes in chunks.
	attrAddChan := make(chan *attrtypes.MsgAddAttributeRequest, len(attrs))

	attrRespChan := make(chan *tx.BroadcastTxResponse)
	attrErrChan := make(chan error)

	renderer := Renderer{
		first:   true,
		Count:   0,
		Current: "",
	}

	go func() {
		defer close(attrRespChan)
		defer close(attrErrChan)

		buff := []sdk.Msg{}

		closed := false
		for {
			if closed {
				break
			}

			attr, ok := <-attrAddChan
			if !ok {
				// We still need to send the last batch of attributes
				closed = true
			} else {
				renderer.SetCurrent(fmt.Sprintf("Adding attribute: %s %s", attr.Name, attr.Account))
				buff = append(buff, attr)
			}

			// Use a buff size of 75 to limit our chance of hitting the 4m max gas limit
			if len(buff) == 75 || (closed && len(buff) > 0) {
				// Attribute additions require 10_000_000_000 additional fee
				txFee := int64(10_000_000_000) * int64(len(buff))
				txBz, err := c.SignTx(buff, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), txFee)

				// Clear the buffer
				buff = []sdk.Msg{}

				if err != nil {
					attrErrChan <- err
					continue
				}

				renderer.SetCurrent(fmt.Sprintf("Sending tx with %d attributes", len(buff)))
				resp, err := c.BroadcastTx(txBz)
				if err != nil {
					attrErrChan <- err
					continue
				}

				attrRespChan <- resp
			}
		}
	}()

	go func() {
		defer close(attrAddChan)

		renderer.SetCurrent(fmt.Sprintf("Adding attributes: %d", len(attrs)))

		for _, attr := range attrs {
			renderer.IncrementCount()
			renderer.SetCurrent(fmt.Sprintf("Processing: %s %s", attr.Name, attr.Acct))

			attrs, err := c.GetAttributes(context.Background(), attr.Name, attr.Acct)
			if err != nil {
				attrErrChan <- err
				continue
			}

			if len(attrs) > 0 {
				attrErrChan <- fmt.Errorf("attribute already exists %s %s", attr.Name, attr.Acct)
				continue
			}

			if attr.JsonValue == "" {
				attrAddChan <- &attrtypes.MsgAddAttributeRequest{
					Name:          attr.Name,
					Value:         []byte("{}"),
					Account:       attr.Acct,
					Owner:         c.Address,
					AttributeType: attrtypes.AttributeType_JSON,
				}
				continue
			}

			attrAddChan <- &attrtypes.MsgAddAttributeRequest{
				Name:          attr.Name,
				Value:         []byte(attr.JsonValue),
				Account:       attr.Acct,
				AttributeType: attrtypes.AttributeType_JSON,
				Owner:         c.Address,
			}
		}
	}()

	return attrRespChan, attrErrChan
}

// GetAttributes retrieves all attributes for the given attribute name and account, and returns them as a slice.
// It handles pagination automatically and will return all attributes across multiple pages.
// The function respects context cancellation and will return ctx.Err() if the context
// is cancelled before completion.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - attrName: The attribute name to query for
//   - acct: The account address to query attributes for
//
// Returns:
//   - []attrtypes.Attribute: A slice containing all attributes matching the name and account
//   - error: Returns an error if the query fails or context is cancelled
func (c *ProvenanceClient) GetAttributes(ctx context.Context, attrName, acct string) ([]attrtypes.Attribute, error) {
	attributesChan, errChan := c.GetAttributesStream(ctx, attrName, acct)

	attributes := []attrtypes.Attribute{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case attribute, ok := <-attributesChan:
			if !ok {
				return attributes, nil
			}
			attributes = append(attributes, attribute)
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		}
	}
}

// GetAttributesStream retrieves attributes for the given attribute name and account and streams them through channels.
// This function is useful for processing large numbers of attributes incrementally without
// loading them all into memory at once. It handles pagination automatically and sends
// attributes as they are retrieved from the blockchain.
//
// The function returns two channels:
//   - attributesChan: Receives attribute values as they are retrieved. The channel is closed
//     when all attributes have been sent or an error occurs.
//   - errChan: Receives any errors that occur during retrieval. If an error is sent,
//     the attributesChan will be closed and no more attributes will be sent.
//
// The function respects context cancellation. If the context is cancelled, ctx.Err()
// will be sent on errChan and both channels will be closed.
//
// The caller must read from both channels until they are closed to prevent goroutine leaks.
// If the context is cancelled or an error occurs, the caller should stop reading from
// attributesChan and read the error from errChan.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - attrName: The attribute name to query for
//   - acct: The account address to query attributes for
//
// Returns:
//   - chan attrtypes.Attribute: Channel that receives attribute values. Closed when complete or on error.
//   - chan error: Channel that receives errors. Closed when the goroutine exits.
func (c *ProvenanceClient) GetAttributesStream(ctx context.Context, attrName, acct string) (chan attrtypes.Attribute, chan error) {
	pageBufferSize := uint64(100) // Match the page size of the client request.

	attributesChan := make(chan attrtypes.Attribute, pageBufferSize)
	errChan := make(chan error, 1) // Buffer of 1 to prevent blocking the goroutine.

	go func() {
		defer close(attributesChan)
		defer close(errChan)

		nextKey := []byte(nil)
		for {
			attrQuery := attrtypes.QueryAttributeRequest{
				Account: acct,
				Name:    attrName,
				Pagination: &query.PageRequest{
					Key:        nextKey,
					Limit:      pageBufferSize,
					CountTotal: false,
				},
			}

			res, err := (*c.AttributeClient()).Attribute(ctx, &attrQuery)
			if err != nil {
				if ctx.Err() != nil {
					errChan <- ctx.Err()
					return
				}
				errChan <- err
				return
			}

			for _, attribute := range res.Attributes {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case attributesChan <- attribute:
				}
			}

			if len(res.Pagination.NextKey) == 0 {
				break
			}
			nextKey = res.Pagination.NextKey
		}
	}()

	return attributesChan, errChan
}

// GetAttributedAccounts retrieves all accounts that have the given attribute name and returns them as a slice.
// It handles pagination automatically and will return all accounts across multiple pages.
// The function respects context cancellation and will return ctx.Err() if the context
// is cancelled before completion.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - name: The attribute name to query accounts for
//
// Returns:
//   - []string: A slice containing all accounts that have the attribute
//   - error: Returns an error if the query fails or context is cancelled
func (c *ProvenanceClient) GetAttributedAccounts(ctx context.Context, name string) ([]string, error) {
	accountsChan, errChan := c.GetAttributedAccountsStream(ctx, name)

	accounts := []string{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case account, ok := <-accountsChan:
			if !ok {
				return accounts, nil
			}
			accounts = append(accounts, account)
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		}
	}
}

// GetAttributedAccountsStream retrieves accounts that have the given attribute name and streams them through channels.
// This function is useful for processing large numbers of accounts incrementally without
// loading them all into memory at once. It handles pagination automatically and sends
// accounts as they are retrieved from the blockchain.
//
// The function returns two channels:
//   - accountsChan: Receives account values as they are retrieved. The channel is closed
//     when all accounts have been sent or an error occurs.
//   - errChan: Receives any errors that occur during retrieval. If an error is sent,
//     the accountsChan will be closed and no more accounts will be sent.
//
// The function respects context cancellation. If the context is cancelled, ctx.Err()
// will be sent on errChan and both channels will be closed.
//
// The caller must read from both channels until they are closed to prevent goroutine leaks.
// If the context is cancelled or an error occurs, the caller should stop reading from
// accountsChan and read the error from errChan.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - name: The attribute name to query accounts for
//
// Returns:
//   - chan string: Channel that receives account values. Closed when complete or on error.
//   - chan error: Channel that receives errors. Closed when the goroutine exits.
func (c *ProvenanceClient) GetAttributedAccountsStream(ctx context.Context, name string) (chan string, chan error) {
	pageBufferSize := uint64(100) // Match the page size of the client request.

	accountsChan := make(chan string, pageBufferSize)
	errChan := make(chan error, 1) // Buffer of 1 to prevent blocking the goroutine.

	go func() {
		defer close(accountsChan)
		defer close(errChan)

		nextKey := []byte(nil)
		for {
			query := attrtypes.QueryAttributeAccountsRequest{
				AttributeName: name,
				Pagination: &query.PageRequest{
					Key:        nextKey,
					Limit:      pageBufferSize,
					CountTotal: false,
				},
			}

			res, err := (*c.AttributeClient()).AttributeAccounts(ctx, &query)
			if err != nil {
				if ctx.Err() != nil {
					errChan <- ctx.Err()
					return
				}
				errChan <- err
				return
			}

			for _, account := range res.Accounts {
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case accountsChan <- account:
				}
			}

			if len(res.Pagination.NextKey) == 0 {
				break
			}
			nextKey = res.Pagination.NextKey
		}
	}()

	return accountsChan, errChan
}

type Renderer struct {
	first bool
	mu    sync.Mutex

	Count   int
	Current string
}

func NewRenderer() *Renderer {
	return &Renderer{
		first:   true,
		mu:      sync.Mutex{},
		Count:   0,
		Current: "",
	}
}

func (r *Renderer) IncrementCount() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Count++
	r.render()
}

func (r *Renderer) SetCurrent(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Current = msg
	r.render()
}

func (r *Renderer) render() {
	if r.first {
		fmt.Println("--------------------------------")
		r.first = false
	} else {
		fmt.Fprint(os.Stdout, "\x1b[2A")
	}

	// For each line: clear it, print new value.
	clear := "\x1b[2K"
	fmt.Printf("%s%-10s %d\n", clear, "processing #:", r.Count)
	fmt.Printf("%s%-10s %s\n", clear, "msg:", r.Current[0:int(math.Min(float64(len(r.Current)), 80))])
}
