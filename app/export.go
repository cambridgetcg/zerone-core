package app

import (
	"encoding/json"
	"fmt"
	"math"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

// ExportAppStateAndValidators exports application state and validators. Its
// height is the next InitChain height after the last committed block. Normal
// exports are continuation genesis; the narrowly accepted historical
// zerone-1 auth shape is archival evidence and remains intentionally invalid
// as fresh genesis.
func (app *ZeroneApp) ExportAppStateAndValidators(
	forZeroHeight bool,
	jailAllowedAddrs []string,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	if forZeroHeight {
		return servertypes.ExportedApp{}, fmt.Errorf(
			"zero-height export is disabled: the current implementation cannot safely rebase staking, slashing, unbonding, redelegation, emergency, and IBC clocks; use an explicitly reviewed social-fork genesis pipeline",
		)
	}
	_ = jailAllowedAddrs

	chainID := app.ChainID()
	if chainID == "" {
		return servertypes.ExportedApp{}, fmt.Errorf(
			"state export requires a non-empty BaseApp chain ID",
		)
	}
	// BaseApp.NewContext is a test helper that deliberately starts from an
	// empty header. Bind the BaseApp chain ID explicitly so module exporters
	// can make exact historical-chain compatibility decisions.
	ctx := app.NewContext(true).WithChainID(chainID)
	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}
	appState, err := json.Marshal(genState)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}
	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}
	if app.LastBlockHeight() == math.MaxInt64 {
		return servertypes.ExportedApp{}, fmt.Errorf("export height overflows int64")
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          app.LastBlockHeight() + 1,
		ConsensusParams: app.GetConsensusParams(ctx),
	}, nil
}
