package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/dcshock/prov-go/pkg/provenance"
)

func main() {
	mnemonicFile := flag.String("mnemonic-file", "", "Path to mnemonic file for the signing key")
	msg := flag.String("msg", "hello", "Message to sign")
	network := flag.String("network", "mainnet", "Network: mainnet or testnet")
	flag.Parse()

	if *mnemonicFile == "" {
		log.Fatalf("-mnemonic-file is required")
	}

	var cfg *provenance.BlockchainConfig
	switch *network {
	case "mainnet":
		cfg = provenance.NewMainnetConfig()
	case "testnet":
		cfg = provenance.NewTestnetConfig()
	default:
		log.Fatalf("invalid network %q: must be mainnet or testnet", *network)
	}

	p, err := provenance.NewProvenanceClient(cfg, mnemonicFile)
	if err != nil {
		log.Fatalf("error creating provenance client: %v", err)
	}
	defer p.Close()

	sig, err := p.SignBytes([]byte(*msg))
	if err != nil {
		log.Fatalf("error signing bytes: %v", err)
	}

	pubKeyBytes := p.PrivKey.PubKey().Bytes()

	fmt.Printf("address:          %s\n", p.Address)
	fmt.Printf("public key (hex): %s\n", hex.EncodeToString(pubKeyBytes))
	fmt.Printf("public key (b64): %s\n", base64.StdEncoding.EncodeToString(pubKeyBytes))
	fmt.Printf("message:          %q\n", *msg)
	fmt.Printf("signature (hex):  %s\n", hex.EncodeToString(sig))
	fmt.Printf("signature (b64):  %s\n", base64.StdEncoding.EncodeToString(sig))
}
