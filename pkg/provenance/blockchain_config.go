package provenance

type BlockchainConfig struct {
	URI           string
	CertFile      string
	AddressPrefix string
	PublicPrefix  string
	CoinType      uint32
	ChainID       string
	GasPrice      int64
	Denom         string
}

type BlockchainConfigProvider interface {
	URI() string
	CertFile() string
	AddressPrefix() string
	PublicPrefix() string
	CoinType() uint32
	ChainID() string
	GasPrice() int64
	Denom() string
	NodeUrl(url string)
	Insecure()
	Mainnet() bool
}

// Verify that the MainnetConfig and TestnetConfig implement the BlockchainConfigProvider interface
var _ BlockchainConfigProvider = (*MainnetConfig)(nil)
var _ BlockchainConfigProvider = (*TestnetConfig)(nil)

type MainnetConfig struct {
	BlockchainConfig
}

func NewMainnetConfig() *MainnetConfig {
	return &MainnetConfig{
		BlockchainConfig: BlockchainConfig{
			URI:           "grpc.provenance.io:443",
			CertFile:      "certs/grpc.provenance.crt",
			AddressPrefix: "pb",
			PublicPrefix:  "pbpub",
			CoinType:      505,
			ChainID:       "pio-mainnet-1",
			GasPrice:      1,
			Denom:         "nhash",
		},
	}
}

func (c *MainnetConfig) URI() string {
	return c.BlockchainConfig.URI
}

func (c *MainnetConfig) CertFile() string {
	return c.BlockchainConfig.CertFile
}

func (c *MainnetConfig) AddressPrefix() string {
	return c.BlockchainConfig.AddressPrefix
}

func (c *MainnetConfig) PublicPrefix() string {
	return c.BlockchainConfig.PublicPrefix
}

func (c *MainnetConfig) CoinType() uint32 {
	return c.BlockchainConfig.CoinType
}

func (c *MainnetConfig) ChainID() string {
	return c.BlockchainConfig.ChainID
}

func (c *MainnetConfig) GasPrice() int64 {
	return c.BlockchainConfig.GasPrice
}

func (c *MainnetConfig) Denom() string {
	return c.BlockchainConfig.Denom
}

func (c *MainnetConfig) NodeUrl(url string) {
	c.BlockchainConfig.URI = url
}

func (c *MainnetConfig) Insecure() {
	c.BlockchainConfig.CertFile = ""
}

func (c *MainnetConfig) Mainnet() bool {
	return true
}

type TestnetConfig struct {
	BlockchainConfig
}

func NewTestnetConfig() *TestnetConfig {
	return &TestnetConfig{
		BlockchainConfig: BlockchainConfig{
			URI:           "grpc.test.provenance.io:443",
			CertFile:      "certs/grpc.test.provenance.crt",
			AddressPrefix: "tp",
			PublicPrefix:  "tppub",
			CoinType:      1,
			ChainID:       "pio-testnet-1",
			GasPrice:      1,
			Denom:         "nhash",
		},
	}
}

func (c *TestnetConfig) URI() string {
	return c.BlockchainConfig.URI
}

func (c *TestnetConfig) CertFile() string {
	return c.BlockchainConfig.CertFile
}

func (c *TestnetConfig) AddressPrefix() string {
	return c.BlockchainConfig.AddressPrefix
}

func (c *TestnetConfig) PublicPrefix() string {
	return c.BlockchainConfig.PublicPrefix
}

func (c *TestnetConfig) CoinType() uint32 {
	return c.BlockchainConfig.CoinType
}

func (c *TestnetConfig) ChainID() string {
	return c.BlockchainConfig.ChainID
}

func (c *TestnetConfig) GasPrice() int64 {
	return c.BlockchainConfig.GasPrice
}

func (c *TestnetConfig) Denom() string {
	return c.BlockchainConfig.Denom
}

func (c *TestnetConfig) NodeUrl(url string) {
	c.BlockchainConfig.URI = url
}

func (c *TestnetConfig) Insecure() {
	c.BlockchainConfig.CertFile = ""
}

func (c *TestnetConfig) Mainnet() bool {
	return false
}
