package keeper_test

import (
	"encoding/hex"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/home/keeper"
	"github.com/zerone-chain/zerone/x/home/types"
)

// TestHomeCompositionBoundaries follows actual message, query and BeginBlock
// consumers over the real keeper fixture. The bank dependency is synthetic;
// this is not a transaction, general authorization or node-deployment test.
func TestHomeCompositionBoundaries(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msg := keeper.NewMsgServerImpl(k)
	query := keeper.NewQueryServerImpl(k)
	owners := map[string]string{"home-1": testAddr("composition-a"), "home-10": testAddr("composition-b")}
	for _, id := range []string{"home-1", "home-10"} {
		k.SetHome(ctx, &types.AgentHome{HomeId: id, OwnerAddress: owners[id], Status: "active", LastActiveBlock: 100})
		k.AddHomeToOwnerIndex(ctx, owners[id], id)
		_, err := msg.RegisterKey(ctx, &types.MsgRegisterKey{Owner: owners[id], HomeId: id, KeyHash: "shared-key", KeyType: "session", Permissions: []string{"read"}})
		require.NoError(t, err)
		_, err = msg.StartSession(ctx, &types.MsgStartSession{HomeId: id, KeyHash: "shared-key", RequestedPermissions: []string{"read", "write"}})
		require.NoError(t, err)
		_, err = msg.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{Owner: owners[id], HomeId: id, KeyType: "session", MaxAmount: "7", PeriodBlocks: 50})
		require.NoError(t, err)
	}
	_, err := msg.RegisterKey(ctx, &types.MsgRegisterKey{Owner: owners["home-1"], HomeId: "home-1", KeyHash: "spare-key", KeyType: "session", Permissions: []string{"read"}})
	require.NoError(t, err)
	spare, err := msg.StartSession(ctx, &types.MsgStartSession{HomeId: "home-1", KeyHash: "spare-key"})
	require.NoError(t, err)
	k.SetAlert(ctx, &types.Alert{HomeId: "home-1", AlertId: "a", Acknowledged: false})
	k.SetAlert(ctx, &types.Alert{HomeId: "home-1", AlertId: "z", Acknowledged: true})
	k.SetAlert(ctx, &types.Alert{HomeId: "home-10", AlertId: "neighbor", Acknowledged: false})

	// Captured from fdec11df before the prefix consolidation. These fixtures
	// bind real query/export bytes and read gas, not raw-store or AppHash identity.
	expected := map[string]struct {
		wire string
		gas  uint64
	}{
		"sessions":          {"0a310a107365732d7368617265642d6b2d3130301206686f6d652d311a0a7368617265642d6b6579220472656164286430cc080a300a107365732d73706172652d6b652d3130301206686f6d652d311a0973706172652d6b6579220472656164286430cc08", 744},
		"keys":              {"0a1f0a0a7368617265642d6b6579120773657373696f6e220472656164286430640a1e0a0973706172652d6b6579120773657373696f6e22047265616428643064", 525},
		"alerts":            {"0a0b0a01611206686f6d652d31", 276},
		"spending":          {"0a130a0773657373696f6e12013718322201302864", 264},
		"supported-genesis": {"0a1b0814100518e807206428a08d0630643a083130303030303030400512470a06686f6d652d31122a7a726e31333467686d7a35716e3568686a676376773036643363733237756d6d777a77676a6c6c6b6a7022076775617264656442061a04080110645064123f0a07686f6d652d3130122a7a726e3172617a76736e67796a6e6d6e6571307a39356e61657a337161333863326e32736a3336687a6a220661637469766550641a4d0a06686f6d652d3112230a0a7368617265642d6b6579120773657373696f6e2204726561642864306440014864121e0a0973706172652d6b6579120773657373696f6e220472656164286430641a2a0a07686f6d652d3130121f0a0a7368617265642d6b6579120773657373696f6e22047265616428643064", 2764},
	}
	snapshot := func(name string, call func() (proto.Message, error)) {
		t.Helper()
		before := ctx.GasMeter().GasConsumed()
		value, err := call()
		require.NoError(t, err)
		gas := ctx.GasMeter().GasConsumed() - before
		body, err := (proto.MarshalOptions{Deterministic: true}).Marshal(value)
		require.NoError(t, err)
		want, found := expected[name]
		require.True(t, found, "missing compatibility fixture for %s", name)
		require.Equal(t, want.wire, hex.EncodeToString(body), "%s wire bytes", name)
		require.Equal(t, want.gas, gas, "%s read gas", name)
		t.Logf("HOME_COMPAT %s bytes=%s gas=%d", name, hex.EncodeToString(body), gas)
	}

	snapshot("sessions", func() (proto.Message, error) {
		result, err := query.Sessions(ctx, &types.QuerySessionsRequest{HomeId: "home-1"})
		if err == nil {
			require.Len(t, result.Sessions, 2)
			for _, session := range result.Sessions {
				require.Equal(t, "home-1", session.HomeId)
				require.Equal(t, []string{"read"}, session.Permissions)
			}
		}
		return result, err
	})
	snapshot("keys", func() (proto.Message, error) {
		result, err := query.Keys(ctx, &types.QueryKeysRequest{HomeId: "home-1"})
		if err == nil {
			require.Len(t, result.Keys, 2)
			require.Equal(t, "shared-key", result.Keys[0].KeyHash)
			require.Equal(t, "spare-key", result.Keys[1].KeyHash)
		}
		return result, err
	})
	snapshot("alerts", func() (proto.Message, error) {
		result, err := query.Alerts(ctx, &types.QueryAlertsRequest{HomeId: "home-1", UnacknowledgedOnly: true})
		if err == nil {
			require.Len(t, result.Alerts, 1)
			require.Equal(t, "a", result.Alerts[0].AlertId)
		}
		return result, err
	})
	snapshot("spending", func() (proto.Message, error) {
		result, err := query.SpendingLimits(ctx, &types.QuerySpendingLimitsRequest{HomeId: "home-1"})
		if err == nil {
			require.Len(t, result.Limits, 1)
			require.Equal(t, "session", result.Limits[0].KeyType)
			require.Equal(t, "7", result.Limits[0].MaxAmount)
		}
		return result, err
	})
	require.EqualValues(t, 2, k.CountSessions(ctx, "home-1"))
	require.EqualValues(t, 1, k.CountSessions(ctx, "home-10"))
	require.EqualValues(t, 2, k.CountActiveKeys(ctx, "home-1"))
	require.EqualValues(t, 1, k.CountPendingAlerts(ctx, "home-1"))

	// The key name is deliberately shared: revoking in one home must not
	// remove the neighboring home's key or session.
	_, err = msg.RevokeKey(ctx, &types.MsgRevokeKey{Owner: owners["home-1"], HomeId: "home-1", KeyHash: "shared-key"})
	require.NoError(t, err)
	require.EqualValues(t, 1, k.CountSessions(ctx, "home-1"))
	require.EqualValues(t, 1, k.CountSessions(ctx, "home-10"))
	require.EqualValues(t, 1, k.CountActiveKeys(ctx, "home-1"))
	otherKey, found := k.GetKeyRegistration(ctx, "home-10", "shared-key")
	require.True(t, found)
	require.False(t, otherKey.Revoked)
	_, err = msg.EndSession(ctx, &types.MsgEndSession{HomeId: "home-1", SessionId: spare.SessionId})
	require.NoError(t, err)
	require.Zero(t, k.CountSessions(ctx, "home-1"))
	require.EqualValues(t, 1, k.CountSessions(ctx, "home-10"))

	// Explicit lifecycle boundaries: equality is not expiry or a deadman
	// trigger. Only the configured home becomes guarded; the neighbor's
	// session survives until the first block strictly after ExpiresAt.
	_, err = msg.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner: owners["home-1"], HomeId: "home-1",
		Deadman: &types.DeadmanConfig{Enabled: true, InactivityThreshold: 100},
	})
	require.NoError(t, err)
	atHeight := func(height int64) sdk.Context { return ctx.WithBlockHeight(height) }
	require.NoError(t, k.BeginBlocker(atHeight(200)))
	home, found := k.GetHome(ctx, "home-1")
	require.True(t, found)
	require.Equal(t, "active", home.Status)
	require.NoError(t, k.BeginBlocker(atHeight(201)))
	home, _ = k.GetHome(ctx, "home-1")
	require.Equal(t, "guarded", home.Status)
	neighbor, found := k.GetHome(ctx, "home-10")
	require.True(t, found)
	require.Equal(t, "active", neighbor.Status)
	require.EqualValues(t, 1, k.CountSessions(ctx, "home-10"))
	k.CleanupExpiredSessions(atHeight(1100))
	require.EqualValues(t, 1, k.CountSessions(ctx, "home-10"))
	k.CleanupExpiredSessions(atHeight(1101))
	require.Zero(t, k.CountSessions(ctx, "home-10"))
	alerts := k.GetAlertsByHome(ctx, "home-10")
	require.Len(t, alerts, 2)
	for _, alert := range alerts {
		require.Equal(t, "home-10", alert.HomeId)
	}

	snapshot("supported-genesis", func() (proto.Message, error) {
		genesis := k.ExportGenesis(ctx)
		require.Len(t, genesis.Homes, 2)
		require.Equal(t, "home-1", genesis.Homes[0].HomeId)
		require.Equal(t, "home-10", genesis.Homes[1].HomeId)
		require.Len(t, genesis.KeySets, 2)
		require.Len(t, genesis.KeySets[0].Keys, 2)
		require.Len(t, genesis.KeySets[1].Keys, 1)
		require.True(t, genesis.KeySets[0].Keys[0].Revoked)
		require.False(t, genesis.KeySets[1].Keys[0].Revoked)
		return genesis, nil
	})
}
