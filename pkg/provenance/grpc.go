package provenance

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	attrtypes "github.com/provenance-io/provenance/x/attribute/types"
	marker "github.com/provenance-io/provenance/x/marker/types"
	meta "github.com/provenance-io/provenance/x/metadata/types"
	registry "github.com/provenance-io/provenance/x/registry/types"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCConnection struct {
	Conn             *grpc.ClientConn
	BlockchainConfig BlockchainConfigProvider
}

func NewGRPCConnection(config BlockchainConfigProvider) (*GRPCConnection, error) {
	fmt.Printf("Connecting to Provenance at grpc(s)://%s\n", config.URI())

	var grpcConn *grpc.ClientConn

	// Use system TLS credentials for gRPCs connections, skipping certFile check
	// For insecure connections, use insecure credentials
	if config.CertFile() != "" {
		// Skip certFile check and use system TLS credentials
		credentials := credentials.NewTLS(nil)
		var err error
		grpcConn, err = grpc.NewClient(config.URI(), grpc.WithTransportCredentials(credentials))
		if err != nil {
			return nil, err
		}
	} else {
		credentials := insecure.NewCredentials()

		var err error
		grpcConn, err = grpc.NewClient(config.URI(), grpc.WithTransportCredentials(credentials))
		if err != nil {
			return nil, err
		}
	}

	g := GRPCConnection{
		Conn:             grpcConn,
		BlockchainConfig: config,
	}

	return &g, nil
}

func (c *GRPCConnection) Close() error {
	return c.Conn.Close()
}

// Create a new tx config with the appropriate provenance interfaces registered
func NewTxConfig() client.TxConfig {
	reg := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(reg)
	marker.RegisterInterfaces(reg)
	meta.RegisterInterfaces(reg)
	banktypes.RegisterInterfaces(reg)
	attrtypes.RegisterInterfaces(reg)
	registry.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	return txConfig
}

func (c *GRPCConnection) SignTx(msg []sdk.Msg, privKeyBytes []byte, accountNumber, sequence uint64, additionalFee int64) ([]byte, error) {
	ctx := context.Background()

	txConfig := NewTxConfig()

	// Add the msgs to the tx builder
	txBuilder := txConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg...); err != nil {
		return nil, err
	}

	// Read the private key from the bytes
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	privs := []cryptotypes.PrivKey{privKey}
	fmt.Println("Public key:", privKey.PubKey().Address())
	fmt.Println("Public key string:", sdk.AccAddress(privKey.PubKey().Address()).String())

	// Create a simple signature placeholder
	// In a real implementation, you would:
	// 1. Get the sign bytes from the transaction
	// 2. Sign those bytes with your private key
	// 3. Create a proper SignatureV2 with the signature and public key
	var sigs []signing.SignatureV2
	for _, priv := range privs {
		sigData := signing.SignatureV2{
			PubKey: priv.PubKey(), // Set to your public key
			Data: &signing.SingleSignatureData{
				SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
				Signature: nil,
			},
			Sequence: sequence,
		}
		sigs = append(sigs, sigData)
	}

	err := txBuilder.SetSignatures(sigs...)
	if err != nil {
		return nil, err
	}

	gas, err := SimulateTx(c.Conn, txConfig, txBuilder)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Gas used: %d\n", gas)

	// Update the gas limit to be 50% more than the estimated gas
	gasLimit := uint64(float64(gas) * 1.5)
	txBuilder.SetGasLimit(gasLimit)

	feeAmt := int64(float64(gasLimit) * float64(c.BlockchainConfig.GasPrice()))
	fee := sdk.NewCoins(sdk.NewInt64Coin(c.BlockchainConfig.Denom(), feeAmt+additionalFee))
	txBuilder.SetFeeAmount(fee)

	sigs = []signing.SignatureV2{}
	for _, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       c.BlockchainConfig.ChainID(),
			AccountNumber: accountNumber,
			Sequence:      sequence,
		}

		sigV2, err := tx.SignWithPrivKey(ctx, signing.SignMode_SIGN_MODE_DIRECT, signerData, txBuilder, priv, txConfig, sequence)
		if err != nil {
			return nil, err
		}
		sigs = append(sigs, sigV2)
	}

	// Set the signature
	if err := txBuilder.SetSignatures(sigs...); err != nil {
		return nil, err
	}

	// Encode the transaction
	txBz, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	// For now, just print the transaction bytes
	// In a real implementation, you would broadcast this transaction
	fmt.Printf("Transaction bytes: %x\n", txBz)
	fmt.Println("Transaction ready for broadcasting")

	txJSONBz, err := txConfig.TxJSONEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}
	fmt.Println("Transaction JSON:")
	txJSON := string(txJSONBz)
	fmt.Println(txJSON)

	return txBz, nil
}

func (grpcConn *GRPCConnection) BroadcastTx(txBytes []byte) (*txtypes.BroadcastTxResponse, error) {
	txClient := txtypes.NewServiceClient(grpcConn.Conn)

	resp, err := txClient.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC, // or BLOCK/ASYNC
	})
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast: %w", err)
	}

	return resp, nil
}
