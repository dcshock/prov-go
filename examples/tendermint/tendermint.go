package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dcshock/prov-go/pkg/provenance"
)

func main() {
	// Connect to a provenance grpc endpoint.
	p, err := provenance.NewProvenanceClient(provenance.NewMainnetConfig(), nil)
	if err != nil {
		log.Fatalf("error creating provenance client: %v", err)
	}
	defer p.Close()

	nodeConfig, err := p.GetNodeConfig(context.Background())
	if err != nil {
		log.Fatalf("error getting node config: %v", err)
	}
	fmt.Println("node config:", nodeConfig)
}
