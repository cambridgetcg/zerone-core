package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	sponsorshipkeeper "github.com/zerone-chain/zerone/x/sponsorship/keeper"
	sponsorshiptypes "github.com/zerone-chain/zerone/x/sponsorship/types"
)

// This fixture uses the installed SDK bank and real keeper stores. The Fact is
// synthetic eligibility input, not evidence of admission, review or truth.
func sponsorshipSettlementFixture(t *testing.T) (*ZeroneApp, sdk.Context, *sponsorshiptypes.MsgFulfillBounty) {
	t.Helper()
	app := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID("sponsorship-settlement-fixture"), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, app.Close()) })
	genesis, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, app))
	require.NoError(t, err)
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: "sponsorship-settlement-fixture"})
	_, err = app.InitChainer(ctx, &abci.RequestInitChain{ChainId: ctx.ChainID(), AppStateBytes: genesis})
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(100)
	sponsor, worker, caller := settlementAddress(231), settlementAddress(232), settlementAddress(233)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 300))
	// Test issuance only. The normal bounty creation handler locks this escrow.
	require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, sponsor, coins))
	created, err := sponsorshipkeeper.NewMsgServerImpl(app.SponsorshipKeeper).CreateBountyOrder(ctx,
		&sponsorshiptypes.MsgCreateBountyOrder{Sponsor: sponsor.String(), Domain: "physics", PricePerArtifact: "100", TargetCount: 3, DurationBlocks: 500})
	require.NoError(t, err)
	require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, &knowledgetypes.Fact{
		Id: "sponsorship-fixture-fact", Domain: "physics", Submitter: worker.String(),
		Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, SubmittedAtBlock: 100,
	}))
	return app, ctx.WithEventManager(sdk.NewEventManager()), &sponsorshiptypes.MsgFulfillBounty{
		Caller: caller.String(), BountyId: created.BountyId, FactId: "sponsorship-fixture-fact",
	}
}

func sponsorshipSettlementStores(t *testing.T, app *ZeroneApp, ctx sdk.Context) map[string][]byte {
	t.Helper()
	rows := make(map[string][]byte)
	for _, name := range []string{sponsorshiptypes.StoreKey, knowledgetypes.StoreKey, "bank", "acc"} {
		it := ctx.KVStore(app.keys[name]).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			rows[name+":"+string(it.Key())] = bytes.Clone(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return rows
}

var errSponsorshipSettlementWrite = errors.New("injected sponsorship settlement write failure")

type sponsorshipSettlementFaultService struct {
	corestore.KVStoreService
	key   []byte
	after bool
	hits  *int
}

func (s sponsorshipSettlementFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return sponsorshipSettlementFaultStore{s.KVStoreService.OpenKVStore(ctx), s}
}

type sponsorshipSettlementFaultStore struct {
	corestore.KVStore
	fault sponsorshipSettlementFaultService
}

func (s sponsorshipSettlementFaultStore) Set(key, value []byte) error {
	if bytes.Equal(key, s.fault.key) {
		*s.fault.hits++
		if s.fault.after {
			if err := s.KVStore.Set(key, value); err != nil {
				return err
			}
		}
		return errSponsorshipSettlementWrite
	}
	return s.KVStore.Set(key, value)
}

func TestSponsorshipSettlementWriteFailuresRollBack(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, record := range []string{"order", "fulfillment"} {
			for _, afterWrite := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/after-%t/nested-%t", record, afterWrite, nested), func(t *testing.T) {
					app, ctx, msg := sponsorshipSettlementFixture(t)
					before := sponsorshipSettlementStores(t, app, ctx)
					callCtx, commit := ctx, func() {}
					if nested {
						callCtx, commit = ctx.CacheContext()
					}
					key := sponsorshiptypes.FulfillmentKey(msg.BountyId, msg.FactId)
					if record == "order" {
						key = sponsorshiptypes.BountyOrderKey(msg.BountyId)
					}
					hits := 0
					service := sponsorshipSettlementFaultService{
						KVStoreService: runtime.NewKVStoreService(app.keys[sponsorshiptypes.StoreKey]),
						key:            key, after: afterWrite, hits: &hits,
					}
					faulted := sponsorshipkeeper.NewKeeper(service, app.appCodec, app.BankKeeper, app.KnowledgeKeeper)
					response, err := sponsorshipkeeper.NewMsgServerImpl(faulted).FulfillBounty(callCtx, msg)
					require.Equal(t, 1, hits, "must reach the exact post-payment record write")
					require.ErrorIs(t, err, errSponsorshipSettlementWrite)
					require.Nil(t, response)
					require.Equal(t, before, sponsorshipSettlementStores(t, app, callCtx))
					require.Empty(t, callCtx.EventManager().Events())
					require.Equal(t, before, sponsorshipSettlementStores(t, app, ctx), "balances, accounts, escrow and consumption record must all roll back")
					require.Empty(t, ctx.EventManager().Events(), "no payment or fulfillment event may escape a failed batch")

					server := sponsorshipkeeper.NewMsgServerImpl(app.SponsorshipKeeper)
					response, err = server.FulfillBounty(callCtx, msg)
					require.NoError(t, err)
					if nested {
						require.Equal(t, before, sponsorshipSettlementStores(t, app, ctx), "success must remain inside the caller's transaction until it commits")
						require.Empty(t, ctx.EventManager().Events())
					}
					commit()
					require.Equal(t, "100", response.AmountPaid)
					worker, err := sdk.AccAddressFromBech32(response.Worker)
					require.NoError(t, err)
					require.Equal(t, "100", app.BankKeeper.GetBalance(ctx, worker, "uzrn").Amount.String())
					require.Equal(t, "200", app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(sponsorshiptypes.ModuleName), "uzrn").Amount.String())
					after := sponsorshipSettlementStores(t, app, ctx)
					events := ctx.EventManager().Events()
					_, err = server.FulfillBounty(ctx, msg)
					require.ErrorIs(t, err, sponsorshiptypes.ErrAlreadyFulfilled)
					require.Equal(t, after, sponsorshipSettlementStores(t, app, ctx))
					require.Equal(t, events, ctx.EventManager().Events())
				})
			}
		}
	}
}

