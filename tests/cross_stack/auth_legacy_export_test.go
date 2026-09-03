package cross_stack_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
)

const (
	legacyAuthAddress   = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"
	legacyAuthPublicKey = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
	legacyAuthDID       = "did:zrn:d75a980182b10ab7d54bfed3c964073a"
)

func TestZerone1LegacyAuthExportIsEvidenceNotFreshGenesis(t *testing.T) {
	application := newTestApp(t, "zerone-1")
	initChainWithValSet(t, application, "zerone-1")

	ctx := application.NewContext(true)
	require.Empty(t, ctx.ChainID(), "SDK NewContext deliberately starts with an empty header")
	require.Equal(t, "zerone-1", application.ChainID(), "BaseApp must retain the trusted historical chain ID")
	committedHeight := uint64(application.LastBlockHeight())
	require.Positive(t, committedHeight)
	legacy := &zeroneauthtypes.Account{
		Address:               legacyAuthAddress,
		Did:                   legacyAuthDID,
		PublicKey:             legacyAuthPublicKey,
		AccountType:           "agent",
		OperationalKeyHash:    "",
		OperationalPublicKey:  legacyAuthPublicKey,
		OperationalKeyVersion: 1,
		ReputationScore:       500_000,
		CreatedAtBlock:        committedHeight,
		LastActiveBlock:       committedHeight,
		Flags: &zeroneauthtypes.AccountFlags{
			CanSubmitClaims: true,
			CanChallenge:    true,
		},
	}
	application.ZeroneAuthKeeper.SetAccount(ctx, legacy)
	application.ZeroneAuthKeeper.SetDIDMapping(ctx, &zeroneauthtypes.DIDMapping{
		Did:    legacy.Did,
		Bech32: legacy.Address,
		PubKey: legacy.PublicKey,
	})

	exported, err := application.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)

	var exportedState zeroneapp.GenesisState
	require.NoError(t, json.Unmarshal(exported.AppState, &exportedState))
	var exportedAuth zeroneauthtypes.GenesisState
	require.NoError(t, application.AppCodec().UnmarshalJSON(
		exportedState[zeroneauthtypes.ModuleName],
		&exportedAuth,
	))
	require.Len(t, exportedAuth.Accounts, 1)
	require.Empty(t, exportedAuth.Accounts[0].OperationalKeyHash,
		"export must preserve the legacy omission instead of inventing a hash")
	require.Empty(t, exportedAuth.LastKeyRotations,
		"export must not invent rotation history for a version-1 account")
	require.NoError(t, exportedAuth.ValidateForExport("zerone-1"))
	require.Error(t, exportedAuth.Validate(),
		"a historical evidence export must not become valid fresh-chain genesis")

	err = zeroneapp.ModuleBasics.ValidateGenesis(
		application.AppCodec(),
		application.TxConfig(),
		exportedState,
	)
	require.ErrorContains(t, err, "DID suffix must be 64 lowercase hex characters",
		"AppModuleBasic has no chain ID and must reject the historical short DID")
}

func TestExportRequiresTrustedBaseAppChainID(t *testing.T) {
	application := newTestApp(t, "")
	_, err := application.ExportAppStateAndValidators(false, nil, nil)
	require.ErrorContains(t, err, "requires a non-empty BaseApp chain ID")
}
