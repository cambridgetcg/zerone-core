package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

func TestRunEmitsOnlyCanonicalPlanInfo(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(
		t,
		db,
		42,
		[][]byte{[]byte("channelUpgrades/a")},
		nil,
		hashBytes(2),
	)
	var stdout, stderr bytes.Buffer
	openerCalled := false
	err := run(
		[]string{
			"--home", "/copied/zeroned",
			"--backend", "goleveldb",
			"--expected-height", "42",
			"--expected-app-hash", hex.EncodeToString(expected.AppHash),
			"--copied-db",
		},
		&stdout,
		&stderr,
		func(home, backend string) (openedPhysicalDB, error) {
			openerCalled = true
			require.Equal(t, "/copied/zeroned", home)
			require.Equal(t, "goleveldb", backend)
			return db, nil
		},
	)
	require.NoError(t, err)
	require.True(t, openerCalled)
	require.Empty(t, stderr.String())
	require.Equal(
		t,
		`{"schema":"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1","channel_upgrades":{"key_count":"1","keys_sha256":"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767"},"pruning_sequence_start":{"key_count":"0","keys_sha256":"`+
			emptySHA256+
			`"}}`+"\n",
		stdout.String(),
	)
}

func TestRunRefusesWithoutCopiedDBAttestation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	openerCalled := false
	err := run(
		[]string{"--home", "/potentially-live"},
		&stdout,
		&stderr,
		func(_, _ string) (openedPhysicalDB, error) {
			openerCalled = true
			return nil, errors.New("must not open")
		},
	)
	require.ErrorContains(t, err, "--copied-db is required")
	require.False(t, openerCalled)
	require.Empty(t, stdout.String())
}

func TestRunRejectsNoncanonicalEvidenceBeforeOpening(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "height",
			args: []string{
				"--copied-db",
				"--expected-height", "042",
				"--expected-app-hash", hex.EncodeToString(hashBytes(1)),
			},
		},
		{
			name: "app hash",
			args: []string{
				"--copied-db",
				"--expected-height", "42",
				"--expected-app-hash", "abc",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			openerCalled := false
			err := run(
				test.args,
				&stdout,
				&stderr,
				func(_, _ string) (openedPhysicalDB, error) {
					openerCalled = true
					return nil, errors.New("must not open")
				},
			)
			require.Error(t, err)
			require.False(t, openerCalled)
			require.Empty(t, stdout.String())
		})
	}
}

func TestRunRejectsShortWrite(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, db, 42, nil, nil, hashBytes(2))
	err := run(
		[]string{
			"--home", "/copied/zeroned",
			"--backend", "goleveldb",
			"--expected-height", "42",
			"--expected-app-hash", hex.EncodeToString(expected.AppHash),
			"--copied-db",
		},
		shortWriter{},
		io.Discard,
		func(_, _ string) (openedPhysicalDB, error) {
			return db, nil
		},
	)
	require.ErrorIs(t, err, io.ErrShortWrite)
}

func TestRunPropagatesWriteError(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, db, 42, nil, nil, hashBytes(2))
	writeFailure := errors.New("injected write failure")
	err := run(
		[]string{
			"--home", "/copied/zeroned",
			"--backend", "goleveldb",
			"--expected-height", "42",
			"--expected-app-hash", hex.EncodeToString(expected.AppHash),
			"--copied-db",
		},
		errorWriter{err: writeFailure},
		io.Discard,
		func(_, _ string) (openedPhysicalDB, error) {
			return db, nil
		},
	)
	require.ErrorIs(t, err, writeFailure)
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}
