// relaunch-accounting reads one explicitly pinned public snapshot. It performs
// custody arithmetic only: no provenance authentication or monetary action.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"syscall"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("relaunch-accounting", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var path, expected string
	for name, target := range map[string]*string{"snapshot": &path, "expected-sha256": &expected} {
		seen := false
		flags.Func(name, "required; supply exactly once", func(value string) error {
			if seen {
				return fmt.Errorf("flag supplied more than once")
			}
			seen = true
			*target = value
			return nil
		})
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if path == "" || path == "-" || !isSHA256(expected) || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: relaunch-accounting --snapshot PATH --expected-sha256 LOWERCASE64HEX (regular file only, no stdin)")
		return 2
	}
	data, err := readRegularFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "relaunch accounting: %v\n", err)
		return 1
	}
	report, err := accountSnapshot(data, expected)
	if err != nil {
		fmt.Fprintf(stderr, "relaunch accounting: %v\n", err)
		return 1
	}
	// All input and report validation finishes before the first stdout byte.
	if n, err := stdout.Write(append(report, '\n')); err != nil || n != len(report)+1 {
		fmt.Fprintln(stderr, "relaunch accounting: cannot write complete report")
		return 1
	}
	return 0
}

// Like validator-recovery-gate/canonical.go, use no-follow/nonblocking opens.
// Only the final component is protected from symlinks: ancestor directories
// belong to the caller's trusted local filesystem boundary, not discovery.
func readRegularFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot inspect snapshot")
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("snapshot must be a non-symlink regular file")
	}
	if before.Size() < 0 || before.Size() > maxInputBytes {
		return nil, fmt.Errorf("snapshot exceeds %d-byte limit", maxInputBytes)
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot open snapshot safely")
	}
	file := os.NewFile(uintptr(fd), "snapshot")
	if file == nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("cannot represent snapshot descriptor")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !sameFile(before, opened) || !opened.Mode().IsRegular() {
		return nil, fmt.Errorf("snapshot changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read snapshot")
	}
	if len(data) > maxInputBytes {
		return nil, fmt.Errorf("snapshot exceeds %d-byte limit", maxInputBytes)
	}
	after, err := file.Stat()
	if err != nil || !sameFile(opened, after) || after.Size() != int64(len(data)) {
		return nil, fmt.Errorf("snapshot changed while reading")
	}
	final, err := os.Lstat(path)
	if err != nil || !sameFile(after, final) || !final.Mode().IsRegular() {
		return nil, fmt.Errorf("snapshot path changed while reading")
	}
	return data, nil
}

func sameFile(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}
