package provenance

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ants "github.com/panjf2000/ants/v2"
	attrtypes "github.com/provenance-io/provenance/x/attribute/types"
	marker "github.com/provenance-io/provenance/x/marker/types"
	meta "github.com/provenance-io/provenance/x/metadata/types"
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
	mu            sync.Mutex

	authClient      *authtypes.QueryClient
	attributeClient *attrtypes.QueryClient
	bankClient      *banktypes.QueryClient
	markerClient    *marker.QueryClient
	metadataClient  *meta.QueryClient
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

func (c *ProvenanceClient) Close() {
	c.Grpc.Close()
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

func connect(conf BlockchainConfigProvider) (*GRPCConnection, error) {
	sdkConf := sdk.GetConfig()
	sdkConf.SetBech32PrefixForAccount(conf.AddressPrefix(), conf.PublicPrefix())
	sdkConf.SetCoinType(conf.CoinType())
	sdkConf.Seal()

	grpc, err := NewGRPCConnection(conf)
	if err != nil {
		return nil, fmt.Errorf("error creating gRPC connection: %w", err)
	}

	return grpc, nil
}
