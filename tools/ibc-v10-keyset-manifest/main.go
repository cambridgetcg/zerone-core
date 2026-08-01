package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type dbOpener func(home, backend string) (openedPhysicalDB, error)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, openCopiedApplicationDB); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ibc-v10-keyset-manifest: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, open dbOpener) error {
	if stdout == nil || stderr == nil {
		return errors.New("stdout and stderr are required")
	}
	if open == nil {
		return errors.New("database opener is nil")
	}

	flags := flag.NewFlagSet("ibc-v10-keyset-manifest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var (
		home             string
		backend          string
		heightText       string
		appHashText      string
		copiedDBAttested bool
	)
	flags.StringVar(&home, "home", "", "absolute home directory containing the copied data/application.db")
	flags.StringVar(&backend, "backend", "", "copied application DB backend: goleveldb or pebbledb")
	flags.StringVar(&heightText, "expected-height", "", "trusted positive root application height")
	flags.StringVar(
		&appHashText,
		"expected-app-hash",
		"",
		"trusted 32-byte post-commit root hash for H (ABCI last_block_app_hash), in hexadecimal",
	)
	flags.BoolVar(
		&copiedDBAttested,
		"copied-db",
		false,
		"attest that --home is a disposable copy made after the node was halted",
	)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if !copiedDBAttested {
		return errors.New(
			"--copied-db is required; never point this tool at a live or only application database",
		)
	}

	height, err := parseCanonicalPositiveHeight(heightText)
	if err != nil {
		return err
	}
	appHash, err := parseAppHash(appHashText)
	if err != nil {
		return err
	}

	db, err := open(home, backend)
	if err != nil {
		return err
	}
	manifest, auditErr := auditApplicationDB(db, expectedEvidence{
		Height:  height,
		AppHash: appHash,
	})
	closeErr := db.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("close copied application DB: %w", closeErr)
	}
	if err := errors.Join(auditErr, closeErr); err != nil {
		return err
	}

	if len(manifest) > maxPlanInfoBytes {
		return fmt.Errorf("refusing to emit plan.Info larger than %d bytes", maxPlanInfoBytes)
	}
	output := append(bytes.Clone(manifest), '\n')
	written, err := stdout.Write(output)
	if err != nil {
		return fmt.Errorf("write plan.Info: %w", err)
	}
	if written != len(output) {
		return fmt.Errorf("write plan.Info: %w", io.ErrShortWrite)
	}
	return nil
}

func parseCanonicalPositiveHeight(raw string) (int64, error) {
	if raw == "" {
		return 0, errors.New("--expected-height is required")
	}
	height, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || height <= 0 || strconv.FormatInt(height, 10) != raw {
		return 0, errors.New(
			"--expected-height must be a canonical positive int64 decimal string",
		)
	}
	return height, nil
}

func parseAppHash(raw string) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("--expected-app-hash is required")
	}
	if len(raw) != 64 {
		return nil, errors.New(
			"--expected-app-hash must be exactly 64 hexadecimal characters",
		)
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("--expected-app-hash must be hexadecimal: %w", err)
	}
	return decoded, nil
}
