package keeper_test

import (
	"encoding/json"
	"flag"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The augmentation inventory charge, shared with the offline audit
// (scripts/audit-augmentation-edge-overwrites.py). RestoreAugmentationReplacedEdges
// refuses an inventory above 100,000 records or 64 MiB, charging each record
// its key (0x63 and the ID) plus its stored value; above either bound the
// knowledge-claim-integrity-v1 upgrade fails at its height, and an import with
// claim integrity refuses the genesis. The audit computes the same charge from
// an export, so an operator can check the bound before the upgrade height.
// This vector ties the two together: the keeper admits exactly the charge
// recorded here and refuses one byte or one record less, and the audit's
// tests require that it computes the same charge from the same records, as
// the export lists them.

var updateAugmentationChargeVector = flag.Bool("update-augmentation-charge-vector", false,
	"rewrite scripts/testdata/augmentation-inventory-charge.json from the keeper")

const augmentationChargeVectorPath = "../../../scripts/testdata/augmentation-inventory-charge.json"

// augmentationChargeVectorRecords sets every Augmentation field, at the
// encoding boundaries that change a record's size: one- and two-byte field
// tags (fields 16-18), string lengths that need one, two and three length
// bytes, multi-byte UTF-8, the largest uint64, zero inside packed repeated
// fields, an empty string inside a repeated string, and a record with an ID
// alone. All values are synthetic.
func augmentationChargeVectorRecords() []*types.Augmentation {
	long := strings.Repeat("Synthetic variant text for the inventory charge vector. ", 300)[:1<<14]
	return []*types.Augmentation{
		{Id: "synthetic-charge-id-only"},
		{
			Id:                        "synthetic-charge-every-field",
			BountyId:                  "synthetic-charge-bounty",
			OriginalFactId:            "synthetic-charge-original",
			VariantContent:            "A synthetic variant whose content is short.",
			VariantReasoningTrace:     strings.Repeat("r", 200),
			Submitter:                 "zrn1syntheticchargesubmitter",
			CreatedAtBlock:            1,
			Accepted:                  true,
			AcceptedAtBlock:           math.MaxUint64,
			AcceptanceNote:            "quote \" backslash \\ newline \n tab \t <html> & ünïcødé 文字",
			Verdict:                   types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR,
			VerdictBlock:              127,
			VerdictVoters:             []string{"zrn1syntheticvoter1", "", "zrn1syntheticvoter3"},
			VerdictVotes:              []types.AugmentationVerdict{types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING, types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT, types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR},
			SponsorVetoed:             true,
			PayoutAmount:              "18446744073709551616",
			VerdictVoteStakes:         []uint64{0, 128, math.MaxUint64},
			VerdictVoteCalibrationBps: []uint64{0, 16383, 16384},
		},
		{
			Id:             "synthetic-charge-ünïcødé-文字/x",
			OriginalFactId: "synthetic-charge-original",
			VariantContent: long,
			Verdict:        types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
			VerdictBlock:   128,
		},
		{
			Id:            "synthetic-charge-zero-lists",
			Verdict:       types.AugmentationVerdict_AUGMENTATION_VERDICT_INFERIOR,
			VerdictVotes:  []types.AugmentationVerdict{types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING},
			VerdictVoters: []string{""},
		},
	}
}

type augmentationChargeVector struct {
	Schema        string          `json:"schema"`
	Note          string          `json:"note"`
	Augmentations json.RawMessage `json:"augmentations"`
	KeeperCharge  struct {
		Records int `json:"records"`
		Bytes   int `json:"bytes"`
	} `json:"keeper_charge"`
}

func TestAugmentationInventoryChargeVector(t *testing.T) {
	f := setupV11SemanticsTest(t)
	for _, aug := range augmentationChargeVectorRecords() {
		require.NoError(t, f.k.SetAugmentation(f.ctx, aug))
	}
	records, size := 0, 0
	it := f.ctx.KVStore(f.key).Iterator(types.AugmentationKeyPrefix, []byte{types.AugmentationKeyPrefix[0] + 1})
	for ; it.Valid(); it.Next() {
		records++
		size += len(it.Key()) + len(it.Value())
	}
	require.NoError(t, it.Close())
	require.Equal(t, len(augmentationChargeVectorRecords()), records)

	// The keeper charges exactly this: its bounds admit it and refuse one less.
	restore := func(maxRecords, maxBytes int) error {
		keeper.SetAugmentationRestoreMaxRecordsForTest(t, maxRecords)
		keeper.SetAugmentationRestoreMaxBytesForTest(t, maxBytes)
		var err error
		f.inCache(func() { _, err = f.k.RestoreAugmentationReplacedEdges(f.ctx) })
		return err
	}
	require.NoError(t, restore(records, size))
	require.ErrorContains(t, restore(records, size-1), "augmentation inventory exceeds")
	require.ErrorContains(t, restore(records-1, size), "augmentation inventory exceeds")

	// The same records as an export lists them (the module's ExportGenesis
	// codec: gogoproto jsonpb, original field names, defaults emitted).
	exported, err := codec.NewProtoCodec(codectypes.NewInterfaceRegistry()).MarshalJSON(f.k.ExportGenesis(f.ctx))
	require.NoError(t, err)
	var module map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported, &module))
	vector := augmentationChargeVector{
		Schema: "zerone.knowledge.augmentation-inventory-charge-vector/v1",
		Note: "Synthetic. Written by TestAugmentationInventoryChargeVector (x/knowledge/keeper) with " +
			"-update-augmentation-charge-vector; read by scripts/test_audit_augmentation_edge_overwrites.py. " +
			"keeper_charge is what RestoreAugmentationReplacedEdges charges these records: key plus stored value, summed.",
		Augmentations: module["augmentations"],
	}
	vector.KeeperCharge.Records, vector.KeeperCharge.Bytes = records, size
	got, err := json.MarshalIndent(vector, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')
	if *updateAugmentationChargeVector {
		require.NoError(t, os.WriteFile(augmentationChargeVectorPath, got, 0o644))
	}
	want, err := os.ReadFile(augmentationChargeVectorPath)
	require.NoError(t, err, "run: go test ./x/knowledge/keeper -run TestAugmentationInventoryChargeVector -update-augmentation-charge-vector")
	require.Equal(t, string(want), string(got), "the vector is stale; regenerate it with -update-augmentation-charge-vector")
}
