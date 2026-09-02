package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	exitOK          = 0
	exitOperational = 1
	exitReconcile   = 2
)

type dbOpener func(home, backend string) (openedPhysicalDB, error)

type censusOptions struct {
	ChainID      string
	SourceCommit string
	Height       int64
	AppHash      []byte
}

type censusRunner func(db physicalDB, options censusOptions) ([]byte, bool, error)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, openCopiedApplicationDB, executeCensus))
}

func run(
	args []string,
	stdout, stderr io.Writer,
	open dbOpener,
	census censusRunner,
) int {
	if stdout == nil || stderr == nil {
		return exitOperational
	}
	if open == nil || census == nil {
		_, _ = fmt.Fprintln(stderr, "custom-staking-census: internal dependency is nil")
		return exitOperational
	}

	flags := flag.NewFlagSet("custom-staking-census", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var (
		home             string
		backend          string
		chainID          string
		heightText       string
		appHashText      string
		sourceCommit     string
		outputPath       string
		copiedDBAttested bool
	)
	flags.StringVar(&home, "home", "", "absolute home directory containing the copied data/application.db")
	flags.StringVar(&backend, "backend", "", "copied application DB backend: goleveldb or pebbledb")
	flags.StringVar(&chainID, "chain-id", "", "exact independently recorded chain ID")
	flags.StringVar(&heightText, "expected-height", "", "trusted positive root application height")
	flags.StringVar(&appHashText, "expected-app-hash", "", "trusted lowercase 32-byte post-commit root hash for H, in hexadecimal")
	flags.StringVar(&sourceCommit, "source-commit", "", "lowercase 40-hex source commit used to build this tool")
	flags.StringVar(&outputPath, "output", "", "absolute new report path; published atomically and never overwritten")
	flags.BoolVar(&copiedDBAttested, "copied-db", false, "attest that --home is a disposable copy made after the node was halted")
	if err := flags.Parse(args); err != nil {
		return exitOperational
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "custom-staking-census: unexpected positional arguments: %s\n", strings.Join(flags.Args(), " "))
		return exitOperational
	}
	if !copiedDBAttested {
		_, _ = fmt.Fprintln(stderr, "custom-staking-census: --copied-db is required; never point this tool at a live or only application database")
		return exitOperational
	}

	options, err := parseCensusOptions(chainID, heightText, appHashText, sourceCommit)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "custom-staking-census: %v\n", err)
		return exitOperational
	}

	db, err := open(home, backend)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "custom-staking-census: %v\n", err)
		return exitOperational
	}
	report, passed, censusErr := census(db, options)
	closeErr := db.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("close copied application DB: %w", closeErr)
	}
	if err := errors.Join(censusErr, closeErr); err != nil {
		_, _ = fmt.Fprintf(stderr, "custom-staking-census: %v\n", err)
		return exitOperational
	}
	if len(report) == 0 {
		_, _ = fmt.Fprintln(stderr, "custom-staking-census: census returned an empty report")
		return exitOperational
	}

	output := append(bytes.Clone(report), '\n')
	if outputPath != "" {
		if err := writeReportAtomically(home, outputPath, output); err != nil {
			_, _ = fmt.Fprintf(stderr, "custom-staking-census: publish report: %v\n", err)
			return exitOperational
		}
		if !passed {
			_, _ = fmt.Fprintln(stderr, "custom-staking-census: FAIL: legacy staking state is not automatically migratable")
			return exitReconcile
		}
		return exitOK
	}
	written, err := stdout.Write(output)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "custom-staking-census: write report: %v\n", err)
		return exitOperational
	}
	if written != len(output) {
		_, _ = fmt.Fprintf(stderr, "custom-staking-census: write report: %v\n", io.ErrShortWrite)
		return exitOperational
	}
	if !passed {
		_, _ = fmt.Fprintln(stderr, "custom-staking-census: FAIL: legacy staking state is not automatically migratable")
		return exitReconcile
	}
	return exitOK
}

