package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestReformulationAdmissionDelimiterGate(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, operation := range []string{"create-bounty", "submit", "submit-volunteer", "submit-bounty-reference"} {
			for _, id := range []string{"plain", "\x00leading", "mid\x00dle", "trailing\x00"} {
				t.Run(fmt.Sprintf("enabled=%t/%s/%q", enabled, operation, id), func(t *testing.T) {
					probe := &reformulationReadProbe{}
					k, ctx, bank, _ := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService {
						return reformulationProbeService{s, probe}
					})
					initReformulationAdmissionGenesis(t, k, ctx)
					sponsor := sdk.AccAddress([]byte("admit-sponsor-0000001"))
					author := sdk.AccAddress([]byte("admit-author-00000001"))
					coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))
					require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
					require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
					require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "original", Submitter: author.String()}))
					bountyID := "existing"
					if operation == "submit-bounty-reference" {
						bountyID = id
					}
					// Retained state is not rewritten or rejected by the setters.
					require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: bountyID, Active: true, TargetFactId: "original", MaxVariants: 1}))
					if enabled {
						require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
					}
					before := reformulationAdmissionSnapshot(t, ctx)
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					*probe = reformulationReadProbe{}
					server := keeper.NewMsgServerImpl(k)
					var err error
					if operation == "create-bounty" {
						_, err = server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{Id: id, Sponsor: sponsor.String(), TargetFactId: "original", RewardPerVariant: 1000, MaxVariants: 1})
					} else {
						augmentationID := id
						if operation == "submit-bounty-reference" {
							augmentationID = "new-variant"
						}
						if operation == "submit-volunteer" {
							bountyID = ""
						}
						_, err = server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{Id: augmentationID, BountyId: bountyID, OriginalFactId: "original", Submitter: author.String(), VariantContent: "synthetic variant"})
					}
					require.Equal(t, 1, probe.markerReads, "entry point snapshots once, even OFF")
					require.False(t, probe.writeBeforeRead)
					if enabled && strings.ContainsRune(id, '\x00') {
						require.ErrorContains(t, err, "id must not contain NUL")
						require.Zero(t, probe.writes)
						require.Equal(t, before, reformulationAdmissionSnapshot(t, ctx), "no record, index, escrow or bank write on refusal")
						require.Empty(t, ctx.EventManager().Events())
					} else {
						require.NoError(t, err)
						require.Positive(t, probe.writes)
						require.NotEmpty(t, ctx.EventManager().Events())
					}
				})
			}
		}
	}
}

func TestReformulationCarrierCensusRejectsNULIdentifiers(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, source := range []string{"bounty", "inactive-bounty", "augmentation", "final-augmentation", "volunteer", "bounty-reference"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, source), func(t *testing.T) {
				probe := &reformulationReadProbe{}
				k, ctx, _, _ := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService {
					return reformulationProbeService{s, probe}
				})
				initReformulationAdmissionGenesis(t, k, ctx)
				const badID = "legacy\x00id"
				switch source {
				case "bounty", "inactive-bounty":
					require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: badID, Active: source == "bounty"}))
				default:
					aug := &types.Augmentation{Id: badID, BountyId: "pool"}
					if source == "volunteer" {
						aug.BountyId = ""
					}
					if source == "final-augmentation" {
						aug.Verdict = types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT
					}
					if source == "bounty-reference" {
						aug.Id, aug.BountyId = "plain", badID
					}
					require.NoError(t, k.SetAugmentation(ctx, aug))
				}
				if enabled {
					require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
				}
				before := reformulationAdmissionSnapshot(t, ctx)
				ctx = ctx.WithEventManager(sdk.NewEventManager())
				*probe = reformulationReadProbe{}
				holds, err := k.CensusReformulationHolds(ctx)
				require.ErrorContains(t, err, "NUL")
				require.ErrorContains(t, err, fmt.Sprintf("%q", badID), "surface the exact offending ID with escaped NUL")
				require.ErrorContains(t, err, "activation must remain on hold")
				require.Nil(t, holds, "never return a successful partial census")
				require.Zero(t, probe.markerReads, "the census is prospective, not marker-selected")
				require.Zero(t, probe.writes, "no implicit migration or rename")
				require.Equal(t, before, reformulationAdmissionSnapshot(t, ctx))
				require.Empty(t, ctx.EventManager().Events())
			})
		}
	}
}

func TestReformulationCarrierCensusChecksBountyRecords(t *testing.T) {
	for _, corruption := range []string{"malformed", "empty-id", "wrong-id"} {
		t.Run(corruption, func(t *testing.T) {
			f := newKnowledgeStoreFixture(t, "bounty-census")
			require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{Id: "pool"}))
			store := f.ctx.KVStore(f.key)
			switch corruption {
			case "malformed":
				store.Set(types.AugmentationBountyKey("pool"), []byte{0xff})
			case "empty-id":
				store.Set(types.AugmentationBountyKey("pool"), []byte{})
			case "wrong-id":
				require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{Id: "other"}))
				store.Set(types.AugmentationBountyKey("pool"), store.Get(types.AugmentationBountyKey("other")))
			}
			before := relationContractSnapshot(t, f.ctx, f.key)
			holds, err := f.k.CensusReformulationHolds(f.ctx)
			require.ErrorContains(t, err, "reformulation census:")
			require.Nil(t, holds)
			require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key))
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}
