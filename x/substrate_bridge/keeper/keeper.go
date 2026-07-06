package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	authority string

	knowledgeKeeper      types.KnowledgeKeeper
	qualificationKeeper  types.QualificationKeeper
	bankKeeper           types.BankKeeper
	accountKeeper        types.AccountKeeper
	vestingRewardsKeeper types.VestingRewardsKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority string,
	kk types.KnowledgeKeeper,
	qk types.QualificationKeeper,
	bk types.BankKeeper,
	ak types.AccountKeeper,
	vrk types.VestingRewardsKeeper,
) Keeper {
	return Keeper{cdc: cdc, storeKey: storeKey, authority: authority,
		knowledgeKeeper: kk, qualificationKeeper: qk, bankKeeper: bk, accountKeeper: ak,
		vestingRewardsKeeper: vrk}
}

func (k Keeper) Authority() string { return k.authority }
func (k Keeper) Logger(ctx interface{ Logger() log.Logger }) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

