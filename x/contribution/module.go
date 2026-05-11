package contribution

import (
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/zerone-chain/zerone/x/contribution/keeper"
	"github.com/zerone-chain/zerone/x/contribution/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
	_ appmodule.AppModule   = AppModule{}
)

// AppModuleBasic implements module.AppModuleBasic.
type AppModuleBasic struct {
	cdc codec.Codec
}

func NewAppModuleBasic(cdc codec.Codec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

func (AppModuleBasic) Name() string { return types.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the module's default genesis as protojson.
//
// Why protojson, not encoding/json: Contribution carries a protobuf
// oneof (ContributionPayload.Payload). Stdlib encoding/json cannot
// unmarshal into the oneof interface field on round-trip — it would
// emit object-shaped JSON for the variant and then fail on decode.
// protojson is the canonical JSON form for modern protoc-gen-go
// messages and handles oneofs correctly. This is load-bearing the
// moment the chain writes non-empty Contributions into genesis (which
// happens once any adapter is registered and InitGenesis emits a
// Substrate-class Contribution per Layer 2 of the UW recursion stack).
func (a AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	bz, err := protojson.Marshal(types.DefaultGenesis())
	if err != nil {
		panic("failed to marshal default genesis: " + err.Error())
	}
	return bz
}

func (a AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, raw json.RawMessage) error {
	gs := &types.GenesisState{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, gs); err != nil {
		return fmt.Errorf("failed to unmarshal x/%s genesis: %w", types.ModuleName, err)
	}
	return gs.Validate()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}
func (AppModuleBasic) GetTxCmd() *cobra.Command                                         { return nil }
func (AppModuleBasic) GetQueryCmd() *cobra.Command                                      { return nil }

// AppModule implements module.AppModule.
type AppModule struct {
	AppModuleBasic
	keeper *keeper.Keeper
}

func NewAppModule(cdc codec.Codec, k *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         k,
	}
}

func (AppModule) IsAppModule()        {}
func (AppModule) IsOnePerModuleType() {}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))
}

// InitGenesis writes the contribution records from genesis state.
//
// Decoded via protojson — see DefaultGenesis for rationale.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, raw json.RawMessage) {
	gs := &types.GenesisState{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, gs); err != nil {
		panic("failed to unmarshal x/contribution genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, gs)
}

// ExportGenesis returns the current contribution set as protojson —
// see DefaultGenesis for rationale.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, err := protojson.Marshal(gs)
	if err != nil {
		panic("failed to marshal x/contribution genesis: " + err.Error())
	}
	return bz
}

// ConsensusVersion returns the module's consensus version.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }
