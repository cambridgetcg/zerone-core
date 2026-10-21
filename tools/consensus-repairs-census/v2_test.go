package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadOnlyCopiedDBV2Census(t *testing.T) {
	for _, fault := range []bool{false, true} {
		home, height, hash := sourceCopy(t, fault, false, true)
		before := fileDigests(t, home)
		var out, diagnostics bytes.Buffer
		code := run([]string{"--repairs", "v2", "--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics)
		want := 0
		if fault {
			want = 2
		}
		require.Equal(t, want, code, diagnostics.String()+out.String())
		var value struct {
			Schema string `json:"schema"`
			Passed bool   `json:"passed"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &value))
		require.Equal(t, "zerone.consensus-repairs-census/v2", value.Schema)
		require.Equal(t, !fault, value.Passed)
		require.Contains(t, out.String(), "fx7_accounting")
		require.Contains(t, out.String(), "fx8_retained")
		require.Contains(t, out.String(), "q5_calibration")
		require.Equal(t, before, fileDigests(t, home), "every copied source file remains byte-identical")
	}
}
