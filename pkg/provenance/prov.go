package provenance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	nodetypes "cosmossdk.io/api/cosmos/base/node/v1beta1"
	tendermint "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	stakingtypes "cosmossdk.io/api/cosmos/staking/v1beta1"
	"github.com/cosmos/cosmos-sdk/codec"
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
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

type ProvenanceClient struct {
	Grpc          *GRPCConnection
	Signer        Signer
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
	nodeClient       *nodetypes.ServiceClient
	stakingClient    *stakingtypes.QueryClient
}

func (c *ProvenanceClient) NextSequence() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentSequence := c.Sequence
	c.Sequence++

	return currentSequence
}

// ResetSequence sets c.Sequence to the on-chain sequence from GetAccountInfo for this client's address.
// It must not hold c.mu while querying: GetAccountInfo uses AuthClient, which also locks c.mu.
func (c *ProvenanceClient) ResetSequence() (accountNumber uint64, sequence uint64, err error) {
	if c.Address == "" {
		return 0, 0, fmt.Errorf("provenance client has no address")
	}
	accountNumber, sequence, err = c.GetAccountInfo(c.Address)
	if err != nil {
		return 0, 0, err
	}
	c.mu.Lock()
	c.Sequence = sequence
	c.mu.Unlock()
	return accountNumber, sequence, nil
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

		privKey, err := PrivKeyFromMnemonic(blockchainConfig, *mnemonic)
		if err != nil {
			return nil, fmt.Errorf("error deriving key from mnemonic: %w", err)
		}
		signer, err := NewPrivKeySigner(privKey)
		if err != nil {
			return nil, err
		}
		if err := config.SetSigner(signer); err != nil {
			return nil, err
		}
	}

	pool, _ := ants.NewPool(50)
	config.Pool = pool

	return &config, nil
}

// NewProvenanceClientWithSigner connects like NewProvenanceClient, then binds
// signer. The signer supplies the account address and signatures, so private
// key bytes are not required.
func NewProvenanceClientWithSigner(blockchainConfig BlockchainConfigProvider, signer Signer) (*ProvenanceClient, error) {
	client, err := NewProvenanceClient(blockchainConfig, nil)
	if err != nil {
		return nil, err
	}
	if err := client.SetSigner(signer); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

// SetSigner binds signer and loads that account's number and sequence.
// Address is taken from the signer (or from its public key when Address is empty).
func (c *ProvenanceClient) SetSigner(signer Signer) error {
	if signer == nil {
		return fmt.Errorf("signer is nil")
	}
	addr, err := signerAccountAddress(signer)
	if err != nil {
		return err
	}
	c.Signer = signer
	c.Address = addr

	accountNumber, sequence, err := c.ResetSequence()
	if err != nil {
		return fmt.Errorf("error getting account info: %w", err)
	}
	c.AccountNumber = accountNumber
	c.Sequence = sequence
	return nil
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

// Node client
func (c *ProvenanceClient) NodeClient() *nodetypes.ServiceClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodeClient == nil {
		qc := nodetypes.NewServiceClient(c.Grpc.Conn)
		c.nodeClient = &qc
	}
	return c.nodeClient
}

// Staking client
func (c *ProvenanceClient) StakingClient() *stakingtypes.QueryClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stakingClient == nil {
		qc := stakingtypes.NewQueryClient(c.Grpc.Conn)
		c.stakingClient = &qc
	}
	return c.stakingClient
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

// SignBytes signs msg with the client's Signer and returns the raw signature.
// It does not build or broadcast a Provenance transaction.
func (c *ProvenanceClient) SignBytes(msg []byte) ([]byte, error) {
	if c.Signer == nil {
		return nil, fmt.Errorf("provenance client has no signer")
	}
	return c.Signer.Sign(msg)
}

func (c *ProvenanceClient) SignTx(msg []sdk.Msg, accountNumber, sequence uint64, additionalFee int64) ([]byte, error) {
	if c.Signer == nil {
		return nil, fmt.Errorf("provenance client has no signer")
	}
	pubKey := c.Signer.PubKey()
	if pubKey == nil {
		return nil, fmt.Errorf("signer has no public key")
	}

	ctx := context.Background()

	txConfig := NewTxConfig()

	// Add the msgs to the tx builder
	txBuilder := txConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg...); err != nil {
		return nil, err
	}

	fmt.Println("Signer address:", c.Address)

	// SIGN_MODE_DIRECT sign bytes include the signer info, so an empty
	// signature is set before simulation and before the real signature.
	emptySig := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: sequence,
	}
	if err := txBuilder.SetSignatures(emptySig); err != nil {
		return nil, err
	}

	feeAmount, _, err := SimulateTx(c.Grpc.Conn, txConfig, txBuilder)
	if err != nil {
		// Reset the sequence since we know that this one failed, and will cause downstream failures
		// in sequence number alignment.
		c.ResetSequence()
		return nil, err
	}
	fmt.Printf("Fee amount: %d\n", feeAmount)

	// Update the gas limit to be 50% more than the estimated gas
	fee := sdk.NewCoins(sdk.NewInt64Coin(c.BcConfig.Denom(), int64(feeAmount)+additionalFee))
	txBuilder.SetFeeAmount(fee)

	sigV2, err := signWithSigner(ctx, txConfig, txBuilder, c.Signer, c.BcConfig.ChainID(), accountNumber, sequence)
	if err != nil {
		return nil, err
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
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

// Wait on a broadcasted tx to complete
func (c *ProvenanceClient) WaitOnTx(txHash string) (*txtypes.GetTxResponse, error) {
	// return an error if there is no tx hash provided
	if txHash == "" {
		return nil, fmt.Errorf("no tx hash provided")
	}

	txClient := txtypes.NewServiceClient(c.Grpc.Conn)

	// Poll until a tx is indexed or ~20s elapses; transient GetTx RPC errors retry.
	for i := 0; i < 10; i++ {
		resp, err := txClient.GetTx(context.Background(), &txtypes.GetTxRequest{
			Hash: txHash,
		})
		if err == nil && resp != nil && resp.TxResponse != nil {
			return resp, nil
		}
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("get tx: gave up after timeout for hash %s", txHash)
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
