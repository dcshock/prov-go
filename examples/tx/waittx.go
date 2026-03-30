package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dcshock/prov-go/pkg/provenance"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <tx-hash>", os.Args[0])
	}
	txHash := os.Args[1]

	p, err := provenance.NewProvenanceClient(provenance.NewMainnetConfig(), nil)
	if err != nil {
		log.Fatalf("error creating provenance client: %v", err)
	}
	defer p.Close()

	resp, err := p.WaitOnTx(txHash)
	if err != nil {
		log.Fatalf("wait on tx: %v", err)
	}

	if resp.TxResponse == nil {
		log.Fatal("empty tx response")
	}
	tr := resp.TxResponse
	fmt.Printf("tx %s included at height %d (code=%d)\n", tr.TxHash, tr.Height, tr.Code)
	if tr.RawLog != "" {
		fmt.Println("raw_log:", tr.RawLog)
	}
}
