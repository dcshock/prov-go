package provenance

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
)

// WrapGroupProposal wraps the provided sdk.Msgs in a group MsgSubmitProposal,
// using the client's own address as the sole proposer. Callers are expected to
// populate GroupPolicyAddress, Title, Summary, Metadata, and Exec on the
// returned message before signing/broadcasting.
func (c *ProvenanceClient) WrapGroupProposal(groupPolicyAddress, metadata, title, summary string, msgs ...sdk.Msg) (*grouptypes.MsgSubmitProposal, error) {
	proposal := &grouptypes.MsgSubmitProposal{
		GroupPolicyAddress: groupPolicyAddress,
		Metadata:           metadata,
		Title:              title,
		Summary:            summary,
		Exec:               grouptypes.Exec_EXEC_TRY,
		Proposers:          []string{c.Address},
	}
	if err := proposal.SetMsgs(msgs); err != nil {
		return nil, err
	}
	return proposal, nil
}
