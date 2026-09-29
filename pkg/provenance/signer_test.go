package provenance

import (
	"bytes"
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

func TestPrivKeySignerAddressAndSign(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("pb", "pbpub")

	priv := secp256k1.GenPrivKey()
	signer, err := NewPrivKeySigner(priv)
	if err != nil {
		t.Fatal(err)
	}

	wantAddr := sdk.AccAddress(priv.PubKey().Address()).String()
	if signer.Address() != wantAddr {
		t.Fatalf("address: got %s want %s", signer.Address(), wantAddr)
	}

	msg := []byte("hello")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !priv.PubKey().VerifySignature(msg, sig) {
		t.Fatal("signature did not verify")
	}
}

func TestSignerAccountAddress(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("pb", "pbpub")

	priv := secp256k1.GenPrivKey()
	derived := sdk.AccAddress(priv.PubKey().Address()).String()

	addr, err := signerAccountAddress(&recordingSigner{priv: priv, addr: derived})
	if err != nil {
		t.Fatal(err)
	}
	if addr != derived {
		t.Fatalf("address: got %s want %s", addr, derived)
	}

	addr, err = signerAccountAddress(&recordingSigner{priv: priv, addr: ""})
	if err != nil {
		t.Fatal(err)
	}
	if addr != derived {
		t.Fatalf("empty address: got %s want %s", addr, derived)
	}

	if _, err := signerAccountAddress(&recordingSigner{priv: priv, addr: "pb1notthepubkey"}); err == nil {
		t.Fatal("expected address mismatch error")
	}
}

// recordingSigner stands in for an external signer: Sign receives the canonical
// bytes and returns a signature without exposing the private key to SignTx.
type recordingSigner struct {
	priv cryptotypes.PrivKey
	addr string
	got  []byte
}

func (s *recordingSigner) Address() string { return s.addr }

func (s *recordingSigner) PubKey() cryptotypes.PubKey { return s.priv.PubKey() }

func (s *recordingSigner) Sign(msg []byte) ([]byte, error) {
	s.got = append([]byte(nil), msg...)
	return s.priv.Sign(msg)
}

func TestSignWithSignerSendsCanonicalBytes(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("pb", "pbpub")

	priv := secp256k1.GenPrivKey()
	local, err := NewPrivKeySigner(priv)
	if err != nil {
		t.Fatal(err)
	}
	remote := &recordingSigner{priv: priv, addr: local.Address()}

	txConfig := NewTxConfig()
	txBuilder := txConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(NewBankSend(remote.Address(), remote.Address(), "nhash", 1)); err != nil {
		t.Fatal(err)
	}
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("nhash", 1)))
	txBuilder.SetGasLimit(100000)
	if err := txBuilder.SetSignatures(signing.SignatureV2{
		PubKey: remote.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 7,
	}); err != nil {
		t.Fatal(err)
	}

	sigV2, err := signWithSigner(context.Background(), txConfig, txBuilder, remote, "pio-testnet-1", 3, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(remote.got) == 0 {
		t.Fatal("signer was not given sign bytes")
	}

	raw := sigV2.Data.(*signing.SingleSignatureData)
	if !remote.PubKey().VerifySignature(remote.got, raw.Signature) {
		t.Fatal("signature did not verify over the bytes sent to the signer")
	}

	again, err := priv.Sign(remote.got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, raw.Signature) {
		t.Fatal("signature bytes differ from signing the adapter output")
	}
}
