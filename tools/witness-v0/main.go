package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/zerone-chain/zerone/tools/witness-v0/protocol"
)

const usage = `witness-v0: deterministic offline verifier for kingdom.witnessed-agent-economy/0.1

Usage:
  witness-v0 verify <record.json|->
  witness-v0 simulate <records.json|->
  witness-v0 merkle <settlement-batch.json|->
  witness-v0 activation-audit <record.json|->
  witness-v0 schema-hashes

The tool performs no network requests, clock reads, randomness, persistence,
Zerone transactions, external receipts, NEN invocations, or scoring.`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "witness-v0:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing command\n%s", usage)
	}
	switch args[0] {
	case "verify":
		if len(args) != 2 {
			return fmt.Errorf("verify requires exactly one input path")
		}
		input, err := readInput(args[1], stdin)
		if err != nil {
			return err
		}
		verified, err := protocol.Verify(input)
		if err != nil {
			return err
		}
		result := struct {
			Protocol       string           `json:"protocol"`
			Kind           protocol.Kind    `json:"kind"`
			Action         protocol.Action  `json:"action"`
			Audience       string           `json:"audience"`
			SubjectRef     string           `json:"subject_ref"`
			Sequence       string           `json:"sequence"`
			SchemaHash     string           `json:"schema_hash"`
			PayloadRoot    string           `json:"payload_root"`
			Commitment     string           `json:"commitment"`
			SignatureValid bool             `json:"signature_valid"`
			Effects        protocol.Effects `json:"effects"`
		}{
			Protocol: protocol.Protocol, Kind: verified.Record.Envelope.Kind, Action: verified.Record.Envelope.Action,
			Audience: verified.Record.Envelope.Audience, SubjectRef: verified.Record.Envelope.SubjectRef,
			Sequence: verified.Record.Envelope.Sequence, SchemaHash: verified.Record.Envelope.SchemaHash,
			PayloadRoot: verified.Record.Envelope.PayloadRoot, Commitment: verified.Record.Commitment,
			SignatureValid: true, Effects: verified.Record.Envelope.Effects,
		}
		return writeCanonical(stdout, result)

	case "simulate":
		if len(args) != 2 {
			return fmt.Errorf("simulate requires exactly one input path")
		}
		input, err := readInput(args[1], stdin)
		if err != nil {
			return err
		}
		result, err := protocol.Simulate(input)
		if err != nil {
			return err
		}
		canonical, err := result.CanonicalJSON()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(canonical))
		return err

	case "merkle":
		if len(args) != 2 {
			return fmt.Errorf("merkle requires exactly one input path")
		}
		input, err := readInput(args[1], stdin)
		if err != nil {
			return err
		}
		batch, root, err := protocol.VerifySettlementBatch(input)
		if err != nil {
			return err
		}
		result := struct {
			Algorithm     string `json:"algorithm"`
			LeafDomain    string `json:"leaf_domain"`
			FirstSequence string `json:"first_sequence"`
			LastSequence  string `json:"last_sequence"`
			ReceiptCount  string `json:"receipt_count"`
			MerkleRoot    string `json:"merkle_root"`
		}{"RFC6962_SHA256", protocol.Protocol + "\\0settlement-leaf\\0", batch.FirstSequence, batch.LastSequence, batch.ReceiptCount, root}
		return writeCanonical(stdout, result)

	case "activation-audit":
		if len(args) != 2 {
			return fmt.Errorf("activation-audit requires exactly one input path")
		}
		input, err := readInput(args[1], stdin)
		if err != nil {
			return err
		}
		verified, err := protocol.Verify(input)
		if err != nil {
			return err
		}
		return writeCanonical(stdout, protocol.AuditActivation(*verified))

	case "schema-hashes":
		if len(args) != 1 {
			return fmt.Errorf("schema-hashes takes no arguments")
		}
		hashes, err := protocol.SchemaHashes()
		if err != nil {
			return err
		}
		type entry struct {
			Kind       protocol.Kind `json:"kind"`
			SchemaHash string        `json:"schema_hash"`
		}
		entries := make([]entry, 0, len(hashes))
		for kind, hash := range hashes {
			entries = append(entries, entry{kind, hash})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Kind < entries[j].Kind })
		recordSchemaHash, err := protocol.RecordSchemaHash()
		if err != nil {
			return err
		}
		batchSchemaHash, err := protocol.SettlementBatchSchemaHash()
		if err != nil {
			return err
		}
		schemaSetDigest, err := protocol.SchemaSetDigest()
		if err != nil {
			return err
		}
		return writeCanonical(stdout, struct {
			Protocol                  string  `json:"protocol"`
			RecordSchemaHash          string  `json:"record_schema_hash"`
			SettlementBatchSchemaHash string  `json:"settlement_batch_schema_hash"`
			SchemaSetDigest           string  `json:"schema_set_digest"`
			Schemas                   []entry `json:"payload_schemas"`
		}{protocol.Protocol, recordSchemaHash, batchSchemaHash, schemaSetDigest, entries})

	case "help", "-h", "--help":
		_, err := fmt.Fprintln(stdout, usage)
		return err
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usage)
	}
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		contents, err := io.ReadAll(io.LimitReader(stdin, protocol.MaxDocumentBytes+1))
		if err != nil {
			return nil, err
		}
		if len(contents) > protocol.MaxDocumentBytes {
			return nil, fmt.Errorf("stdin exceeds %d bytes", protocol.MaxDocumentBytes)
		}
		return contents, nil
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing symlink input %q", path)
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("input %q is not a regular file", path)
	}
	if before.Size() < 0 || before.Size() > protocol.MaxDocumentBytes {
		return nil, fmt.Errorf("input %q exceeds %d bytes", path, protocol.MaxDocumentBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() != before.Size() || !opened.ModTime().Equal(before.ModTime()) {
		return nil, fmt.Errorf("input %q changed before open", path)
	}
	contents, err := io.ReadAll(io.LimitReader(file, protocol.MaxDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if len(contents) > protocol.MaxDocumentBytes || int64(len(contents)) != before.Size() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return nil, fmt.Errorf("input %q changed while reading", path)
	}
	return contents, nil
}

func writeCanonical(out io.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := protocol.CanonicalJSON(encoded)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(canonical))
	return err
}
