package provenance

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	attrtypes "github.com/provenance-io/provenance/x/attribute/types"
	marker "github.com/provenance-io/provenance/x/marker/types"
	meta "github.com/provenance-io/provenance/x/metadata/types"
	registry "github.com/provenance-io/provenance/x/registry/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCConnection struct {
	Conn *grpc.ClientConn
}

func NewGRPCConnection(uri string, secure bool) (*GRPCConnection, error) {
	// Use system TLS credentials for gRPCs connections, skipping certFile check
	// For insecure connections, use insecure credentials
	var creds credentials.TransportCredentials
	if secure {
		creds = credentials.NewTLS(nil)
	} else {
		creds = insecure.NewCredentials()
	}

	grpcConn, err := grpc.NewClient(uri, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	g := GRPCConnection{
		Conn: grpcConn,
	}

	return &g, nil
}

func (c *GRPCConnection) Close() error {
	return c.Conn.Close()
}

func Codec() *codec.ProtoCodec {
	reg := codectypes.NewInterfaceRegistry()

	cryptocodec.RegisterInterfaces(reg)
	marker.RegisterInterfaces(reg)
	meta.RegisterInterfaces(reg)
	banktypes.RegisterInterfaces(reg)
	attrtypes.RegisterInterfaces(reg)
	registry.RegisterInterfaces(reg)

	cdc := codec.NewProtoCodec(reg)
	return cdc
}

// Create a new tx config with the appropriate provenance interfaces registered
func NewTxConfig() client.TxConfig {
	txConfig := authtx.NewTxConfig(Codec(), authtx.DefaultSignModes)
	return txConfig
}
