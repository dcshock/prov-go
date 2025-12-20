package provenance

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/google/uuid"
	meta "github.com/provenance-io/provenance/x/metadata/types"
)

func (c *ProvenanceClient) CreateScope(scopeSpecUUID, scopeUUID string) (*tx.BroadcastTxResponse, error) {
	// Verify that the scope doesn't already exist
	scope, err := c.GetScope(scopeUUID)
	if err != nil {
		return nil, fmt.Errorf("error getting scope: %w", err)
	}
	if scope != nil {
		return nil, fmt.Errorf("scope already exists")
	}

	msg := NewScope(c.Address, scopeSpecUUID, scopeUUID) // requires 10_000_000_000 additional fee

	txBz, err := c.Grpc.SignTx([]sdk.Msg{msg}, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), 10_000_000_000)
	if err != nil {
		return nil, fmt.Errorf("error creating tx: %w", err)
	}

	resp, err := c.Grpc.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("error broadcasting transaction: %w", err)
	}

	return resp, nil
}

// Delete a scope
func (c *ProvenanceClient) DeleteScope(scopeUuid string) (*tx.BroadcastTxResponse, error) {
	msg := NewDeleteScope(c.Address, scopeUuid)

	txBz, err := c.Grpc.SignTx([]sdk.Msg{msg}, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), 0)
	if err != nil {
		return nil, fmt.Errorf("error creating tx: %w", err)
	}

	resp, err := c.Grpc.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("error broadcasting transaction: %w", err)
	}

	return resp, nil
}

func (c *ProvenanceClient) UpdateRecords(session *meta.MsgWriteSessionRequest, records []meta.MsgWriteRecordRequest) (*tx.BroadcastTxResponse, error) {
	if records == nil {
		panic("records cannot be nil")
	}

	msgs := []sdk.Msg{session}
	for _, record := range records {
		msgs = append(msgs, &record)
	}

	txBz, err := c.Grpc.SignTx(msgs, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), 0)
	if err != nil {
		return nil, fmt.Errorf("error creating tx: %w", err)
	}

	resp, err := c.Grpc.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("error broadcasting transaction: %w", err)
	}

	return resp, nil
}

// Get the NAV for a given scope
func (c *ProvenanceClient) GetNAV(scopeId string) (*meta.NetAssetValue, error) {
	metaClient := meta.NewQueryClient(c.Grpc.Conn)
	res, err := metaClient.ScopeNetAssetValues(context.Background(), &meta.QueryScopeNetAssetValuesRequest{
		Id: scopeId,
	})
	if err != nil {
		return nil, err
	}

	// If we don't have any net asset values, return 0
	if len(res.NetAssetValues) == 0 {
		return nil, nil
	}

	nav := res.GetNetAssetValues()[0]
	return &nav, nil
}

func (c *ProvenanceClient) GetContractSpec(specId string) (*meta.ContractSpecification, error) {
	metaClient := meta.NewQueryClient(c.Grpc.Conn)
	res, err := metaClient.ContractSpecification(context.Background(), &meta.ContractSpecificationRequest{
		SpecificationId:    specId,
		IncludeRecordSpecs: false,
	})
	if err != nil {
		return nil, err
	}

	return res.ContractSpecification.Specification, nil
}

func (c *ProvenanceClient) GetScopeSpec(specId string) (*meta.ScopeSpecification, error) {
	metaClient := meta.NewQueryClient(c.Grpc.Conn)
	res, err := metaClient.ScopeSpecification(context.Background(), &meta.ScopeSpecificationRequest{
		SpecificationId:      specId,
		IncludeContractSpecs: false,
		IncludeRecordSpecs:   false,
	})
	if err != nil {
		return nil, err
	}

	return res.ScopeSpecification.Specification, nil
}

func (c *ProvenanceClient) GetRecordSpec(contractSpecUUID string, recordName string) (*meta.RecordSpecification, error) {
	specId := meta.RecordSpecMetadataAddress(uuid.MustParse(contractSpecUUID), recordName)

	metaClient := meta.NewQueryClient(c.Grpc.Conn)
	res, err := metaClient.RecordSpecification(context.Background(), &meta.RecordSpecificationRequest{
		SpecificationId: specId.String(),
	})
	if err != nil {
		return nil, err
	}

	return res.RecordSpecification.Specification, nil
}

func (c *ProvenanceClient) GetScope(scopeUuid string) (*meta.Scope, error) {
	metaClient := meta.NewQueryClient(c.Grpc.Conn)
	res, err := metaClient.Scope(context.Background(), &meta.ScopeRequest{
		ScopeId: scopeUuid,
	})
	if err != nil {
		return nil, err
	}

	return res.Scope.Scope, nil
}
