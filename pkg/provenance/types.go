package provenance

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/google/uuid"
	meta "github.com/provenance-io/provenance/x/metadata/types"
)

func NewBankSend(fromAddress string, toAddress string, denom string, amount int64) *banktypes.MsgSend {
	coins := sdk.NewCoins(sdk.NewInt64Coin(denom, amount))
	return &banktypes.MsgSend{
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Amount:      coins,
	}
}

func NewScope(signer, scopeSpecUuid, scopeUuid string) *meta.MsgWriteScopeRequest {
	scope := &meta.Scope{
		ScopeId:         meta.ScopeMetadataAddress(uuid.MustParse(scopeUuid)),
		SpecificationId: meta.MetadataAddress(meta.ScopeSpecMetadataAddress(uuid.MustParse(scopeSpecUuid))),
		Owners: []meta.Party{
			{
				Address:  signer,
				Role:     meta.PartyType_PARTY_TYPE_OWNER,
				Optional: false,
			},
		},
		ValueOwnerAddress: signer,
	}

	msgWriteScope := &meta.MsgWriteScopeRequest{
		Scope:   *scope,
		Signers: []string{signer},
	}

	return msgWriteScope
}

func NewDeleteScope(signer, scopeUuid string) *meta.MsgDeleteScopeRequest {
	msgDeleteScope := &meta.MsgDeleteScopeRequest{
		ScopeId: meta.ScopeMetadataAddress(uuid.MustParse(scopeUuid)),
		Signers: []string{signer},
	}

	return msgDeleteScope
}

func NewSession(signer, scopeUuid, sessionUuid, sessionName, contractSpecUuid string) *meta.MsgWriteSessionRequest {
	sessionID := meta.SessionMetadataAddress(uuid.MustParse(scopeUuid), uuid.MustParse(sessionUuid))

	contractSpecID := meta.ContractSpecMetadataAddress(uuid.MustParse(contractSpecUuid))

	session := meta.Session{
		Name:            sessionName,
		SessionId:       sessionID,
		SpecificationId: contractSpecID,
		Parties: []meta.Party{
			{
				Address: signer,
				Role:    meta.PartyType_PARTY_TYPE_OWNER,
			},
		},
	}

	msgSession := &meta.MsgWriteSessionRequest{
		Session: session,
		Signers: []string{signer},
	}

	return msgSession
}

func NewJsonRecord(signer, processName, processMethod, contractSpecUuid, inputName, recordName, recordJson string, sessionId meta.MetadataAddress) *meta.MsgWriteRecordRequest {
	record := &meta.Record{
		SpecificationId: meta.RecordSpecMetadataAddress(uuid.MustParse(contractSpecUuid), recordName),
		SessionId:       sessionId,
		Name:            recordName,
		Process: meta.Process{
			ProcessId: &meta.Process_Hash{
				Hash: "{}",
			},
			Name:   processName,
			Method: processMethod,
		},
		Inputs: []meta.RecordInput{
			{
				Name:     inputName,
				TypeName: "json",
				Status:   meta.RecordInputStatus_Proposed,
				Source: &meta.RecordInput_Hash{
					Hash: "{}",
				},
			},
		},
		Outputs: []meta.RecordOutput{
			{
				Hash:   recordJson,
				Status: meta.ResultStatus_RESULT_STATUS_PASS,
			},
		},
	}

	msgWriteRecord := &meta.MsgWriteRecordRequest{
		Record:  *record,
		Signers: []string{signer},
	}

	return msgWriteRecord
}
