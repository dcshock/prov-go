package provenance

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	marker "github.com/provenance-io/provenance/x/marker/types"

	ants "github.com/panjf2000/ants/v2"
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

	registry := codectypes.NewInterfaceRegistry()
	marker.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	config := ProvenanceClient{
		Grpc:     grpc,
		BcConfig: blockchainConfig,
		Cdc:      cdc,
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

// Returns the account number and sequence for the given address
func (c *ProvenanceClient) GetAccountInfo(address string) (uint64, uint64, error) {
	authClient := authtypes.NewQueryClient(c.Grpc.Conn)
	res, err := authClient.Account(context.Background(), &authtypes.QueryAccountRequest{
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
