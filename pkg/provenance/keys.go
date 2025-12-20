package provenance

import (
	"fmt"
	"os"
	"strings"

	hd "github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptokey "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"github.com/cosmos/go-bip39"
)

func PrivKeyFromMnemonic(conf BlockchainConfigProvider, mnemonic string) (*cryptokey.PrivKey, error) {
	algo := hd.Secp256k1
	hdPath := hd.CreateHDPath(conf.CoinType(), 0, 0).String()

	derivedPriv, err := algo.Derive()(mnemonic, "", hdPath)
	if err != nil {
		return nil, err
	}

	priv := algo.Generate()(derivedPriv)
	return priv.(*cryptokey.PrivKey), nil
}

func ReadMnemonic(path string) (*string, error) {
	mnemonic, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Remove trailing newline
	mnemonicStr := strings.TrimSpace(string(mnemonic))

	if !bip39.IsMnemonicValid(mnemonicStr) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	return &mnemonicStr, nil
}
