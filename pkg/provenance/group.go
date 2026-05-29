package provenance

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
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

// GroupVote signs and broadcasts MsgVote for proposalID using the client's address
// as voter. metadata may be empty. exec is when to run the proposal after voting
// (e.g. grouptypes.Exec_EXEC_TRY or grouptypes.Exec_EXEC_UNSPECIFIED).
func (c *ProvenanceClient) GroupVote(proposalID uint64, option grouptypes.VoteOption, metadata string, exec grouptypes.Exec) (*tx.BroadcastTxResponse, error) {
	msg := &grouptypes.MsgVote{
		ProposalId: proposalID,
		Voter:      c.Address,
		Option:     option,
		Metadata:   metadata,
		Exec:       exec,
	}

	txBz, err := c.SignTx([]sdk.Msg{msg}, c.PrivKey.Bytes(), c.AccountNumber, c.NextSequence(), 0)
	if err != nil {
		return nil, fmt.Errorf("group vote tx: %w", err)
	}

	resp, err := c.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("group vote broadcast: %w", err)
	}

	return resp, nil
}
