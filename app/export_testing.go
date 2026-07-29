package app

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
)

// GetBaseApp implements ibctesting.TestingApp.
func (app *ZeroneApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetIBCKeeper implements ibctesting.TestingApp.
func (app *ZeroneApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetTxConfig implements ibctesting.TestingApp.
func (app *ZeroneApp) GetTxConfig() client.TxConfig {
	return app.txConfig
}
