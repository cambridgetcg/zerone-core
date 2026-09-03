// zerone-component-signature-verifier verifies a release component's local
// Sigstore message-signature bundle and emits authenticated signing time.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/zerone-chain/zerone/tools/sigstore-substrate-compiler/verification"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "zerone-component-signature-verifier: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("zerone-component-signature-verifier", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var policy verification.ComponentPolicy
	flags.StringVar(&policy.BundlePath, "bundle", "", "path to a local Sigstore bundle JSON file (required)")
	flags.StringVar(&policy.TrustedRootPath, "trusted-root", "", "path to a pinned local Sigstore trusted-root JSON file (required)")
	flags.StringVar(&policy.CertificateIssuer, "certificate-issuer", "", "exact Fulcio OIDC certificate issuer (required; no regex)")
	flags.StringVar(&policy.CertificateSAN, "certificate-san", "", "exact signing certificate SAN (required; no regex)")
	flags.StringVar(&policy.SourceRepositoryDigest, "source-repository-digest", "", "exact 40-hex source commit from the Fulcio certificate (required)")
	flags.StringVar(&policy.ArtifactDigest, "artifact-digest", "", "required artifact digest in sha256:<64 lowercase hex> form")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not accepted; use named flags")
	}

	result, err := verification.VerifyComponent(policy)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode component verification result: %w", err)
	}
	return nil
}
