package keeper_test

import (
	"fmt"
	"testing"

	errorsmod "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/emergency/types"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

func TestHaltReasonEnabledRealEntryConsumers(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		start        uint64
		gate         uint64
		rollbackErr  error
	}{
		{"none", "", 0, 0, govtypes.ErrNoRollbackJustification},
		{"old_prose", "validator_keys", 100, 0, govtypes.ErrNoRollbackJustification},
		{"research_fund", types.HaltReasonResearchFund, 45_000_000, 1, nil},
		{"unknown_prose", "research_fund_misuse", 45_000_000, 1, types.ErrHaltEvidence},
		{"unrelated_unknown", "validator_keys", 45_000_000, 1, types.ErrHaltEvidence},
		{"empty_unknown", "", 45_000_000, 1, types.ErrHaltEvidence},
	} {
		for _, entry := range []string{"forward", "rollback", "query"} {
			t.Run(tc.name+"/"+entry, func(t *testing.T) {
				k, g, ctx, _ := setupHaltReasonKeepers(t)
				if tc.name != "none" {
					storeHaltReasonCeremony(t, k, ctx, "halt", tc.reason, "finalized", tc.start)
				}
				require.NoError(t, k.EnableHaltReasonScoping(ctx))
				response, err := haltEntry(g, ctx, entry)
				switch entry {
				case "forward":
					if tc.gate > 0 {
						require.ErrorIs(t, err, govtypes.ErrExitConditionsNotMet)
					} else {
						require.NoError(t, err)
					}
				case "rollback":
					if tc.rollbackErr != nil {
						require.ErrorIs(t, err, tc.rollbackErr)
					} else {
						require.NoError(t, err)
					}
				case "query":
					require.NoError(t, err)
					require.Equal(t, tc.gate, response.(*govtypes.QueryResearchFundGovernanceResponse).CurrentConditions.EmergencyHaltsFromMisuse)
				}
			})
		}
	}
}

func TestHaltReasonCurrentPhaseStartsAtCurrentHeight(t *testing.T) {
	k, g, ctx, _ := setupHaltReasonKeepers(t)
	state := g.GetResearchFundGovernanceState(ctx)
	state.PhaseStartedAtBlock = uint64(ctx.BlockHeight())
	g.SetResearchFundGovernanceState(ctx, state)
	storeHaltReasonCeremony(t, k, ctx, "same-height", types.HaltReasonResearchFund, "finalized", uint64(ctx.BlockHeight()))
	require.NoError(t, k.EnableHaltReasonScoping(ctx))
	selected := g.WithHaltReasonScoping(true)
	_, met := selected.CheckPhaseExitConditions(ctx)
	require.False(t, met)
	require.NoError(t, selected.ValidatePhaseRollback(ctx))
}

func TestHaltReasonUnknownErrorHasRegisteredABCIResult(t *testing.T) {
	k, g, ctx, _ := setupHaltReasonKeepers(t)
	storeHaltReasonCeremony(t, k, ctx, "prose", "validator double-sign incident", "finalized", 45_000_000)
	require.NoError(t, k.EnableHaltReasonScoping(ctx))
	_, err := haltEntry(g, ctx, "rollback")
	require.ErrorIs(t, err, types.ErrHaltEvidence)
	codespace, code, message := errorsmod.ABCIInfo(err, false)
	require.Equal(t, types.ModuleName, codespace)
	require.Equal(t, uint32(21), code)
	require.Equal(t, types.ErrHaltEvidence.Error(), message)
}

func TestHaltReasonBadCeremonyEnvelopeIsEvidenceError(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(fmt.Sprint(mismatch), func(t *testing.T) {
			k, g, ctx, key := setupHaltReasonKeepers(t)
			if mismatch {
				storeHaltReasonCeremony(t, k, ctx, "halt", types.HaltReasonResearchFund, "finalized", 45_000_000)
				ctx.KVStore(key).Set(types.CeremonyKey("wrong"), ctx.KVStore(key).Get(types.CeremonyKey("halt")))
				ctx.KVStore(key).Delete(types.CeremonyKey("halt"))
			} else {
				ctx.KVStore(key).Set(types.CeremonyKey("bad-wire"), []byte{0xff})
			}
			require.NoError(t, k.EnableHaltReasonScoping(ctx))
			selected := g.WithHaltReasonScoping(true)
			_, met := selected.CheckPhaseExitConditions(ctx)
			require.False(t, met)
			require.ErrorIs(t, selected.ValidatePhaseRollback(ctx), types.ErrHaltEvidence)
		})
	}
}
