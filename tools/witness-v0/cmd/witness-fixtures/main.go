package main

import (
	"fmt"
	"os"

	"github.com/zerone-chain/zerone/tools/witness-v0/fixturegen"
)

const usage = `witness-fixtures deterministically reproduces the witnessed-agent-economy v0 corpus.

Usage:
  witness-fixtures --write <explicit-directory>
  witness-fixtures --check <explicit-directory>

The generator uses fixed inputs and performs no network, clock, or random reads.
Only --write mutates storage, and it requires an explicit bounded destination.`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "witness-fixtures:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected --write or --check plus an explicit directory\n%s", usage)
	}
	switch args[0] {
	case "--write":
		corpus, err := fixturegen.WriteDir(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("wrote %d deterministic files\nschema_set_digest=%s\ncorpus_digest=%s\n", len(corpus.Files), corpus.Manifest.SchemaSetDigest, corpus.Manifest.CorpusDigest)
		return nil
	case "--check":
		if err := fixturegen.CheckDir(args[1]); err != nil {
			return err
		}
		corpus, err := fixturegen.Build()
		if err != nil {
			return err
		}
		fmt.Printf("fixture corpus matches\nschema_set_digest=%s\ncorpus_digest=%s\n", corpus.Manifest.SchemaSetDigest, corpus.Manifest.CorpusDigest)
		return nil
	default:
		return fmt.Errorf("unknown mode %q\n%s", args[0], usage)
	}
}
