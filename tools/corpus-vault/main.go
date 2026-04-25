// Package main is the corpus-vault server: an off-chain data store
// referenced on-chain by the x/private_corpus module on ZERONE.
//
// The server reads items from a local directory, signs manifest
// responses with the operator's private key, and serves both the
// signed manifest list and the raw item bytes over HTTP. The chain
// records nothing about the items themselves — only the vault's
// identity and the manifest content_hash. Verification of received
// items against the on-chain hash is the client's job; this server
// supplies the canonical signed bytes.
//
// See PROTOCOL.md (in x/private_corpus) for the wire protocol and
// README.md (in this directory) for setup.
package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"log"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, `usage: corpus-vault <command> [flags]

commands:
  serve       run the HTTP server
  genkey      generate an ed25519 operator keypair
  hash        compute the content_hash of a manifest's item directory
  pubkey      print the operator public key from a private key file

run "corpus-vault <command> -h" for command-specific flags.
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "serve":
		runServe(args)
	case "genkey":
		runGenKey(args)
	case "hash":
		runHash(args)
	case "pubkey":
		runPubkey(args)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "", "path to YAML config file (required)")
	_ = fs.Parse(args)
	if *configPath == "" {
		fs.Usage()
		os.Exit(2)
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}
	defer srv.Close()
	if err := srv.Run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func runGenKey(args []string) {
	fs := flag.NewFlagSet("genkey", flag.ExitOnError)
	out := fs.String("out", "", "path to write the private key PEM (required)")
	_ = fs.Parse(args)
	if *out == "" {
		fs.Usage()
		os.Exit(2)
	}
	pub, err := GenerateKeyPair(*out)
	if err != nil {
		log.Fatalf("generate key: %v", err)
	}
	fmt.Printf("private key written to: %s\n", *out)
	fmt.Printf("public key (operator_pubkey for MsgRegisterVault):\n  %s\n", EncodePubkey(pub))
}

func runHash(args []string) {
	fs := flag.NewFlagSet("hash", flag.ExitOnError)
	manifestID := fs.String("manifest-id", "", "manifest id (required)")
	vaultID := fs.String("vault-id", "", "vault id (required)")
	root := fs.String("dir", "", "directory of items (required)")
	_ = fs.Parse(args)
	if *manifestID == "" || *vaultID == "" || *root == "" {
		fs.Usage()
		os.Exit(2)
	}
	body, err := BuildManifestFromDir(*manifestID, *vaultID, *root, "/item/"+*manifestID+"/")
	if err != nil {
		log.Fatalf("build manifest: %v", err)
	}
	hash, err := ContentHash(body)
	if err != nil {
		log.Fatalf("content hash: %v", err)
	}
	fmt.Printf("manifest_id: %s\n", body.ManifestID)
	fmt.Printf("vault_id:    %s\n", body.VaultID)
	fmt.Printf("items:       %d\n", len(body.Items))
	fmt.Printf("content_hash (publish on-chain): %s\n", hash)
}

func runPubkey(args []string) {
	fs := flag.NewFlagSet("pubkey", flag.ExitOnError)
	keyPath := fs.String("key", "", "path to private key PEM (required)")
	_ = fs.Parse(args)
	if *keyPath == "" {
		fs.Usage()
		os.Exit(2)
	}
	priv, err := LoadPrivateKey(*keyPath)
	if err != nil {
		log.Fatalf("load private key: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	fmt.Println(EncodePubkey(pub))
}
