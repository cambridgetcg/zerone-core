package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdkstakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	customstakingtypes "github.com/zerone-chain/zerone/x/staking/types"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestPreflightCommitInfoBoundsStoreCountBeforeTypedDecode(t *testing.T) {
	info := storetypes.CommitInfo{
		Version: 42,
		StoreInfos: []storetypes.StoreInfo{{
			Name: "bank",
			CommitId: storetypes.CommitID{
				Version: 42,
				Hash:    bytes.Repeat([]byte{0x42}, 32),
			},
		}},
	}
	valid, err := info.Marshal()
	require.NoError(t, err)
	require.NoError(t, preflightCommitInfoProto(valid))

	oversized := make([]byte, 0, 2*(maxCommitInfoStores+1))
	for range maxCommitInfoStores + 1 {
		oversized = protowire.AppendTag(oversized, 2, protowire.BytesType)
		oversized = protowire.AppendBytes(oversized, nil)
	}
	err = preflightCommitInfoProto(oversized)
	require.ErrorIs(t, err, errDecodeResourceLimit)
	require.ErrorContains(t, err, "mounted stores")
}

func TestPreflightJSONBoundsArraysAndStringsBeforeTypedDecode(t *testing.T) {
	t.Run("array", func(t *testing.T) {
		raw := jsonObjectArray("tier_configs", maxJSONArrayElements+1, `{}`)
		var destination struct {
			TierConfigs []map[string]any `json:"tier_configs"`
		}
		err := decodeStrictJSON(raw, &destination)
		require.ErrorIs(t, err, errDecodeResourceLimit)
		require.ErrorContains(t, err, "array")
		require.Nil(t, destination.TierConfigs, "typed decode must not run after preflight rejection")
	})

	t.Run("string", func(t *testing.T) {
		raw := []byte(`{"value":"` + strings.Repeat("x", maxJSONStringBytes+1) + `"}`)
		var destination struct {
			Value string `json:"value"`
		}
		err := decodeStrictJSON(raw, &destination)
		require.ErrorIs(t, err, errDecodeResourceLimit)
		require.ErrorContains(t, err, "string")
		require.Empty(t, destination.Value, "typed decode must not run after preflight rejection")
	})
}

func TestCensusPromotesDecodeLimitsToOperationalErrors(t *testing.T) {
	collector := newCensus()
	err := collector.ingest(
		customStakingStore,
		customstakingtypes.ParamsKey,
		jsonObjectArray("tier_configs", maxJSONArrayElements+1, `{}`),
	)
	require.ErrorIs(t, err, errDecodeResourceLimit)
	require.Empty(t, collector.findings)

	rawValidator := make([]byte, 0, 2*(maxSDKValidatorUnbondingIDs+1))
	for range maxSDKValidatorUnbondingIDs + 1 {
		rawValidator = protowire.AppendTag(rawValidator, 13, protowire.VarintType)
		rawValidator = protowire.AppendVarint(rawValidator, 1)
	}
	address := bytes.Repeat([]byte{0x23}, 20)
	key := append([]byte{sdkstakingtypes.ValidatorsKey[0], byte(len(address))}, address...)

	collector = newCensus()
	err = collector.ingest(sdkStakingStore, key, rawValidator)
	require.True(t, errors.Is(err, errDecodeResourceLimit))
	require.ErrorContains(t, err, "unbonding IDs")
	require.Empty(t, collector.findings)
}

func jsonObjectArray(name string, count int, element string) []byte {
	var builder strings.Builder
	builder.Grow(len(name) + count*(len(element)+1) + 8)
	builder.WriteString(`{"`)
	builder.WriteString(name)
	builder.WriteString(`":[`)
	for index := range count {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(element)
	}
	builder.WriteString(`]}`)
	return []byte(builder.String())
}
