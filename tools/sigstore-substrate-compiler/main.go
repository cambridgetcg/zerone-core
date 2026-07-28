// sigstore-substrate-compiler verifies a local Sigstore in-toto bundle and
// emits a deterministic, witness-only Zerone SubstrateLink.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/compile"
	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/verification"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "sigstore-substrate-compiler: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("sigstore-substrate-compiler", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var policy verification.Policy
	var sourceURL string
	var fetchedAtBlock uint64

	flags.StringVar(&policy.BundlePath, "bundle", "", "path to a local Sigstore bundle JSON file (required)")
	flags.StringVar(&policy.TrustedRootPath, "trusted-root", "", "path to a pinned local Sigstore trusted-root JSON file (required)")
	flags.StringVar(&policy.CertificateIssuer, "certificate-issuer", "", "exact Fulcio OIDC certificate issuer (required; no regex)")
	flags.StringVar(&policy.CertificateSAN, "certificate-san", "", "exact signing certificate SAN (required; no regex)")
	flags.StringVar(&policy.ArtifactDigest, "artifact-digest", "", "required artifact digest in sha256:<64 lowercase hex> form")
	flags.StringVar(&policy.PredicateType, "predicate-type", "", "exact required in-toto predicate type URI")
	flags.StringVar(&sourceURL, "source-url", "", "public HTTPS audit URL for the bundle")
	flags.Uint64Var(&fetchedAtBlock, "fetched-at-block", 0, "Zerone block height associated with the local fetch (optional)")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not accepted; use named flags")
	}

	verified, err := verification.VerifyBundle(policy)
	if err != nil {
		return err
	}
	link, err := compile.Compile(compile.Input{
		Attestation:    verified,
		SourceURL:      sourceURL,
		FetchedAtBlock: fetchedAtBlock,
	})
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(link); err != nil {
		return fmt.Errorf("encode substrate link: %w", err)
	}
	return nil
}
