package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dcshock/prov-go/pkg/provenance"
)

func main() {
	p, err := provenance.NewProvenanceClient(provenance.NewMainnetConfig(), nil)
	if err != nil {
		log.Fatalf("error creating provenance client: %v", err)
	}
	defer p.Close()

	syncBalances(p, "pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9")

	streamBalances(p, "pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9")
}

func syncBalances(p *provenance.ProvenanceClient, address string) {
	balances, err := p.GetBalances(context.Background(), address)
	if err != nil {
		log.Fatalf("error getting account info: %v", err)
	}
	fmt.Println("balances:", balances)
}

func streamBalances(p *provenance.ProvenanceClient, address string) {
	balancesChan, errChan := p.GetBalancesStream(context.Background(), address)
	for {
		select {
		case balance, ok := <-balancesChan:
			if !ok {
				fmt.Println("no more balances")
				return
			}
			fmt.Println("balance:", balance)
		case err := <-errChan:
			log.Fatalf("error getting account info: %v", err)
		}
	}
}
