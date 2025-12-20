package provenance

type BlockchainConfig struct {
	uri           string
	tls           bool
	addressPrefix string
	publicPrefix  string
	coinType      uint32
	chainID       string
	gasPrice      int64
	denom         string
}

type BlockchainConfigProvider interface {
	URI() string
	TLS() bool
	AddressPrefix() string
	PublicPrefix() string
	CoinType() uint32
	ChainID() string
	GasPrice() int64
	Denom() string
	NodeUrl(url string)
	Mainnet() bool
}

// Verify that the BlockchainConfig implement the BlockchainConfigProvider interface
var _ BlockchainConfigProvider = (*BlockchainConfig)(nil)

func NewMainnetConfig() *BlockchainConfig {
	return &BlockchainConfig{
		uri:           "grpc.provenance.io:443",
		tls:           true,
		addressPrefix: "pb",
		publicPrefix:  "pbpub",
		coinType:      505,
		chainID:       "pio-mainnet-1",
		gasPrice:      1,
		denom:         "nhash",
	}
}

func NewTestnetConfig() *BlockchainConfig {
	return &BlockchainConfig{
		uri:           "grpc.test.provenance.io:443",
		tls:           true,
		addressPrefix: "tp",
		publicPrefix:  "tppub",
		coinType:      1,
		chainID:       "pio-testnet-1",
		gasPrice:      1,
		denom:         "nhash",
	}
}

func (c *BlockchainConfig) URI() string {
	return c.uri
}

func (c *BlockchainConfig) TLS() bool {
	return c.tls
}

func (c *BlockchainConfig) AddressPrefix() string {
	return c.addressPrefix
}

func (c *BlockchainConfig) PublicPrefix() string {
	return c.publicPrefix
}

func (c *BlockchainConfig) CoinType() uint32 {
	return c.coinType
}

func (c *BlockchainConfig) ChainID() string {
	return c.chainID
}

func (c *BlockchainConfig) GasPrice() int64 {
	return c.gasPrice
}

func (c *BlockchainConfig) Denom() string {
	return c.denom
}

func (c *BlockchainConfig) NodeUrl(url string) {
	c.uri = url
}

func (c *BlockchainConfig) Mainnet() bool {
	return true
}
