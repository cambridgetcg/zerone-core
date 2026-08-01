package app

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/core/address"
	upgrademodule "cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// singletonUpgradeAppModule preserves the upstream x/upgrade module behavior
// while replacing only its Msg service. Upstream ScheduleUpgrade intentionally
// overwrites an existing plan; Zerone requires an explicit MsgCancelUpgrade
// governance decision before a different plan can be scheduled.
type singletonUpgradeAppModule struct {
	upgrademodule.AppModule
	keeper *upgradekeeper.Keeper
}

func newSingletonUpgradeAppModule(
	keeper *upgradekeeper.Keeper,
	accountAddressCodec address.Codec,
) singletonUpgradeAppModule {
	return singletonUpgradeAppModule{
		AppModule: upgrademodule.NewAppModule(keeper, accountAddressCodec),
		keeper:    keeper,
	}
}

// RegisterServices mirrors the upstream module registration so migrations and
// queries remain byte-for-byte SDK behavior. Only the Msg server is wrapped.
func (am singletonUpgradeAppModule) RegisterServices(cfg module.Configurator) {
	delegate := upgradekeeper.NewMsgServerImpl(am.keeper)
	upgradetypes.RegisterMsgServer(cfg.MsgServer(), singletonUpgradeMsgServer{
		keeper:   am.keeper,
		delegate: delegate,
	})
	upgradetypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	migrator := upgradekeeper.NewMigrator(am.keeper)
	if err := cfg.RegisterMigration(
		upgradetypes.ModuleName,
		1,
		migrator.Migrate1to2,
	); err != nil {
		panic(fmt.Sprintf(
			"failed to migrate x/%s from version 1 to 2: %v",
			upgradetypes.ModuleName,
			err,
		))
	}
}

type singletonUpgradeMsgServer struct {
	keeper   *upgradekeeper.Keeper
	delegate upgradetypes.MsgServer
}

func (s singletonUpgradeMsgServer) SoftwareUpgrade(
	ctx context.Context,
	msg *upgradetypes.MsgSoftwareUpgrade,
) (*upgradetypes.MsgSoftwareUpgradeResponse, error) {
	if s.keeper == nil || s.delegate == nil {
		return nil, errors.New("SDK upgrade Msg server is not configured")
	}
	existing, err := s.keeper.GetUpgradePlan(ctx)
	switch {
	case err == nil:
		return nil, fmt.Errorf(
			"refusing to overwrite scheduled SDK upgrade %q at height %d; governance must pass MsgCancelUpgrade before scheduling a replacement",
			existing.Name,
			existing.Height,
		)
	case !errors.Is(err, upgradetypes.ErrNoUpgradePlanFound):
		return nil, fmt.Errorf("read scheduled SDK upgrade: %w", err)
	}
	return s.delegate.SoftwareUpgrade(ctx, msg)
}

func (s singletonUpgradeMsgServer) CancelUpgrade(
	ctx context.Context,
	msg *upgradetypes.MsgCancelUpgrade,
) (*upgradetypes.MsgCancelUpgradeResponse, error) {
	if s.delegate == nil {
		return nil, errors.New("SDK upgrade Msg server is not configured")
	}
	return s.delegate.CancelUpgrade(ctx, msg)
}

var _ upgradetypes.MsgServer = singletonUpgradeMsgServer{}
