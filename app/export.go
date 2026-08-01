package app

import (
	"encoding/json"
	"fmt"
	"math"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

// ExportAppStateAndValidators exports application state and validators for a
// continuation genesis. Its height is the next InitChain height after the last
// committed block.
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

	ctx := app.NewContext(true)
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
