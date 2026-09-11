package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func claimRecordsAdmissionFixture(t *testing.T) (*ZeroneApp, sdk.Context, *knowledgetypes.MsgSubmitContradiction) {
	t.Helper()
	app, ctx := settlementFixture(t)
	address := settlementAddress(211)
	fund := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3000000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, fund))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, address, fund))
	fact := &knowledgetypes.Fact{Id: "counter-target", Domain: "physics", Category: "empirical", Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, Content: "A test observation", Submitter: address.String()}
	require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, fact))
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	return app, ctx, &knowledgetypes.MsgSubmitContradiction{Submitter: address.String(), FactId: fact.Id, CounterClaim: "A repeat experiment produced a counterexample.", Stake: "1000000", Category: "empirical", Reason: "  Retain this exact argument α.\n", EvidenceIds: []string{"sha256:observation-b", "fact:observation-a"}}
}

func claimRecordsStores(t *testing.T, app *ZeroneApp, ctx sdk.Context) map[string][]byte {
	t.Helper()
	rows := make(map[string][]byte)
	for _, name := range []string{knowledgetypes.StoreKey, "bank", "acc"} {
		it := ctx.KVStore(app.keys[name]).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			rows[name+":"+string(it.Key())] = bytes.Clone(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return rows
}

func TestClaimRecordsActualBankRetainsInputsAndRefusesOverwrite(t *testing.T) {
	for _, reason := range []string{"  Retain this exact argument α.\n", ""} {
		t.Run(fmt.Sprintf("reason-bytes-%d", len(reason)), func(t *testing.T) {
			app, ctx, msg := claimRecordsAdmissionFixture(t)
			msg.Reason = reason
			supply := app.BankKeeper.GetSupply(ctx, "uzrn")
			module := authtypes.NewModuleAddress(knowledgetypes.ModuleName)
			beforeModule := app.BankKeeper.GetBalance(ctx, module, "uzrn")
			server := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper)
			response, err := server.SubmitContradiction(ctx, msg)
			require.NoError(t, err)
			claim, found := app.KnowledgeKeeper.GetClaim(ctx, response.CounterFactId)
			require.True(t, found)
			require.Equal(t, msg.Reason, claim.ArgumentText)
			require.Equal(t, msg.EvidenceIds, claim.EvidenceIds)
			require.Equal(t, msg.CounterClaim, claim.FactContent)
			require.Equal(t, msg.Stake, claim.Stake)
			require.Equal(t, knowledgetypes.ReviewPolicyNeutral, claim.ReviewPolicyVersion)
			require.Equal(t, "2000000", app.BankKeeper.GetBalance(ctx, settlementAddress(211), "uzrn").Amount.String())
			require.Equal(t, "1000000", app.BankKeeper.GetBalance(ctx, module, "uzrn").Amount.Sub(beforeModule.Amount).String())
			require.Equal(t, supply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			target, found := app.KnowledgeKeeper.GetFact(ctx, msg.FactId)
			require.True(t, found)
			require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_CONTESTED, target.Status)
			round, found := app.KnowledgeKeeper.GetVerificationRound(ctx, claim.VerificationRoundId)
			require.True(t, found)
			require.Equal(t, claim.Id, round.ClaimId)
			require.Equal(t, knowledgetypes.CommitmentSchemeReviewV2, round.CommitmentScheme)
			// The keeper owns its stored payload, not the caller's slice.
			original := bytes.Clone(ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get(knowledgetypes.ClaimKey(claim.Id)))
			msg.EvidenceIds[0] = "replacement"
			msg.Reason = "replacement argument"
			before := claimRecordsStores(t, app, ctx)
			events := ctx.EventManager().Events()
			_, err = server.SubmitContradiction(ctx, msg)
			require.ErrorContains(t, err, "identity already exists")
			require.Equal(t, before, claimRecordsStores(t, app, ctx))
			require.Equal(t, events, ctx.EventManager().Events())
			require.Equal(t, original, ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Get(knowledgetypes.ClaimKey(claim.Id)))
		})
	}
}

func TestClaimRecordsInvalidInputsDoNotMoveCollateral(t *testing.T) {
	for _, scenario := range []string{"reason length", "reason UTF8", "too many evidence", "evidence length", "empty evidence", "duplicate evidence", "unknown fields"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, msg := claimRecordsAdmissionFixture(t)
			switch scenario {
			case "reason length":
				msg.Reason = strings.Repeat("x", knowledgetypes.MaxReviewReasonBytes+1)
			case "reason UTF8":
				msg.Reason = string([]byte{0xff})
			case "too many evidence":
				msg.EvidenceIds = make([]string, knowledgetypes.MaxEvidenceReferences+1)
			case "evidence length":
				msg.EvidenceIds = []string{strings.Repeat("x", knowledgetypes.MaxEvidenceReferenceBytes+1)}
			case "empty evidence":
				msg.EvidenceIds = []string{" "}
			case "duplicate evidence":
				msg.EvidenceIds = []string{"same", "same"}
			case "unknown fields":
				msg.ProtoReflect().SetUnknown(protowire.AppendVarint(protowire.AppendTag(nil, 99, protowire.VarintType), 1))
			}
			before := claimRecordsStores(t, app, ctx)
			_, err := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper).SubmitContradiction(ctx, msg)
			require.Error(t, err)
			require.Equal(t, before, claimRecordsStores(t, app, ctx))
			require.Empty(t, ctx.EventManager().Events())
		})
	}
}

