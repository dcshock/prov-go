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

	syncDenomBalance(p, "pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9", "nhash")
	syncBalances(p, "pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9")
	streamBalances(p, "pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9")
	syncBalancesWithHeight(p, "pb1pr93cqdh4kfnmrknhwa87a5qrwxw9k3dya4wr9", 28208262)
}

func syncDenomBalance(p *provenance.ProvenanceClient, address string, denom string) {
	balance, err := p.GetBalance(context.Background(), address, denom)
	if err != nil {
		log.Fatalf("error getting account info: %v", err)
	}
	fmt.Println("balance:", balance)
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
			if err != nil {
				log.Fatalf("error getting account info: %v", err)
			}
		}
	}
}

func syncBalancesWithHeight(p *provenance.ProvenanceClient, address string, height int64) {
	// Block height is set in the context for the gRPC call
	ctx := p.ContextWithBlockHeight(context.Background(), height)

	balances, err := p.GetBalances(ctx, address)
	if err != nil {
		log.Fatalf("error getting account info: %v", err)
	}
	fmt.Println("balances:", balances)
}