// The wrapper faults only after the actual bank has transferred funds and
// created the recipient account; it is not a balance-counting mock.
type sponsorshipSettlementBankFault struct{ sponsorshiptypes.BankKeeper }

func (b sponsorshipSettlementBankFault) SendCoinsFromModuleToAccount(ctx context.Context, module string, recipient sdk.AccAddress, amount sdk.Coins) error {
	if err := b.BankKeeper.SendCoinsFromModuleToAccount(ctx, module, recipient, amount); err != nil {
		return err
	}
	return errSponsorshipSettlementWrite
}

func TestSponsorshipSettlementBankFailureRollsBack(t *testing.T) {
	app, ctx, msg := sponsorshipSettlementFixture(t)
	before := sponsorshipSettlementStores(t, app, ctx)
	faulted := sponsorshipkeeper.NewKeeper(runtime.NewKVStoreService(app.keys[sponsorshiptypes.StoreKey]), app.appCodec,
		sponsorshipSettlementBankFault{app.BankKeeper}, app.KnowledgeKeeper)
	response, err := sponsorshipkeeper.NewMsgServerImpl(faulted).FulfillBounty(ctx, msg)
	require.ErrorIs(t, err, errSponsorshipSettlementWrite)
	require.Nil(t, response)
	require.Equal(t, before, sponsorshipSettlementStores(t, app, ctx))
	require.Empty(t, ctx.EventManager().Events())
	_, err = sponsorshipkeeper.NewMsgServerImpl(app.SponsorshipKeeper).FulfillBounty(ctx, msg)
	require.NoError(t, err)
	require.Equal(t, "100", app.BankKeeper.GetBalance(ctx, settlementAddress(232), "uzrn").Amount.String())
}