type claimRecordFaultService struct {
	corestore.KVStoreService
	op     string
	prefix []byte
	after  bool
}

func (s claimRecordFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return claimRecordFaultStore{s.KVStoreService.OpenKVStore(ctx), s.op, s.prefix, s.after}
}

type claimRecordFaultStore struct {
	corestore.KVStore
	op     string
	prefix []byte
	after  bool
}

func (s claimRecordFaultStore) Get(key []byte) ([]byte, error) {
	if s.op == "get" && bytes.HasPrefix(key, s.prefix) {
		return nil, errors.New("injected claim-record read failure")
	}
	return s.KVStore.Get(key)
}

func (s claimRecordFaultStore) Set(key, value []byte) error {
	if s.op == "set" && bytes.HasPrefix(key, s.prefix) {
		if s.after {
			if err := s.KVStore.Set(key, value); err != nil {
				return err
			}
		}
		return errors.New("injected claim-record write failure")
	}
	return s.KVStore.Set(key, value)
}

type claimRecordAfterBankFailure struct{ knowledgetypes.BankKeeper }

func (b claimRecordAfterBankFailure) SendCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, module string, amount sdk.Coins) error {
	if err := b.BankKeeper.SendCoinsFromAccountToModule(ctx, from, module, amount); err != nil {
		return err
	}
	return errors.New("injected error after actual bank transfer")
}

func TestClaimRecordsActualBankAndRecordFailuresRollBack(t *testing.T) {
	for _, scenario := range []struct {
		name, op string
		prefix   []byte
	}{
		{"activation read", "get", []byte(knowledgekeeper.ClaimRecordsEnabledStoreKey)},
		{"target read", "get", knowledgetypes.FactKeyPrefix},
		{"prior claim read", "get", knowledgetypes.ClaimKeyPrefix},
		{"claim write", "set", knowledgetypes.ClaimKeyPrefix},
		{"round write", "set", knowledgetypes.VerificationRoundKeyPrefix},
		{"target write", "set", knowledgetypes.FactKeyPrefix},
		{"target history write", "set", knowledgetypes.StatusTransitionKeyPrefix},
		{"bank transfer", "bank", nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			app, ctx, msg := claimRecordsAdmissionFixture(t)
			before := claimRecordsStores(t, app, ctx)
			var bank knowledgetypes.BankKeeper = app.BankKeeper
			if scenario.op == "bank" {
				bank = claimRecordAfterBankFailure{bank}
			}
			service := claimRecordFaultService{runtime.NewKVStoreService(app.keys[knowledgetypes.StoreKey]), scenario.op, scenario.prefix, true}
			k := knowledgekeeper.NewKeeper(service, app.appCodec, app.KnowledgeKeeper.GetAuthority(), bank, nil)
			_, err := knowledgekeeper.NewMsgServerImpl(k).SubmitContradiction(ctx, msg)
			require.Error(t, err)
			require.Equal(t, before, claimRecordsStores(t, app, ctx))
			require.Empty(t, ctx.EventManager().Events())
			response, err := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper).SubmitContradiction(ctx, msg)
			require.NoError(t, err)
			claim, found := app.KnowledgeKeeper.GetClaim(ctx, response.CounterFactId)
			require.True(t, found)
			require.Equal(t, msg.Reason, claim.ArgumentText)
		})
	}
}

func TestClaimRecordsLegacyBranchDoesNotInventRetainedInputs(t *testing.T) {
	app, ctx, msg := claimRecordsAdmissionFixture(t)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
	response, err := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper).SubmitContradiction(ctx, msg)
	require.NoError(t, err)
	claim, found := app.KnowledgeKeeper.GetClaim(ctx, response.CounterFactId)
	require.True(t, found)
	require.Empty(t, claim.ArgumentText)
	require.Empty(t, claim.EvidenceIds)
	before := proto.Clone(claim).(*knowledgetypes.Claim)
	require.NoError(t, app.KnowledgeKeeper.EnableClaimRecords(ctx))
	claim, found = app.KnowledgeKeeper.GetClaim(ctx, claim.Id)
	require.True(t, found)
	require.True(t, proto.Equal(before, claim), "activation cannot recover dropped historical fields")
}
