package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/sponsorship/keeper"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func setRawLegacyFulfillment(
	t *testing.T,
	kv interface {
		Set(key, value []byte) error
	},
	fulfillment *types.BountyFulfillment,
) []byte {
	t.Helper()
	bz, err := proto.Marshal(fulfillment)
	require.NoError(t, err)
	require.NoError(
		t,
		kv.Set(
			types.FulfillmentKey(fulfillment.BountyId, fulfillment.FactId),
			bz,
		),
	)
	return bz
}

func TestMigration1To2_MalformedRawOrderFailsBeforeV2Writes(t *testing.T) {
	k, ctx, _, _, storeService := setupWithStoreService(t)
	k.SetParams(ctx, types.DefaultParams())
	kv := storeService.OpenKVStore(ctx)
	require.NoError(t, kv.Delete(types.EscrowLiabilityKey))

	validFulfillment := &types.BountyFulfillment{
		BountyId:         "a-valid-bounty",
		FactId:           "a-valid-fact",
		Worker:           mkAddr("raw-order-worker").String(),
		AmountPaid:       "+1",
		FulfilledAtBlock: 1,
	}
	validFulfillmentBytes := setRawLegacyFulfillment(t, kv, validFulfillment)
	malformedOrderKey := types.BountyOrderKey("z-malformed-order")
	malformedOrderBytes := []byte{0xff}
	require.NoError(t, kv.Set(malformedOrderKey, malformedOrderBytes))

	paramsBefore, err := kv.Get(types.ParamsKey)
	require.NoError(t, err)
	paramsBefore = append([]byte(nil), paramsBefore...)

	err = keeper.NewMigrator(k).Migrate1to2(ctx)
	require.ErrorContains(t, err, "census legacy bounty orders")
	require.ErrorContains(t, err, "decode bounty-order row")

	paramsAfter, readErr := kv.Get(types.ParamsKey)
	require.NoError(t, readErr)
	require.Equal(t, paramsBefore, paramsAfter)
	require.Equal(t, malformedOrderBytes, mustGetRaw(t, kv, malformedOrderKey))
	require.Equal(
		t,
		validFulfillmentBytes,
		mustGetRaw(
			t,
			kv,
			types.FulfillmentKey(validFulfillment.BountyId, validFulfillment.FactId),
		),
	)
	require.Nil(t, mustGetRaw(t, kv, types.FactConsumptionKey(validFulfillment.FactId)))
	require.Nil(t, mustGetRaw(t, kv, types.EscrowLiabilityKey))
}

func TestMigration1To2_MalformedRawFulfillmentFailsBeforeV2Writes(t *testing.T) {
	k, ctx, _, _, storeService := setupWithStoreService(t)
	k.SetParams(ctx, types.DefaultParams())
	kv := storeService.OpenKVStore(ctx)
	require.NoError(t, kv.Delete(types.EscrowLiabilityKey))

	validFulfillment := &types.BountyFulfillment{
		BountyId:         "a-valid-bounty",
		FactId:           "a-valid-fact",
		Worker:           mkAddr("raw-fulfillment-work").String(),
		AmountPaid:       "+1",
		FulfilledAtBlock: 1,
	}
	validFulfillmentBytes := setRawLegacyFulfillment(t, kv, validFulfillment)
	malformedFulfillmentKey := types.FulfillmentKey(
		"z-malformed-bounty",
		"z-malformed-fact",
	)
	malformedFulfillmentBytes := []byte{0xff}
	require.NoError(
		t,
		kv.Set(malformedFulfillmentKey, malformedFulfillmentBytes),
	)

	paramsBefore, err := kv.Get(types.ParamsKey)
	require.NoError(t, err)
	paramsBefore = append([]byte(nil), paramsBefore...)

	err = keeper.NewMigrator(k).Migrate1to2(ctx)
	require.ErrorContains(t, err, "census legacy fulfillments")
	require.ErrorContains(t, err, "decode fulfillment row")

	paramsAfter, readErr := kv.Get(types.ParamsKey)
	require.NoError(t, readErr)
	require.Equal(t, paramsBefore, paramsAfter)
	require.Equal(
		t,
		validFulfillmentBytes,
		mustGetRaw(
			t,
			kv,
			types.FulfillmentKey(validFulfillment.BountyId, validFulfillment.FactId),
		),
	)
	require.Equal(
		t,
		malformedFulfillmentBytes,
		mustGetRaw(t, kv, malformedFulfillmentKey),
	)
	require.Nil(t, mustGetRaw(t, kv, types.FactConsumptionKey(validFulfillment.FactId)))
	require.Nil(t, mustGetRaw(t, kv, types.EscrowLiabilityKey))
}

func mustGetRaw(
	t *testing.T,
	kv interface {
		Get(key []byte) ([]byte, error)
	},
	key []byte,
) []byte {
	t.Helper()
	bz, err := kv.Get(key)
	require.NoError(t, err)
	return bz
}
