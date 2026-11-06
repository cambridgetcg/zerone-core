package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Traces are compared to the unchanged consumers at 77521a0d using compiler
// overlays, then pinned below. They include exact response/error/events/gas and
// byte snapshots of ALL three real stores, not only the augmentation verdict.
func TestReformulationPreActivationCompatibility(t *testing.T) {
	// Captured by overlaying only training_economics.go, vindication.go and
	// route_b_heartbeat.go from 77521a0daf14420836483ae9d8ea3fbcc1ae0d40.
	// Do not regenerate these from current code: that would erase the oracle.
	predecessor := map[string]string{
		"vote":           "ea3fa5cf4d43c98d4a7aa9170dd80af9e15df17aa1a7f6bf78a866677213b3d1",
		"invalid":        "602cc09fa07396d532857f9cc237db551d0bfb333287fc531717978f71e6b223",
		"stored-invalid": "93e7cc79a44e3f7af3ee205f1a295b92bb04251f387d3d467f63b55e4d93dc86",
		"missing":        "ee12b59041e341d01a07aa0f6abdd21aab891c921c1be6207da582c963b4ce9e",
		"final":          "a13813de9cc84123f1080c5c8ad2d6d43d3fe879f7501e08cee9678db67118dd",
		"pending":        "953a966b14cb9f617d86c0733975f401d4e6fd06d9dc2bae60e79e22ea8af470",
		"duplicate":      "fa4ee58ed08a856471375a5bd651fc52f388c2346e8ad9a2baf5f96928aa74b7",
		"sponsor":        "ff82a896ed59a22c4f24b6e360537d80f342dafd357119227d7324c8392a49eb",
		"large":          "941bb98fb9f0b19003b30f45ee38fefa27c078635e055e5ad1081d2d013bcc0e",
		"vindication":    "b5ea12eeeef58259ae1788db0e1c0836d17b7d9f7d22d56ddfedbce86b9b5dd0",
		"expiry":         "7ef5e044297287355ccbb1709bd895edc3e429e0d7222c6c2195e66361750227",
		"forfeit":        "abf44d2ef9465bc4a29f385f9254967db4f3cd69acf31c20e658ca1c62e2f412",
		"release":        "3099f9b72f5702ce1fe4c72789573f3dc694127b51f2325fcd310187ef7836c0",
	}
	for _, name := range []string{"vote", "invalid", "stored-invalid", "missing", "final", "pending", "duplicate", "sponsor", "large", "vindication", "expiry", "forfeit", "release"} {
		t.Run(name, func(t *testing.T) {
			k, ctx, bank, staking := newKnowledgeOrderFixture(t)
			enabled, err := k.ReformulationVindicationOrderingEnabled(ctx)
			require.NoError(t, err)
			require.False(t, enabled)
			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			params.ReformulationMinPanelVotes = 1
			require.NoError(t, k.SetParams(ctx, params))
			sponsor := sdk.AccAddress([]byte("compat-sponsor-000001"))
			submitter := sdk.AccAddress([]byte("compat-author-0000010"))
			coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))
			require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
			require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
			require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor.String(), "bounty", sdkmath.NewInt(1500)))
			require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: "bounty", SponsorAddress: sponsor.String(), Active: true, RewardPerVariant: 1000, MaxVariants: 1, ExpiresAtBlock: 99}))
			aug := &types.Augmentation{Id: "variant", BountyId: "bounty", Submitter: submitter.String()}
			vote, verifier, id := types.AugmentationVerdict(1), "new-voter", "variant"
			stake := uint64(100_000)
			switch name {
			case "invalid":
				vote = 7
			case "stored-invalid", "expiry":
				aug.VerdictVoters, aug.VerdictVotes, aug.VerdictVoteStakes = []string{"old-voter"}, []types.AugmentationVerdict{7}, []uint64{300_000}
			case "missing":
				id = "missing"
			case "final":
				aug.Verdict = 7
			case "pending":
				vote = 0
			case "duplicate":
				aug.VerdictVoters = []string{verifier}
			case "sponsor":
				verifier = sponsor.String()
			case "large":
				stake = 200_000_000_000_000
			}
			staking.addValidator(verifier, stake, "bonded")
			require.NoError(t, k.SetAugmentation(ctx, aug))
			if name == "vindication" {
				require.NoError(t, bank.MintCoins(ctx, orderTestMint, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3))))
				require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, orderTestMint, types.VindicationEscrowModuleName, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3))))
				require.NoError(t, k.SetVindicationPending(ctx, "expired", []types.VindicationEntry{{FactId: "expired", Height: 1, SlashAmount: "3"}}))
			}
			snapshot := func() map[string]map[string][]byte {
				result := map[string]map[string][]byte{}
				ms := ctx.MultiStore().(*rootmulti.Store)
				for _, module := range []string{types.StoreKey, authtypes.StoreKey, banktypes.StoreKey} {
					store := ms.GetStoreByName(module).(storetypes.KVStore)
					iter := store.Iterator(nil, nil)
					rows := map[string][]byte{}
					for ; iter.Valid(); iter.Next() {
						rows[hex.EncodeToString(iter.Key())] = append([]byte(nil), iter.Value()...)
					}
					require.NoError(t, iter.Error())
					require.NoError(t, iter.Close())
					result[module] = rows
				}
				return result
			}
			before := snapshot()
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10_000_000)).WithEventManager(sdk.NewEventManager())
			var response *types.MsgVoteOnAugmentationResponse
			switch name {
			case "vindication":
				k.PruneExpiredVindications(ctx, 100, 10)
			case "expiry":
				k.ProcessRouteBLifecycle(ctx)
			case "forfeit":
				err = k.ForfeitAugmentationDeposit(ctx, "bounty", sdkmath.NewInt(100))
			case "release":
				err = k.ReleaseAugmentationPayout(ctx, "bounty", submitter.String(), sdkmath.NewInt(100))
			default:
				response, err = keeper.NewMsgServerImpl(k).VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: id, Verifier: verifier, Vote: vote})
			}
			gas := ctx.GasMeter().GasConsumed()
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			trace := struct {
				Before, After map[string]map[string][]byte
				Events        sdk.Events
				Response      *types.MsgVoteOnAugmentationResponse
				Error         string
				Gas           uint64
			}{before, snapshot(), ctx.EventManager().Events(), response, errText, gas}
			encoded, encodeErr := json.Marshal(trace)
			require.NoError(t, encodeErr)
			digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
			t.Logf("AW118_COMPAT %s gas=%d digest=%s", name, gas, digest)
			require.Equal(t, predecessor[name], digest, "off must preserve old state bytes, events, errors, response and gas")
		})
	}
}
