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
				txBz, err := c.Grpc.SignTx(buff, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), txFee)

				// Clear the buffer
				buff = []sdk.Msg{}

				if err != nil {
					attrErrChan <- err
					continue
				}

				renderer.SetCurrent(fmt.Sprintf("Sending tx with %d attributes", len(buff)))
				resp, err := c.Grpc.BroadcastTx(txBz)
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

			attrs, err := c.GetAttributes(attr.Name, attr.Acct)
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

func (c *ProvenanceClient) GetAttributes(attrName, acct string) ([]attrtypes.Attribute, error) {
	query := attrtypes.QueryAttributeRequest{
		Account: acct,
		Name:    attrName,
	}

	attrClient := attrtypes.NewQueryClient(c.Grpc.Conn)
	res, err := attrClient.Attribute(context.Background(), &query)
	if err != nil {
		return nil, err
	}

	return res.Attributes, nil
}

func (c *ProvenanceClient) GetAttributedAccounts(name string) ([]string, error) {
	attrClient := attrtypes.NewQueryClient(c.Grpc.Conn)

	accounts := []string{}
	nextKey := []byte(nil)
	for {
		query := attrtypes.QueryAttributeAccountsRequest{
			AttributeName: name,
			Pagination: &query.PageRequest{
				Key:        nextKey,
				Limit:      100, // adjust as needed
				CountTotal: false,
			},
		}

		res, err := attrClient.AttributeAccounts(context.Background(), &query)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, res.Accounts...)

		nextKey = res.Pagination.NextKey
		if len(nextKey) == 0 {
			break
		}
	}

	return accounts, nil
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