func TestSponsorshipSettlementSuccessCompatibility(t *testing.T) {
	app, ctx, msg := sponsorshipSettlementFixture(t)
	server := sponsorshipkeeper.NewMsgServerImpl(app.SponsorshipKeeper)
	supply := app.BankKeeper.GetSupply(ctx, "uzrn")
	// Captured from pre-repair 7a68ca51 with these same inputs. The digest
	// covers response JSON, ordered events and changed raw store bytes (not AppHash).
	expected := []struct {
		gas    uint64
		digest string
	}{
		{43684, "bad261026940129d5e56ad468affb3cffaae791b0d3e60da1362265061ba3800"},
		{30157, "657a022c81a8a623df805fbdd54a4d99e4aa47170051e74ec225e36227f8a86d"},
		{26417, "31c234587a9b7be712400055c6984a9e87f0560d34ce561bb793180e98cb3def"},
	}
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("fulfillment-%d", i+1), func(t *testing.T) {
			if i > 0 {
				fact, found := app.KnowledgeKeeper.GetFact(ctx, msg.FactId)
				require.True(t, found)
				msg.FactId = fmt.Sprintf("sponsorship-fixture-fact-%d", i+1)
				fact.Id = msg.FactId
				require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, fact))
			}
			before := sponsorshipSettlementStores(t, app, ctx)
			ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			response, err := server.FulfillBounty(ctx, msg)
			gas := ctx.GasMeter().GasConsumed()
			require.NoError(t, err)
			require.Equal(t, settlementAddress(232).String(), response.Worker, "pay the fact author, not the caller")
			require.Equal(t, "100", response.AmountPaid)
			require.Equal(t, i == 2, response.BountyNowFulfilled)
			require.Equal(t, int64(100*(i+1)), app.BankKeeper.GetBalance(ctx, settlementAddress(232), "uzrn").Amount.Int64())
			require.True(t, app.BankKeeper.GetBalance(ctx, settlementAddress(233), "uzrn").IsZero())
			require.Equal(t, int64(100*(2-i)), app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(sponsorshiptypes.ModuleName), "uzrn").Amount.Int64())
			require.Equal(t, supply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			order, found := app.SponsorshipKeeper.GetBountyOrder(ctx, msg.BountyId)
			require.True(t, found)
			require.Equal(t, uint32(i+1), order.FulfilledCount)
			require.Equal(t, fmt.Sprint(100*(2-i)), order.EscrowRemaining)
			expectedStatus := sponsorshiptypes.BountyStatus_BOUNTY_STATUS_ACTIVE
			if i == 2 {
				expectedStatus = sponsorshiptypes.BountyStatus_BOUNTY_STATUS_FULFILLED
			}
			require.Equal(t, expectedStatus, order.Status)
			record, found := app.SponsorshipKeeper.GetFulfillment(ctx, msg.BountyId, msg.FactId)
			require.True(t, found)
			require.Equal(t, response.Worker, record.Worker)
			require.Equal(t, response.AmountPaid, record.AmountPaid)
			require.Equal(t, uint64(100), record.FulfilledAtBlock)

			after := sponsorshipSettlementStores(t, app, ctx)
			delta := make(map[string][]byte)
			for key, value := range after {
				if !bytes.Equal(before[key], value) {
					delta[hex.EncodeToString([]byte(key))] = value
				}
			}
			for key := range before {
				if _, found := after[key]; !found {
					delta[hex.EncodeToString([]byte(key))] = nil
				}
			}
			receipt, err := json.Marshal(struct {
				Response *sponsorshiptypes.MsgFulfillBountyResponse
				Events   sdk.Events
				Delta    map[string][]byte
			}{response, ctx.EventManager().Events(), delta})
			require.NoError(t, err)
			require.Equal(t, expected[i].gas, gas, "successful-path gas must remain unchanged")
			require.Equal(t, expected[i].digest, fmt.Sprintf("%x", sha256.Sum256(receipt)), "response, events and store delta must remain unchanged")
			events := ctx.EventManager().Events()
			_, err = server.FulfillBounty(ctx, msg)
			expectedErr := sponsorshiptypes.ErrAlreadyFulfilled
			if i == 2 {
				expectedErr = sponsorshiptypes.ErrBountyNotActive
			}
			require.ErrorIs(t, err, expectedErr)
			require.Equal(t, after, sponsorshipSettlementStores(t, app, ctx))
			require.Equal(t, events, ctx.EventManager().Events())
		})
	}
}
