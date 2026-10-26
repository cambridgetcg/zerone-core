package keeper

import (
	"fmt"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func TestRefundIdentityOffKeeperBodiesPreserveStateErrorsEventsGas(t *testing.T) {
	for _, operation := range []string{"set", "set_error", "delete", "delete_error", "recv", "recv_limit", "reset", "add", "add_duplicate", "add_unauthorized", "add_invalid", "remove", "remove_missing", "params"} {
		t.Run(operation, func(t *testing.T) {
			var state [2]map[string][]byte
			var gas [2]uint64
			var events [2]sdk.Events
			var results [2]string
			for i, legacy := range []bool{false, true} {
				k, ctx, key := chargeFixtureWithActivation(t, false)
				k.SetRateLimitLegacy(ctx, chargeRule())
				k.SetPacketFlow(ctx, &types.PacketFlow{ChannelId: "channel-0", Sequence: 1, Denom: "uzrn", Amount: "30"})
				if operation == "set_error" || operation == "delete_error" {
					fault := &chargeFaultService{KVStoreService: k.storeService, prefix: types.RateLimitKeyPrefix[0]}
					if operation == "delete_error" {
						fault.prefix, fault.delete = types.PacketFlowPrefix[0], true
					}
					k = NewKeeper(fault, k.cdc, k.authority)
				}
				ctx = ctx.WithBlockHeight(100).WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				ms := &msgServer{Keeper: k}
				var err error
				switch operation {
				case "set", "set_error":
					if legacy {
						k.SetRateLimitLegacy(ctx, chargeRule())
					} else {
						k.SetRateLimitForActivation(ctx, false, chargeRule())
					}
				case "delete", "delete_error":
					if legacy {
						k.DeletePacketFlowLegacy(ctx, "channel-0", 1)
					} else {
						k.DeletePacketFlowForActivation(ctx, false, "channel-0", 1)
					}
				case "recv", "recv_limit":
					amount := big.NewInt(20)
					if operation == "recv_limit" {
						amount.SetInt64(101)
					}
					if legacy {
						err = k.CheckAndUpdateRecvQuotaLegacy(ctx, "channel-0", "uzrn", amount)
					} else {
						err = k.CheckAndUpdateRecvQuotaForActivation(ctx, false, "channel-0", "uzrn", amount)
					}
				case "reset":
					if legacy {
						k.ResetExpiredWindowsLegacy(ctx)
					} else {
						k.ResetExpiredWindowsForActivation(ctx, false)
					}
				case "add", "add_duplicate", "add_unauthorized", "add_invalid":
					msg := &types.MsgAddRateLimit{Authority: "authority", ChannelId: "channel-1", Denom: "uzrn", MaxSend: "100", MaxRecv: "100", WindowBlocks: 10}
					if operation == "add_duplicate" {
						msg.ChannelId = "channel-0"
					}
					if operation == "add_unauthorized" {
						msg.Authority = "other"
					}
					if operation == "add_invalid" {
						msg.MaxSend = "0"
					}
					if legacy {
						_, err = ms.addRateLimitLegacy(ctx, msg)
					} else {
						_, err = ms.AddRateLimit(ctx, msg)
					}
				case "remove", "remove_missing":
					msg := &types.MsgRemoveRateLimit{Authority: "authority", ChannelId: "channel-0", Denom: "uzrn"}
					if operation == "remove_missing" {
						msg.ChannelId = "channel-1"
					}
					if legacy {
						_, err = ms.removeRateLimitLegacy(ctx, msg)
					} else {
						_, err = ms.RemoveRateLimit(ctx, msg)
					}
				case "params":
					msg := &types.MsgUpdateParams{Authority: "authority", Params: &types.Params{Enabled: false}}
					if legacy {
						_, err = ms.updateParamsLegacy(ctx, msg)
					} else {
						_, err = ms.UpdateParams(ctx, msg)
					}
				}
				gas[i], events[i], results[i] = ctx.GasMeter().GasConsumed(), ctx.EventManager().Events(), fmt.Sprint(err)
				state[i] = rawChargeState(ctx, key)
			}
			require.Equal(t, state[1], state[0])
			require.Equal(t, gas[1], gas[0])
			require.Equal(t, events[1], events[0])
			require.Equal(t, results[1], results[0])
		})
	}
}

func TestRefundIdentityNativeGenesisAndReimportGeneration(t *testing.T) {
	k, ctx, key := chargeFixtureWithActivation(t, false)
	gs := types.DefaultGenesis()
	gs.RateLimits = []*types.RateLimit{chargeRule()}
	k.InitGenesis(ctx, gs)
	require.Equal(t, uint64(1), k.ruleGeneration(ctx, "channel-0", "uzrn"))
	charge, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	require.NoError(t, err)
	k.RecordPacketCharge(ctx, 1, charge)
	k.DeleteRateLimit(ctx, "channel-0", "uzrn")
	k.SetRateLimitForActivation(ctx, true, chargeRule())
	require.Equal(t, uint64(2), k.ruleGeneration(ctx, "channel-0", "uzrn"))
	exported := k.ExportGenesis(ctx)
	require.True(t, exported.RefundIdentityEnabled)
	require.NotNil(t, ctx.KVStore(key).Get(packetFlowKey("channel-0", 1)))

	other, otherCtx, otherKey := chargeFixtureWithActivation(t, false)
	other.InitGenesis(otherCtx, exported)
	require.Equal(t, uint64(1), other.ruleGeneration(otherCtx, "channel-0", "uzrn"))
	require.Nil(t, otherCtx.KVStore(otherKey).Get(packetFlowKey("channel-0", 1)))
}

func TestRefundIdentityNumericSpellingAdvancesGeneration(t *testing.T) {
	k, ctx, _ := chargeFixture(t)
	rl := chargeRule()
	k.SetRateLimitForActivation(ctx, true, rl)
	rl.MaxSend = "0100"
	k.SetRateLimitForActivation(ctx, true, rl)
	require.Equal(t, uint64(2), k.ruleGeneration(ctx, "channel-0", "uzrn"))
}
