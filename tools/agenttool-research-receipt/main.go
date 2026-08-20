// Command agenttool-research-receipt independently compiles one exact local
// AgentTool Research Commons shadow settlement/public projection pair into a
// deterministic, zero-effect Zerone compatibility receipt.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"

	"golang.org/x/sys/unix"

	"github.com/zerone-chain/zerone/tools/agenttool-research-receipt/bridge"
)

const (
	maxReceiptInputBytes = 64 << 10
	maxTreeInputBytes    = 256 << 10
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "agenttool-research-receipt: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("agenttool-research-receipt", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var settlementPath string
	var projectionPath string
	var treePath string
	flags.StringVar(&settlementPath, "settlement", "", "path to an exact local AgentTool research settlement bundle (required)")
	flags.StringVar(&projectionPath, "projection", "", "path to its exact local digest-only public projection (required)")
	flags.StringVar(&treePath, "tree", "", "path to the exact local Constructive Intelligence Tree v1 artifact (required)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not accepted; use named flags")
	}
	if settlementPath == "" || projectionPath == "" || treePath == "" {
		return errors.New("--settlement, --projection, and --tree are required")
	}

	settlement, err := readBoundedRegularFile(settlementPath, maxReceiptInputBytes)
	if err != nil {
		return fmt.Errorf("read settlement: %w", err)
	}
	projection, err := readBoundedRegularFile(projectionPath, maxReceiptInputBytes)
	if err != nil {
		return fmt.Errorf("read projection: %w", err)
	}
	tree, err := readBoundedRegularFile(treePath, maxTreeInputBytes)
	if err != nil {
		return fmt.Errorf("read tree: %w", err)
	}

	receipt, err := bridge.Evaluate(settlement, projection, tree)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	return nil
}

func readBoundedRegularFile(path string, maximum int64) ([]byte, error) {
	return readBoundedRegularFileWithHook(path, maximum, nil)
}

// readBoundedRegularFileWithHook exists so tests can deterministically replace
// the pathname after the no-follow descriptor has opened. Production always
// calls readBoundedRegularFile, which supplies no hook.
func readBoundedRegularFileWithHook(path string, maximum int64, afterOpenForTest func() error) ([]byte, error) {
	if maximum < 0 {
		return nil, fmt.Errorf("%s has an invalid negative byte limit", path)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() < 0 || info.Size() > maximum {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, maximum)
	}
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file == nil {
		_ = unix.Close(descriptor)
		return nil, fmt.Errorf("%s could not be represented as an opened file", path)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || !sameFileMetadata(info, openedInfo) {
		return nil, fmt.Errorf("%s changed while it was being opened", path)
	}
	if openedInfo.Size() < 0 || openedInfo.Size() > maximum {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, maximum)
	}
	if afterOpenForTest != nil {
		if err := afterOpenForTest(); err != nil {
			return nil, fmt.Errorf("test hook after opening %s: %w", path, err)
		}
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, maximum)
	}
	// file.Stat is an fstat of the still-open descriptor. It must identify the
	// same inode with the same mode, size, mtime, and ctime observed before the
	// read, and the byte count must agree with that descriptor snapshot.
	afterReadInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !sameFileMetadata(openedInfo, afterReadInfo) || afterReadInfo.Size() != int64(len(data)) {
		return nil, fmt.Errorf("%s changed while it was being read", path)
	}
	// Re-resolve the final pathname after reading. A rename/swap must not let an
	// authenticated descriptor stand in for different bytes now named by path.
	finalPathInfo, err := os.Lstat(path)
	if err != nil || !finalPathInfo.Mode().IsRegular() || !sameFileMetadata(afterReadInfo, finalPathInfo) {
		return nil, fmt.Errorf("%s path changed while it was being read", path)
	}
	return data, nil
}

func sameFileMetadata(before, after os.FileInfo) bool {
	beforeCTimeSeconds, beforeCTimeNanoseconds, beforeHasCTime := fileChangeTime(before)
	afterCTimeSeconds, afterCTimeNanoseconds, afterHasCTime := fileChangeTime(after)
	return beforeHasCTime &&
		afterHasCTime &&
		os.SameFile(before, after) &&
		before.Mode() == after.Mode() &&
		before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) &&
		beforeCTimeSeconds == afterCTimeSeconds &&
		beforeCTimeNanoseconds == afterCTimeNanoseconds
}

// os.FileInfo exposes ctime only through its platform Stat_t. Supported Unix
// targets name that timespec Ctim or Ctimespec. Refusing an unknown shape is
// safer than silently dropping the ctime invariant.
func fileChangeTime(info os.FileInfo) (seconds int64, nanoseconds int64, ok bool) {
	value := reflect.ValueOf(info.Sys())
	if !value.IsValid() {
		return 0, 0, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, 0, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, 0, false
	}
	for _, fieldName := range []string{"Ctim", "Ctimespec"} {
		timespec := value.FieldByName(fieldName)
		if !timespec.IsValid() || timespec.Kind() != reflect.Struct {
			continue
		}
		secondsField := timespec.FieldByName("Sec")
		nanosecondsField := timespec.FieldByName("Nsec")
		if secondsField.IsValid() && secondsField.CanInt() && nanosecondsField.IsValid() && nanosecondsField.CanInt() {
			return secondsField.Int(), nanosecondsField.Int(), true
		}
	}
	return 0, 0, false
}
