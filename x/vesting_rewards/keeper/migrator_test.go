package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

func TestMigrate1to2RetiresAutomaticClaimsWithoutMovingState(t *testing.T) {
	founderAddress := sdk.AccAddress("legacy_founder______").String()
	for _, fixture := range []struct {
		name    string
		share   uint64
		address string
	}{
		{name: "share_only", share: 70_000},
		{name: "address_only", address: founderAddress},
		{name: "share_and_address", share: 70_000, address: founderAddress},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			bank := newMockBankKeeper()

			legacy := types.DefaultParams()
			legacy.FounderShareBps = fixture.share
			legacy.FounderAddress = fixture.address
			legacy.BlockReward = "10000000"
			legacy.FloorReward = "100000"
			legacy.EmptyBlockRewardRate = 500
			genesis := types.DefaultGenesis()
			genesis.Params = legacy
			k, ctx := setupKeeperWithBankAndGenesis(t, bank, &mockStakingKeeper{activeCount: 1}, genesis)

			history := &types.BlockRewardDistribution{
				BlockHeight:    9,
				ProducerReward: "123",
				FounderShare:   "7",
				TotalMinted:    "456",
			}
			k.SetBlockRewardDistribution(ctx, history)

			expected := proto.Clone(legacy).(*types.Params)
			expected.FounderShareBps = 0
			expected.FounderAddress = ""
			expected.BlockReward = "0"
			expected.FloorReward = "0"
			expected.EmptyBlockRewardRate = 0

			require.NoError(t, keeper.NewMigrator(k).Migrate1to2(ctx))
			require.True(t, proto.Equal(expected, k.GetParams(ctx)),
				"migration may change only retired compatibility parameters")
			require.NoError(t, types.ValidateParams(k.GetParams(ctx)))

			gotHistory, found := k.GetBlockRewardDistribution(ctx, history.BlockHeight)
			require.True(t, found)
			require.True(t, proto.Equal(history, gotHistory),
				"historical routing records must not be rewritten")
			require.True(t, bank.mintedCoins.IsZero())
			require.Empty(t, bank.sentToAccount)
			require.Empty(t, bank.sentToModule)
		})
	}
}
