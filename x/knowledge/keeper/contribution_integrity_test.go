package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestContributionRawPolicyRefusesTruncationAndUnsupportedRecords(t *testing.T) {
	base, err := proto.Marshal(&types.ContributionRecord{ModelId: "model", FactIds: []string{"fact"}})
	require.NoError(t, err)
	policy := func(value uint64) []byte {
		return protowire.AppendVarint(protowire.AppendTag(nil, 9, protowire.VarintType), value)
	}
	for name, suffix := range map[string][]byte{
		"overflow-to-legacy":  policy(1 << 32),
		"overflow-to-current": policy((1 << 32) + 1),
		"unknown-policy":      policy(2),
		"duplicate-policy":    append(policy(0), policy(1)...),
		"wrong-wire":          protowire.AppendBytes(protowire.AppendTag(nil, 9, protowire.BytesType), []byte{1}),
		"overlong-policy":     append(protowire.AppendTag(nil, 9, protowire.VarintType), 0x81, 0x00),
		"unknown-field":       protowire.AppendVarint(protowire.AppendTag(nil, 100, protowire.VarintType), 1),
		"malformed-wire":      {0xff},
		"model-mismatch":      protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "other"),
		"declared-value":      append(policy(1), protowire.AppendVarint(protowire.AppendTag(nil, 6, protowire.VarintType), 1)...),
	} {
		t.Run(name, func(t *testing.T) {
			f := setupHandoff(t)
			require.NoError(t, f.k.SetContributionRecord(f.ctx, &types.ContributionRecord{ModelId: "aaa-valid"}))
			raw := append(append([]byte{}, base...), suffix...)
			if name == "overflow-to-legacy" {
				var unchecked types.ContributionRecord
				require.NoError(t, proto.Unmarshal(raw, &unchecked))
				require.Zero(t, unchecked.AttributionPolicyVersion, "this reproduces the decoder alias the raw guard must reject")
			}
			store := f.ctx.KVStore(f.keys[0])
			store.Set(types.ContributionByModelKey("model"), raw)
			before := f.snapshot(t)
			record, found, err := f.k.GetContributionRecordChecked(f.ctx, "model")
			require.Error(t, err)
			require.False(t, found)
			require.Nil(t, record)
			all, err := f.k.GetAllContributionRecordsChecked(f.ctx)
			require.Error(t, err)
			require.Nil(t, all, "a later malformed record must not produce a partial export")
			visited := 0
			require.Error(t, f.k.IterateContributionRecords(f.ctx, func(*types.ContributionRecord) bool { visited++; return false }))
			require.Zero(t, visited)
			_, err = NewQueryServerImpl(f.k).ModelContributions(f.ctx, &types.QueryModelContributionsRequest{ModelId: "model"})
			require.Equal(t, codes.Internal, status.Code(err))
			require.Error(t, f.k.SetContributionRecord(f.ctx, &types.ContributionRecord{ModelId: "model", AttributionPolicyVersion: 1}))
			_, err = NewMsgServerImpl(f.k).ChallengeContribution(f.ctx, &types.MsgChallengeContribution{Id: "challenge", ModelId: "model", DisputedFactId: "fact", Challenger: "challenger", DisputeType: "missing"})
			require.Error(t, err)
			func() {
				defer func() { require.Contains(t, fmt.Sprint(recover()), "export contribution records") }()
				f.k.ExportGenesis(f.ctx)
				t.Error("export must refuse the malformed declaration")
			}()
			require.Equal(t, before, f.snapshot(t))
		})
	}
}

func TestContributionCheckedReadsPreservePoliciesAndPropagateStorageFailure(t *testing.T) {
	f := setupHandoff(t)
	for _, record := range []*types.ContributionRecord{
		{ModelId: "legacy", AttributionPolicyVersion: 0, ComputedTvw: 321, PerFactCalibrationBps: []uint64{500_000}},
		{ModelId: "declaration", AttributionPolicyVersion: 1, FactIds: []string{"fact"}, TotalWeight: 7},
	} {
		require.NoError(t, f.k.SetContributionRecord(f.ctx, record))
		got, found, err := f.k.GetContributionRecordChecked(f.ctx, record.ModelId)
		require.NoError(t, err)
		require.True(t, found)
		require.True(t, proto.Equal(record, got))
	}
	before := f.snapshot(t)
	cache, _ := f.ctx.CacheContext()
	all, err := f.k.GetAllContributionRecordsChecked(cache)
	require.NoError(t, err, "normal nested cache iterator exhaustion remains accepted")
	require.Len(t, all, 2)
	_, found, err := f.k.GetContributionRecordChecked(f.ctx, "missing")
	require.NoError(t, err)
	require.False(t, found)
	*f.kFault = handoffFault{"get", types.ContributionByModelKey("legacy"), false}
	_, found, err = f.k.GetContributionRecordChecked(f.ctx, "legacy")
	require.Error(t, err)
	require.False(t, found)
	require.Error(t, f.k.SetContributionRecord(f.ctx, &types.ContributionRecord{ModelId: "legacy", AttributionPolicyVersion: 1}))
	require.Equal(t, before, f.snapshot(t))
}
