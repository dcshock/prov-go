package provenance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"google.golang.org/grpc/metadata"

	/// Blockchain Query Clients
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ants "github.com/panjf2000/ants/v2"
	attrtypes "github.com/provenance-io/provenance/x/attribute/types"
	marker "github.com/provenance-io/provenance/x/marker/types"
	meta "github.com/provenance-io/provenance/x/metadata/types"

	// Signing packages
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type ProvenanceClient struct {
	Grpc          *GRPCConnection
	PrivKey       *secp256k1.PrivKey
	BcConfig      BlockchainConfigProvider
	Cdc           *codec.ProtoCodec
	Address       string
	AccountNumber uint64
	Sequence      uint64
	Pool          *ants.Pool

	// Mutex for clients and sequence
	mu sync.Mutex

	// Blockchain Query Clients
	authClient       *authtypes.QueryClient
	attributeClient  *attrtypes.QueryClient
	bankClient       *banktypes.QueryClient
	markerClient     *marker.QueryClient
	metadataClient   *meta.QueryClient
	tendermintClient *tendermint.ServiceClient
}

func (c *ProvenanceClient) NextSequence() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentSequence := c.Sequence
	c.Sequence++

	return currentSequence
}

func NewProvenanceClient(blockchainConfig BlockchainConfigProvider, mnemonicFilePath *string) (*ProvenanceClient, error) {
	grpc, err := connect(blockchainConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating gRPC connection: %w", err)
	}

	config := ProvenanceClient{
		Grpc:     grpc,
		BcConfig: blockchainConfig,
		Cdc:      Codec(),
		mu:       sync.Mutex{},
	}

	if mnemonicFilePath != nil && strings.TrimSpace(*mnemonicFilePath) != "" {
		mnemonic, err := ReadMnemonic(*mnemonicFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading mnemonic: %w", err)
		}

		// Derive the private key from the mnemonic
		config.PrivKey, err = PrivKeyFromMnemonic(blockchainConfig, *mnemonic)
		if err != nil {
			return nil, fmt.Errorf("error deriving key from mnemonic: %w", err)
		}

		address := sdk.AccAddress(config.PrivKey.PubKey().Address()).String()
		accountNumber, sequence, err := config.GetAccountInfo(address)
		if err != nil {
			return nil, fmt.Errorf("error getting account info: %w", err)
		}

		config.Address = address
		config.AccountNumber = accountNumber
		config.Sequence = sequence
	}

	pool, _ := ants.NewPool(50)
	config.Pool = pool

	return &config, nil
}

func (c *ProvenanceClient) Close() {
	c.Grpc.Close()
}

func (c *ProvenanceClient) AuthClient() *authtypes.QueryClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.authClient == nil {
		qc := authtypes.NewQueryClient(c.Grpc.Conn)
		c.authClient = &qc
	}
	return c.authClient
}

// Attribute client
func (c *ProvenanceClient) AttributeClient() *attrtypes.QueryClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.attributeClient == nil {
		qc := attrtypes.NewQueryClient(c.Grpc.Conn)
		c.attributeClient = &qc
	}
	return c.attributeClient
}

func (c *ProvenanceClient) BankClient() *banktypes.QueryClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.bankClient == nil {
		qc := banktypes.NewQueryClient(c.Grpc.Conn)
		c.bankClient = &qc
	}
	return c.bankClient
}

// Marker client
func (c *ProvenanceClient) MarkerClient() *marker.QueryClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.markerClient == nil {
		qc := marker.NewQueryClient(c.Grpc.Conn)
		c.markerClient = &qc
	}
	return c.markerClient
}

// Metadata client
func (c *ProvenanceClient) MetadataClient() *meta.QueryClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.metadataClient == nil {
		qc := meta.NewQueryClient(c.Grpc.Conn)
		c.metadataClient = &qc
	}
	return c.metadataClient
}

// Tendermint client
func (c *ProvenanceClient) TendermintClient() *tendermint.ServiceClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tendermintClient == nil {
		qc := tendermint.NewServiceClient(c.Grpc.Conn)
		c.tendermintClient = &qc
	}
	return c.tendermintClient
}

// Returns the account number and sequence for the given address
func (c *ProvenanceClient) GetAccountInfo(address string) (uint64, uint64, error) {
	res, err := (*c.AuthClient()).Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return 0, 0, err
	}

	var baseAcc authtypes.BaseAccount
	if err := baseAcc.Unmarshal(res.Account.Value); err != nil {
		return 0, 0, err
	}

	return baseAcc.AccountNumber, baseAcc.Sequence, nil
}

func (c *ProvenanceClient) SignTx(msg []sdk.Msg, privKeyBytes []byte, accountNumber, sequence uint64, additionalFee int64) ([]byte, error) {
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

	gas, err := SimulateTx(c.Grpc.Conn, txConfig, txBuilder)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Gas used: %d\n", gas)

	// Update the gas limit to be 50% more than the estimated gas
	gasLimit := uint64(float64(gas) * 1.5)
	txBuilder.SetGasLimit(gasLimit)

	feeAmt := int64(float64(gasLimit) * float64(c.BcConfig.GasPrice()))
	fee := sdk.NewCoins(sdk.NewInt64Coin(c.BcConfig.Denom(), feeAmt+additionalFee))
	txBuilder.SetFeeAmount(fee)

	sigs = []signing.SignatureV2{}
	for _, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       c.BcConfig.ChainID(),
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

func (c *ProvenanceClient) BroadcastTx(txBytes []byte) (*txtypes.BroadcastTxResponse, error) {
	txClient := txtypes.NewServiceClient(c.Grpc.Conn)

	resp, err := txClient.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC, // or BLOCK/ASYNC
	})
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast: %w", err)
	}

	return resp, nil
}

// Context Set Block Height
func (c *ProvenanceClient) ContextWithBlockHeight(ctx context.Context, height int64) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-cosmos-block-height", strconv.FormatInt(height, 10))
}

func connect(conf BlockchainConfigProvider) (*GRPCConnection, error) {
	// Internal configuration for the SDK to know what prefix to use for the account and public keys
	sdkConf := sdk.GetConfig()
	sdkConf.SetBech32PrefixForAccount(conf.AddressPrefix(), conf.PublicPrefix())
	sdkConf.SetCoinType(conf.CoinType())
	sdkConf.Seal()

	grpc, err := NewGRPCConnection(conf.URI(), conf.TLS())
	if err != nil {
		return nil, fmt.Errorf("error creating gRPC connection: %w", err)
	}

	return grpc, nil
}