func writeReportAtomically(sourceHome, outputPath string, contents []byte) error {
	if outputPath == "" || !filepath.IsAbs(outputPath) {
		return errors.New("--output must be an absolute path")
	}
	cleanPath := filepath.Clean(outputPath)
	if cleanPath == string(filepath.Separator) || filepath.Base(cleanPath) == "." {
		return errors.New("--output must name a file")
	}
	parent := filepath.Dir(cleanPath)
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("resolve report parent directory: %w", err)
	}
	parentInfo, err := os.Stat(resolvedParent)
	if err != nil {
		return fmt.Errorf("inspect report parent directory: %w", err)
	}
	if !parentInfo.IsDir() {
		return errors.New("report parent path is not a directory")
	}
	finalPath := filepath.Join(resolvedParent, filepath.Base(cleanPath))
	resolvedSourceHome, err := filepath.EvalSymlinks(filepath.Clean(sourceHome))
	if err != nil {
		return fmt.Errorf("resolve copied source home before publishing report: %w", err)
	}
	if pathIsEqualOrDescendant(resolvedSourceHome, finalPath) {
		return errors.New("--output must be outside the copied source home")
	}
	insideByIdentity, err := pathIsSameFileOrDescendant(resolvedSourceHome, resolvedParent)
	if err != nil {
		return fmt.Errorf("compare report parent to copied source home: %w", err)
	}
	if insideByIdentity {
		return errors.New("--output must be outside the copied source home")
	}
	if _, err := os.Lstat(finalPath); err == nil {
		return fmt.Errorf("report path %q already exists; refusing to overwrite evidence", finalPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect report output path: %w", err)
	}

	temporary, err := os.CreateTemp(resolvedParent, ".custom-staking-census-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary report: %w", err)
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary report permissions: %w", err)
	}
	if err := writeAll(temporary, contents); err != nil {
		return fmt.Errorf("write temporary report: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		closed = true
		return fmt.Errorf("close temporary report: %w", err)
	}
	closed = true
	// Linking a fully synced temporary file publishes the final name atomically
	// and fails if that name already exists. Both names are in one directory and
	// therefore on one filesystem.
	if err := os.Link(temporaryPath, finalPath); err != nil {
		return fmt.Errorf("publish new report without overwrite: %w", err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove temporary report link after publish: %w", err)
	}
	directory, err := os.Open(resolvedParent)
	if err != nil {
		return fmt.Errorf("open report directory for sync: %w", err)
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if err := errors.Join(syncErr, closeErr); err != nil {
		return fmt.Errorf("sync report directory: %w", err)
	}
	return nil
}

func writeAll(destination io.Writer, contents []byte) error {
	for len(contents) > 0 {
		written, err := destination.Write(contents)
		if written < 0 || written > len(contents) {
			return errors.New("writer returned an invalid byte count")
		}
		contents = contents[written:]
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func parseCensusOptions(
	chainID, heightText, appHashText, sourceCommit string,
) (censusOptions, error) {
	if !validChainID(chainID) {
		return censusOptions{}, errors.New("--chain-id must be 1-128 ASCII letters, digits, '.', '_', or '-', beginning with a letter or digit")
	}
	height, err := parseCanonicalPositiveHeight(heightText)
	if err != nil {
		return censusOptions{}, err
	}
	appHash, err := parseCanonicalAppHash(appHashText)
	if err != nil {
		return censusOptions{}, err
	}
	if len(sourceCommit) != 40 || strings.ToLower(sourceCommit) != sourceCommit {
		return censusOptions{}, errors.New("--source-commit must be exactly 40 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(sourceCommit); err != nil {
		return censusOptions{}, fmt.Errorf("--source-commit must be hexadecimal: %w", err)
	}
	return censusOptions{
		ChainID:      chainID,
		SourceCommit: sourceCommit,
		Height:       height,
		AppHash:      appHash,
	}, nil
}

func validChainID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, character := range []byte(value) {
		letter := character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
		digit := character >= '0' && character <= '9'
		if index == 0 && !letter && !digit {
			return false
		}
		if !letter && !digit && character != '.' && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func parseCanonicalPositiveHeight(raw string) (int64, error) {
	if raw == "" {
		return 0, errors.New("--expected-height is required")
	}
	height, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || height <= 0 || strconv.FormatInt(height, 10) != raw {
		return 0, errors.New("--expected-height must be a canonical positive int64 decimal string")
	}
	return height, nil
}

func parseCanonicalAppHash(raw string) ([]byte, error) {
	if len(raw) != 64 || strings.ToLower(raw) != raw {
		return nil, errors.New("--expected-app-hash must be exactly 64 lowercase hexadecimal characters")
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("--expected-app-hash must be hexadecimal: %w", err)
	}
	return decoded, nil
}
