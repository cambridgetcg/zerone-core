package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/tokens/types"
)

// Keeper manages the tokens module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper types.BankKeeper

	// vestingRewardsKeeper is the chain's single cap-gated mint entry point,
	// wired post-init in app.go. Emission-period minting routes through it so
	// the 222,222,222 ZRN cap is enforced once, chain-wide. Optional: when nil
	// (isolated unit tests) minting falls back to a direct bank mint.
	vestingRewardsKeeper types.VestingRewardsKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new tokens module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bk,
		authority:    authority,
	}
}

// SetVestingRewardsKeeper wires the chain's cap-gated mint entry point into the
// tokens keeper (post-init, app.go). Emission-period minting gates through it.
func (k *Keeper) SetVestingRewardsKeeper(vrk types.VestingRewardsKeeper) { k.vestingRewardsKeeper = vrk }

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		p := types.DefaultParams()
		return &p
	}
	return &params
}

// ---------- Genesis ----------

// genesisJSON is the combined genesis structure for JSON marshal/unmarshal.
type genesisJSON struct {
	Params            *types.Params                  `json:"params,omitempty"`
	TokenEntries      []tokenGenesisEntry            `json:"token_entries,omitempty"`
	DelegationEntries []delegationGenesisEntry       `json:"delegation_entries,omitempty"`
	WrapEntries       []wrapGenesisEntry             `json:"wrap_entries,omitempty"`
	EmissionEntries   []*types.EmissionPeriod        `json:"emission_entries,omitempty"`
}

type tokenGenesisEntry struct {
	Token      *types.TokenDefinition `json:"token"`
	Balances   map[string]string      `json:"balances,omitempty"`
	Allowances map[string]string      `json:"allowances,omitempty"`
}

type delegationGenesisEntry struct {
	TokenId     string            `json:"token_id"`
	Delegations map[string]string `json:"delegations,omitempty"`
	Totals      map[string]string `json:"totals,omitempty"`
}

type wrapGenesisEntry struct {
	TokenId      string `json:"token_id"`
	WrappedDenom string `json:"wrapped_denom"`
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
}

// InitGenesisTokens initializes token state from raw genesis JSON.
func (k Keeper) InitGenesisTokens(ctx sdk.Context, data json.RawMessage) {
	if data == nil {
		return
	}
	var g genesisJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return
	}
	for i := range g.TokenEntries {
		entry := &g.TokenEntries[i]
		k.SetToken(ctx, entry.Token)

		for ownerAddr, amtStr := range entry.Balances {
			bal := new(big.Int)
			if _, ok := bal.SetString(amtStr, 10); ok && bal.Sign() > 0 {
				k.SetBalance(ctx, entry.Token.Id, ownerAddr, bal)
			}
		}

		for key, amtStr := range entry.Allowances {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) != 2 {
				continue
			}
			al := new(big.Int)
			if _, ok := al.SetString(amtStr, 10); ok && al.Sign() > 0 {
				k.SetAllowance(ctx, entry.Token.Id, parts[0], parts[1], al)
			}
		}
	}
}

// InitGenesisDelegations initializes delegation state from raw genesis JSON.
func (k Keeper) InitGenesisDelegations(ctx sdk.Context, data json.RawMessage) {
	if data == nil {
		return
	}
	var g genesisJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return
	}
	for i := range g.DelegationEntries {
		entry := &g.DelegationEntries[i]
		for key, amtStr := range entry.Delegations {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) != 2 {
				continue
			}
			amt := new(big.Int)
			if _, ok := amt.SetString(amtStr, 10); ok && amt.Sign() > 0 {
				k.SetDelegation(ctx, entry.TokenId, parts[0], parts[1], amt)
			}
		}
		for delegator, totalStr := range entry.Totals {
			total := new(big.Int)
			if _, ok := total.SetString(totalStr, 10); ok && total.Sign() > 0 {
				k.SetDelegatorTotal(ctx, entry.TokenId, delegator, total)
			}
		}
	}
}

// InitGenesisWraps initializes wrap record state from raw genesis JSON.
func (k Keeper) InitGenesisWraps(ctx sdk.Context, data json.RawMessage) {
	if data == nil {
		return
	}
	var g genesisJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return
	}
	for i := range g.WrapEntries {
		entry := &g.WrapEntries[i]
		k.SetWrapRecord(ctx, entry.TokenId, entry.WrappedDenom)
	}
}

