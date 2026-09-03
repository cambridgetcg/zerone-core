package schedule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/zerone-chain/zerone/x/schedule/client/cli"
	"github.com/zerone-chain/zerone/x/schedule/keeper"
	"github.com/zerone-chain/zerone/x/schedule/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
	_ appmodule.AppModule   = AppModule{}
)

type AppModuleBasic struct{ cdc codec.Codec }

func (AppModuleBasic) Name() string                                    { return types.ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) { types.RegisterCodec(cdc) }
func (AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}
func (AppModuleBasic) GetTxCmd() *cobra.Command                                        { return cli.NewTxCmd() }
func (AppModuleBasic) GetQueryCmd() *cobra.Command                                     { return cli.NewQueryCmd() }

func (AppModuleBasic) DefaultGenesis(codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(types.DefaultGenesis())
	if err != nil {
		panic("marshal schedule genesis: " + err.Error())
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	genesis, err := decodeGenesisStrict(bz)
	if err != nil {
		return fmt.Errorf("unmarshal %s genesis: %w", types.ModuleName, err)
	}
	return genesis.Validate()
}

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{AppModuleBasic: AppModuleBasic{cdc: cdc}, keeper: keeper}
}

func (AppModule) IsOnePerModuleType() {}
func (AppModule) IsAppModule()        {}

func (am AppModule) RegisterServices(configurator module.Configurator) {
	types.RegisterMsgServer(configurator.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(configurator.QueryServer(), keeper.NewQueryServerImpl(am.keeper))
}

func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) {
	genesis, err := decodeGenesisStrict(data)
	if err != nil {
		panic("unmarshal schedule genesis: " + err.Error())
	}
	am.keeper.InitGenesis(ctx, genesis)
}

func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	bz, err := json.Marshal(am.keeper.ExportGenesis(ctx))
	if err != nil {
		panic("marshal schedule genesis: " + err.Error())
	}
	return bz
}

func (AppModule) ConsensusVersion() uint64 { return 1 }

func decodeGenesisStrict(data json.RawMessage) (*types.GenesisState, error) {
	if err := rejectDuplicateGenesisJSONKeys(data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var genesis types.GenesisState
	if err := decoder.Decode(&genesis); err != nil {
		return nil, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("multiple JSON values are not allowed")
	}
	return &genesis, nil
}

func rejectDuplicateGenesisJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanUniqueGenesisJSONValue(decoder, 0); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values are not allowed; found trailing token %v", token)
	}
	return nil
}

func scanUniqueGenesisJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 256 {
		return errors.New("JSON nesting exceeds 256 levels")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := scanUniqueGenesisJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return errors.New("JSON object did not end with }")
		}
	case '[':
		for decoder.More() {
			if err := scanUniqueGenesisJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return errors.New("JSON array did not end with ]")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

// BeginBlock makes the same-height ordering explicit: committed due work runs
// before DeliverTx, so a same-block amendment or cancellation must use the
// post-execution revision/count or its compare-and-swap check fails.
func (am AppModule) BeginBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	_, err := am.keeper.ProcessDueSchedules(ctx)
	return err
}

func (AppModule) EndBlock(context.Context) error { return nil }
