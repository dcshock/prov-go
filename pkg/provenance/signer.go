package provenance

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// Signer produces a signature over canonical sign-bytes and identifies the
// account that signature belongs to.
//
// SignTx builds the transaction, then calls x/auth/signing.GetSignBytesAdapter
// to encode those bytes for SIGN_MODE_DIRECT. That function is not a hook; it
// only serializes the transaction. Signer.Sign is the extension point: an
// implementation may sign locally, or send msg to an external signer and
// return the signature bytes.
//
// Address is the bech32 account address and must match PubKey. Messages are
// built for that address, and the chain checks the signature against PubKey.
// When private key bytes are not available, both values come from this
// interface.
type Signer interface {
	Address() string
	PubKey() cryptotypes.PubKey
	Sign(msg []byte) ([]byte, error)
}

type privKeySigner struct {
	priv cryptotypes.PrivKey
}

// NewPrivKeySigner wraps a local private key. The key stays in-process;
// callers that cannot hold key bytes should implement Signer themselves.
func NewPrivKeySigner(priv cryptotypes.PrivKey) (Signer, error) {
	if priv == nil || priv.PubKey() == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	return &privKeySigner{priv: priv}, nil
}

func (s *privKeySigner) Address() string {
	return sdk.AccAddress(s.priv.PubKey().Address()).String()
}

func (s *privKeySigner) PubKey() cryptotypes.PubKey {
	return s.priv.PubKey()
}

func (s *privKeySigner) Sign(msg []byte) ([]byte, error) {
	return s.priv.Sign(msg)
}

func signerAccountAddress(signer Signer) (string, error) {
	pub := signer.PubKey()
	if pub == nil {
		return "", fmt.Errorf("signer has no public key")
	}
	derived := sdk.AccAddress(pub.Address()).String()
	addr := signer.Address()
	if addr == "" {
		return derived, nil
	}
	if addr != derived {
		return "", fmt.Errorf("signer address %s does not match public key address %s", addr, derived)
	}
	return derived, nil
}

// signWithSigner asks the SDK for SIGN_MODE_DIRECT bytes, then returns a
// signature produced by signer. The tx builder must already contain signer
// infos (an empty signature is enough) so the sign bytes include them.
func signWithSigner(ctx context.Context, txConfig client.TxConfig, txBuilder client.TxBuilder, signer Signer, chainID string, accountNumber, sequence uint64) (signing.SignatureV2, error) {
	var sigV2 signing.SignatureV2
	if signer == nil {
		return sigV2, fmt.Errorf("signer is nil")
	}
	pub := signer.PubKey()
	if pub == nil {
		return sigV2, fmt.Errorf("signer has no public key")
	}

	signerData := xauthsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Address:       signer.Address(),
		PubKey:        pub,
	}
	signBytes, err := xauthsigning.GetSignBytesAdapter(
		ctx,
		txConfig.SignModeHandler(),
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		return sigV2, err
	}

	signature, err := signer.Sign(signBytes)
	if err != nil {
		return sigV2, err
	}

	return signing.SignatureV2{
		PubKey: pub,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: signature,
		},
		Sequence: sequence,
	}, nil
}