// InitGenesisEmissions initializes emission period state from raw genesis JSON.
func (k Keeper) InitGenesisEmissions(ctx sdk.Context, data json.RawMessage) {
	if data == nil {
		return
	}
	var g genesisJSON
	if err := json.Unmarshal(data, &g); err != nil {
		return
	}
	for _, emission := range g.EmissionEntries {
		k.SetEmissionPeriod(ctx, emission)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}

// ExportGenesisJSON exports the full genesis state including token entries as raw JSON.
func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	params := k.GetParams(ctx)

	var tokenEntries []tokenGenesisEntry
	k.IterateTokens(ctx, func(token *types.TokenDefinition) bool {
		entry := tokenGenesisEntry{
			Token:      token,
			Balances:   make(map[string]string),
			Allowances: make(map[string]string),
		}

		k.IterateBalancesByToken(ctx, token.Id, func(ownerAddr string, balance *big.Int) bool {
			entry.Balances[ownerAddr] = balance.String()
			return false
		})

		kvStore := k.storeService.OpenKVStore(ctx)
		tokenPrefix := append(types.AllowanceKeyPrefix, []byte(token.Id+"/")...)
		iter, err := kvStore.Iterator(tokenPrefix, prefixEndBytes(tokenPrefix))
		if err == nil {
			defer iter.Close()
			for ; iter.Valid(); iter.Next() {
				suffix := string(iter.Key()[len(tokenPrefix):])
				al := new(big.Int)
				if _, ok := al.SetString(string(iter.Value()), 10); ok && al.Sign() > 0 {
					entry.Allowances[suffix] = al.String()
				}
			}
		}

		tokenEntries = append(tokenEntries, entry)
		return false
	})

	var delegationEntries []delegationGenesisEntry
	k.IterateTokens(ctx, func(token *types.TokenDefinition) bool {
		delegations := make(map[string]string)
		totals := make(map[string]string)

		k.IterateDelegationsByToken(ctx, token.Id, func(delegator, delegate string, amount *big.Int) bool {
			delegations[delegator+"/"+delegate] = amount.String()
			return false
		})

		k.IterateDelegatorTotalsByToken(ctx, token.Id, func(delegator string, total *big.Int) bool {
			totals[delegator] = total.String()
			return false
		})

		if len(delegations) > 0 {
			delegationEntries = append(delegationEntries, delegationGenesisEntry{
				TokenId:     token.Id,
				Delegations: delegations,
				Totals:      totals,
			})
		}
		return false
	})

	var wrapEntries []wrapGenesisEntry
	k.IterateWrapRecords(ctx, func(tokenId, wrappedDenom string) bool {
		wrapEntries = append(wrapEntries, wrapGenesisEntry{
			TokenId:      tokenId,
			WrappedDenom: wrappedDenom,
		})
		return false
	})

	var emissionEntries []*types.EmissionPeriod
	k.IterateEmissionPeriods(ctx, func(emission *types.EmissionPeriod) bool {
		emissionEntries = append(emissionEntries, emission)
		return false
	})

	g := genesisJSON{
		Params:            params,
		TokenEntries:      tokenEntries,
		DelegationEntries: delegationEntries,
		WrapEntries:       wrapEntries,
		EmissionEntries:   emissionEntries,
	}

	bz, err := json.Marshal(g)
	if err != nil {
		panic("failed to marshal genesis: " + err.Error())
	}
	return bz
}

// mintCappedUzrn issues `amount` uzrn into module through the chain's single
// cap-gated mint entry point (x/vesting_rewards.MintWithCap), so emission
// schedules cannot push total supply past the 222,222,222 ZRN cap. Returns the
// amount actually minted. Falls back to a direct mint only when the vesting-
// rewards keeper is unwired (isolated unit tests).
func (k Keeper) mintCappedUzrn(ctx sdk.Context, module string, amount *big.Int) (*big.Int, error) {
	if amount == nil || amount.Sign() <= 0 {
		return new(big.Int), nil
	}
	if k.vestingRewardsKeeper != nil {
		return k.vestingRewardsKeeper.MintWithCap(ctx, module, amount)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(amount)))
	if err := k.bankKeeper.MintCoins(ctx, module, coins); err != nil {
		return nil, err
	}
	return amount, nil
}

// ---------- BeginBlocker ----------

// BeginBlocker processes active emission periods, minting tokens for the current block.
func (k Keeper) BeginBlocker(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())

	k.IterateEmissionPeriods(ctx, func(emission *types.EmissionPeriod) bool {
		if !emission.Active {
			return false
		}

		if currentBlock < emission.StartBlock || currentBlock > emission.EndBlock {
			// Deactivate if past end
			if currentBlock > emission.EndBlock {
				emission.Active = false
				k.SetEmissionPeriod(ctx, emission)
			}
			return false
		}

		// Mint amount_per_block to recipient
		amountPerBlock := new(big.Int)
		if _, ok := amountPerBlock.SetString(emission.AmountPerBlock, 10); !ok || amountPerBlock.Sign() <= 0 {
			return false
		}

		recipientAddr, err := sdk.AccAddressFromBech32(emission.Recipient)
		if err != nil {
			return false
		}

		minted, err := k.mintCappedUzrn(ctx, types.ModuleName, amountPerBlock)
		if err != nil || minted.Sign() <= 0 {
			return false
		}
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(minted)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, coins); err != nil {
			return false
		}

		// Track total emitted (actually-minted, honouring the supply cap)
		totalEmitted := new(big.Int)
		if emission.TotalEmitted != "" {
			totalEmitted.SetString(emission.TotalEmitted, 10)
		}
		totalEmitted.Add(totalEmitted, minted)
		emission.TotalEmitted = totalEmitted.String()
		k.SetEmissionPeriod(ctx, emission)

		return false
	})
}
