// zerone-self-compiler turns a ZERONE git-commit SHA into a deterministic
// SubstrateLink for the zerone-self-v1 adapter, emitted as canonical JSON
// to stdout.
//
// Usage:
//
//	zerone-self-compiler <commit-sha>
//
// The output is exactly what a submitter would attest under the
// zerone-self-v1 adapter (see docs/specs/adapters/zerone-self-v1.md).
// Validators re-run this binary on the cited commit-sha to verify the
// submitted link matches (compiler-drift slash protection).
//
// Determinism: identical commit-sha in same git history → identical bytes
// out (down to the canonical-JSON byte order). Re-run anywhere with git
// access and the binary; result is bit-stable.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/zerone-chain/zerone/tools/zerone-self-compiler/compile"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: zerone-self-compiler <commit-sha>")
		os.Exit(2)
	}
	sha := strings.ToLower(strings.TrimSpace(os.Args[1]))

	meta, err := readCommitMeta(sha)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read commit %s: %v\n", sha, err)
		os.Exit(1)
	}

	link, err := compile.Compile(*meta, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

// readCommitMeta uses `git show` with a deterministic format to populate
// CommitMeta. The format is fixed so output is byte-stable across git
// versions that respect --format= (all modern versions).
func readCommitMeta(sha string) (*compile.CommitMeta, error) {
	// %H = full hash, %an = author name, %ae = author email, %cI = committer
	// date strict ISO 8601, %s = subject. Separator is \x1f (unit separator,
	// won't appear in any of these fields).
	cmd := exec.Command("git", "show", "--no-patch",
		"--format=%H\x1f%an\x1f%ae\x1f%cI\x1f%s",
		sha)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "\x1f")
	if len(parts) != 5 {
		return nil, fmt.Errorf("git show: expected 5 fields, got %d", len(parts))
	}
	t, err := time.Parse(time.RFC3339, parts[3])
	if err != nil {
		return nil, fmt.Errorf("parse date %q: %w", parts[3], err)
	}

	// Touched files: diff-tree gives one path per line without commit
	// header. Works for any commit (including merge/root via -r flag).
	cmd2 := exec.Command("git", "diff-tree", "--no-commit-id",
		"--name-only", "-r", sha)
	out2, err := cmd2.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree: %w", err)
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out2)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return &compile.CommitMeta{
		Hash:         strings.ToLower(parts[0]),
		Author:       fmt.Sprintf("%s <%s>", parts[1], parts[2]),
		Date:         t.UTC(),
		Subject:      parts[4],
		TouchedFiles: files,
	}, nil
}
