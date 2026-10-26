package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func initReformulationAdmissionGenesis(t *testing.T, k keeper.Keeper, ctx sdk.Context) {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.ReformulationVindicationOrderingEnabled = false
	gs.ReviewNeutralityEnabled = false // Admission remains available independently of the repair.
	gs.RecordIntegrityEnabled = false
	gs.ClaimRecordsEnabled = false
	gs.FundSettlementEnabled = false
	gs.ClaimIntegrityEnabled = false
	gs.BootstrapFundAllocation = "0"
	require.NoError(t, k.InitGenesis(ctx, gs))
}

func reformulationAdmissionSnapshot(t *testing.T, ctx sdk.Context) map[string]map[string][]byte {
	t.Helper()
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

// Captured from the real message handlers at 6a1e1ecf46bdb5ef1ad94e2b1a88e763132af8d9,
// before adding prospective delimiter validation. Do not regenerate from the repair.
// Traces include all knowledge/auth/bank bytes, events, responses, errors and gas.
func TestReformulationAdmissionPreActivationCompatibility(t *testing.T) {
	predecessor := map[string]string{
		"create":                "11077fd39cda6248a5c706a994502eb65aa167755227b2b8cefd238219fa4911",
		"create-nul":            "7db58d22e7ee7a0bcfa31d46211b87dddc1d7316387097cec947b54c751342be",
		"create-nil":            "247b7fef5a07d812b6ff6cea6e40cf23341367ba5612a282b95b3f7767f44474",
		"create-duplicate":      "d98e1b84a28690fd80b0cf343c3bd4b050576aeb642a455d0d1dc02b1040b784",
		"create-no-sponsor":     "b13b05f3c319a1f8b43c720e553a0f0263c238eeec9fecdc46215c951edb978a",
		"create-missing-fact":   "55365b8739d24f06378f02e7deaeba1153ea028261d8146d2795eebac01abc08",
		"create-underfunded":    "4b2224887ea8f8674cedcbd02d20bbb58a913055ece57600578d915c6971730e",
		"submit":                "3740ae9c0cd06df80ad035ff1b2f606e6857406bb3e9d6441ac78c9bffccc6b9",
		"submit-nul":            "d7a6887c281dce949b608b8bd55b2224cd6cb315ff8426eca3edfafdf6511ea1",
		"submit-nul-bounty":     "44873f117406a4d24fb171baf1135edd71078914180c9e971a966d4315e457e1",
		"submit-nil":            "0f8c1cccd2671121e3f5eb3df63171940ebe076c51bdc272d6f42658bcf2da0c",
		"submit-duplicate":      "f934794de0b145b0c48a71da674fa23587ab9e30b9961045c7b13202dedfc451",
		"submit-missing-bounty": "2cd6e837b5a242eef97731b84761f35b7f027f17832e62b4c35adc5f95757a3f",
		"submit-missing-fact":   "0c5e8ca4f56db5cd42af4e5628934bc2e92e14376b8775c8d6c8cda9c5192a16",
		"submit-no-content":     "bb494cf88efe81db9510a658a8483ef1a382f9441a35a35f4780e6fe4839e9a2",
	}
	for _, name := range []string{
		"create", "create-nul", "create-nil", "create-duplicate", "create-no-sponsor", "create-missing-fact", "create-underfunded",
		"submit", "submit-nul", "submit-nul-bounty", "submit-nil", "submit-duplicate", "submit-missing-bounty", "submit-missing-fact", "submit-no-content",
	} {
		t.Run(name, func(t *testing.T) {
			k, ctx, bank, _ := newKnowledgeOrderFixture(t)
			initReformulationAdmissionGenesis(t, k, ctx)
			enabled, err := k.ReformulationVindicationOrderingEnabled(ctx)
			require.NoError(t, err)
			require.False(t, enabled)
			sponsor := sdk.AccAddress([]byte("admit-sponsor-0000001"))
			author := sdk.AccAddress([]byte("admit-author-00000001"))
			coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3000))
			require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
			require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
			require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "original", Submitter: author.String()}))
			for _, id := range []string{"existing", "pool\x00legacy"} {
				require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: id, Active: true, TargetFactId: "original", MaxVariants: 1}))
			}
			require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "existing-variant"}))
			before := reformulationAdmissionSnapshot(t, ctx)
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10_000_000)).WithEventManager(sdk.NewEventManager())
			server := keeper.NewMsgServerImpl(k)
			var response any
			if strings.HasPrefix(name, "create") {
				msg := &types.MsgCreateAugmentationBounty{Id: "new", Sponsor: sponsor.String(), TargetFactId: "original", RewardPerVariant: 1000, MaxVariants: 1}
				switch name {
				case "create-nul":
					msg.Id = "new\x00bounty"
				case "create-nil":
					msg = nil
				case "create-duplicate":
					msg.Id = "existing"
				case "create-no-sponsor":
					msg.Sponsor = ""
				case "create-missing-fact":
					msg.TargetFactId = "missing"
				case "create-underfunded":
					msg.RewardPerVariant = 5000
				}
				response, err = server.CreateAugmentationBounty(ctx, msg)
			} else {
				msg := &types.MsgSubmitAugmentation{Id: "new-variant", BountyId: "existing", Submitter: author.String(), OriginalFactId: "original", VariantContent: "synthetic variant"}
				switch name {
				case "submit-nul":
					msg.Id = "new\x00variant"
				case "submit-nul-bounty":
					msg.BountyId = "pool\x00legacy"
				case "submit-nil":
					msg = nil
				case "submit-duplicate":
					msg.Id = "existing-variant"
				case "submit-missing-bounty":
					msg.BountyId = "missing"
				case "submit-missing-fact":
					msg.OriginalFactId = "missing"
				case "submit-no-content":
					msg.VariantContent = ""
				}
				response, err = server.SubmitAugmentation(ctx, msg)
			}
			gas := ctx.GasMeter().GasConsumed()
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			trace := struct {
				Before, After map[string]map[string][]byte
				Events        sdk.Events
				Response      any
				Error         string
				Gas           uint64
			}{before, reformulationAdmissionSnapshot(t, ctx), ctx.EventManager().Events(), response, errText, gas}
			encoded, encodeErr := json.Marshal(trace)
			require.NoError(t, encodeErr)
			digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
			t.Logf("AW118c_ADMISSION_COMPAT %s gas=%d digest=%s", name, gas, digest)
			require.Equal(t, predecessor[name], digest, "OFF must preserve admission state, events, errors, response and gas")
		})
	}
}
