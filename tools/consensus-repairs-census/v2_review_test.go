package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/app"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestV2SourceEligibilityAndExportRefusalAreSeparate(t *testing.T) {
	for _, scenario := range []string{"good", "target-map", "extra-version", "missing-v1-marker", "v1-receipt-without-done", "fx7-orphan", "fx7-counter", "fx7-bank-malformed", "fx7-bank-empty", "fx7-bank-negative", "fx8-counter", "fx8-proposal"} {
		t.Run(scenario, func(t *testing.T) {
			home, height, hash := sourceCopyMutated(t, false, []bool{false, true}, func(a *app.ZeroneApp, ctx sdk.Context) {
				switch scenario {
				case "target-map":
					require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, a.CurrentModuleVersionMap()))
				case "extra-version":
					vm, err := a.UpgradeKeeper.GetModuleVersionMap(ctx)
					require.NoError(t, err)
					vm["extra"] = 1
					require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
				case "missing-v1-marker":
					ctx.KVStore(a.GetStoreKeyForTests("knowledge")).Delete([]byte{0x90})
				case "v1-receipt-without-done":
					require.NoError(t, a.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_receipt_consensus-repairs-v1", "{}"))
				case "fx7-orphan":
					ctx.KVStore(a.GetStoreKeyForTests("knowledge")).Set([]byte("\x6borphan"), []byte("10"))
				case "fx7-counter":
					ctx.KVStore(a.GetStoreKeyForTests("knowledge")).Set([]byte("\x35bad"), []byte{1})
				case "fx7-bank-malformed", "fx7-bank-empty", "fx7-bank-negative":
					key, err := collections.EncodeKeyWithPrefix(banktypes.BalancesPrefix.Bytes(), collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey), collections.Join(authtypes.NewModuleAddress(knowledge.TrainingFundModuleName), "uzrn"))
					require.NoError(t, err)
					value := []byte{0xff}
					if scenario == "fx7-bank-empty" {
						value = []byte{}
					} else if scenario == "fx7-bank-negative" {
						value = []byte("-1")
					}
					ctx.KVStore(a.GetStoreKeyForTests("bank")).Set(key, value)
				case "fx8-counter":
					ctx.KVStore(a.GetStoreKeyForTests("zerone_gov")).Set([]byte{8}, binary.BigEndian.AppendUint64(nil, ^uint64(0)-1))
				case "fx8-proposal":
					ctx.KVStore(a.GetStoreKeyForTests("zerone_gov")).Set(binary.BigEndian.AppendUint64([]byte{6}, 7), []byte("null"))
				}
			})
			before := fileDigests(t, home)
			var out, diagnostics bytes.Buffer
			code := run([]string{"--repairs", "v2", "--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics)
			var value struct {
				Passed         bool   `json:"passed"`
				SourceEligible *bool  `json:"source_eligible"`
				SourceError    string `json:"source_error"`
				Error          string `json:"error"`
			}
			require.NoError(t, json.Unmarshal(out.Bytes(), &value), diagnostics.String())
			ineligible := scenario == "target-map" || scenario == "extra-version" || scenario == "missing-v1-marker" || scenario == "v1-receipt-without-done"
			exportFault := strings.HasPrefix(scenario, "fx")
			require.NotNil(t, value.SourceEligible, "explicit false must not disappear or become null")
			require.Equal(t, !ineligible, *value.SourceEligible)
			require.Equal(t, !ineligible, value.SourceError == "")
			require.Equal(t, !exportFault, value.Passed)
			if exportFault {
				require.Equal(t, 2, code)
				require.NotEmpty(t, value.Error)
			} else {
				require.Zero(t, code, "passed is inventory-only, not source-rung eligibility")
			}
			require.Equal(t, before, fileDigests(t, home))
		})
	}
}

func TestCensusUsageNamesRepairsSelection(t *testing.T) {
	var out, diagnostics bytes.Buffer
	require.Equal(t, 1, run(nil, &out, &diagnostics))
	require.Contains(t, diagnostics.String(), "--repairs v1|v2")
	for _, selection := range []string{"", "V2", "v3"} {
		diagnostics.Reset()
		require.Equal(t, 1, run([]string{"--repairs", selection}, &out, &diagnostics))
		require.Contains(t, diagnostics.String(), "--repairs v1|v2")
	}
}
