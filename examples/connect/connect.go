package main

import (
	"fmt"
	"log"

	"github.com/dcshock/prov-go/pkg/provenance"
)

func main() {
	p, err := provenance.NewProvenanceClient(provenance.NewMainnetConfig(), nil)
	if err != nil {
		log.Fatalf("error creating provenance client: %v", err)
	}

	balances, err := p.GetBalance("pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9")
	if err != nil {
		log.Fatalf("error getting account info: %v", err)
	}

	fmt.Println("balances:", balances)
}
