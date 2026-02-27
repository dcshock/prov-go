package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dcshock/prov-go/pkg/provenance"
	registry "github.com/provenance-io/provenance/x/registry/types"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "generate":
		generateCmd(os.Args[2:])
	case "load":
		loadCmd(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `registry-loader usage:

  registry-loader generate -input-json entries.json -out entries.b64
      Reads a JSON file containing a slice of RegistryEntry objects and writes a
      base64-encoded representation to the specified output file.

  registry-loader load [-file] entries.b64 -mnemonic-file mnemonic.txt [-network mainnet|testnet]
      Reads a base64-encoded slice of RegistryEntry objects from the file,
      loads a signer key from the mnemonic file, connects to Provenance, and
      executes RegistryBulkUpdate. Use -network testnet for testnet (default: mainnet).

Notes:
  - The JSON structure for RegistryEntry must match the fields defined in
    github.com/provenance-io/provenance/x/registry/types.RegistryEntry.

`)
}

// generateCmd reads a JSON file of []registry.RegistryEntry and writes a
// base64-encoded JSON representation to an output file. Uses the provenance
// codec so the key field (and other proto fields) round-trip correctly.
func generateCmd(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	inputPath := fs.String("input-json", "", "Path to JSON file containing []RegistryEntry")
	outPath := fs.String("out", "registry_entries.b64", "Output file path for base64-encoded entries")
	_ = fs.Parse(args)

	if *inputPath == "" {
		log.Fatalf("input-json is required")
	}

	data, err := os.ReadFile(*inputPath)
	if err != nil {
		log.Fatalf("error reading input JSON: %v", err)
	}

	// Parse as JSON array of raw objects so we can codec-unmarshal each entry (key field round-trips).
	var rawEntries []json.RawMessage
	if err := json.Unmarshal(data, &rawEntries); err != nil {
		log.Fatalf("error unmarshalling JSON array: %v", err)
	}

	cdc := provenance.Codec()
	entries := make([]registry.RegistryEntry, 0, len(rawEntries))
	for i, raw := range rawEntries {
		var e registry.RegistryEntry
		if err := cdc.UnmarshalJSON(raw, &e); err != nil {
			log.Fatalf("error unmarshalling entry %d: %v", i+1, err)
		}
		entries = append(entries, e)
	}

	// Build output JSON array using codec per entry so key is serialized correctly.
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := range entries {
		if i > 0 {
			buf.WriteByte(',')
		}
		entryBytes, err := cdc.MarshalJSON(&entries[i])
		if err != nil {
			log.Fatalf("error marshalling entry %d: %v", i+1, err)
		}
		buf.Write(entryBytes)
	}
	buf.WriteByte(']')

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	if err := os.WriteFile(*outPath, []byte(encoded), 0o644); err != nil {
		log.Fatalf("error writing base64 output file: %v", err)
	}

	fmt.Printf("Wrote base64-encoded RegistryEntry slice to %s\n", *outPath)
}

// loadCmd reads a base64-encoded []RegistryEntry from a file, loads a signer key,
// connects to Provenance, and executes RegistryBulkUpdate.
func loadCmd(args []string) {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to base64 file containing []RegistryEntry")
	mnemonicFile := fs.String("mnemonic-file", "", "Path to mnemonic file for signer key")
	network := fs.String("network", "mainnet", "Network: mainnet or testnet")
	_ = fs.Parse(args)

	if *filePath == "" && len(fs.Args()) > 0 {
		*filePath = fs.Args()[0]
	}
	if *filePath == "" {
		log.Fatalf("file is required (use -file path or pass path as first argument)")
	}
	if *mnemonicFile == "" {
		log.Fatalf("mnemonic-file is required")
	}

	encoded, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("error reading base64 entries file: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		log.Fatalf("error decoding base64 entries: %v", err)
	}

	// Unmarshal JSON array of objects, then codec-unmarshal each so key field is populated.
	var rawEntries []json.RawMessage
	if err := json.Unmarshal(decoded, &rawEntries); err != nil {
		log.Fatalf("error unmarshalling decoded JSON array: %v", err)
	}

	cdc := provenance.Codec()
	entries := make([]registry.RegistryEntry, 0, len(rawEntries))
	for i, raw := range rawEntries {
		var e registry.RegistryEntry
		if err := cdc.UnmarshalJSON(raw, &e); err != nil {
			log.Fatalf("error unmarshalling entry %d: %v", i+1, err)
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		log.Fatalf("no registry entries found in decoded data")
	}

	// Choose blockchain configuration
	var cfg *provenance.BlockchainConfig
	switch *network {
	case "mainnet":
		cfg = provenance.NewMainnetConfig()
	case "testnet":
		cfg = provenance.NewTestnetConfig()
	default:
		log.Fatalf("invalid network %q: must be mainnet or testnet", *network)
	}

	// The NewProvenanceClient helper will read the mnemonic file, derive the key,
	// and populate the ProvenanceClient with signer details.
	p, err := provenance.NewProvenanceClient(cfg, mnemonicFile)
	if err != nil {
		log.Fatalf("error creating provenance client: %v", err)
	}
	defer p.Close()

	const maxPerTx = 15
	for i := 0; i < len(entries); i += maxPerTx {
		end := i + maxPerTx
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[i:end]
		resp, err := p.RegistryBulkUpdate(batch)
		if err != nil {
			log.Fatalf("error executing RegistryBulkUpdate (batch %d-%d): %v", i+1, end, err)
		}
		fmt.Printf("RegistryBulkUpdate batch %d-%d: %s\n", i+1, end, resp)
	}
}

