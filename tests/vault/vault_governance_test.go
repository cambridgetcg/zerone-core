package vault_test

import (
	"encoding/json"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/log"

	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	sdkstakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
	zeronegovkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
)

const testChainID = "zerone-vault-test-1"

// ---------- Test Harness (minimal, vault-specific) ----------

type vaultTestHarness struct {
	t             *testing.T
	app           *zeroneapp.ZeroneApp
	ctx           sdk.Context
	govKeeper     zeronegovkeeper.Keeper
	bankKeeper    bankkeeper.Keeper
	accountKeeper authkeeper.AccountKeeper
	currentHeight int64
}

func testAddr(seed string) sdk.AccAddress {
	return sdk.AccAddress([]byte(seed + "________________")[:20])
}

func newVaultTestHarness(t *testing.T) *vaultTestHarness {
	t.Helper()

	db := dbm.NewMemDB()
	app := zeroneapp.NewZeroneApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID(testChainID),
	)

	genState := app.DefaultGenesis()
	genState = genesisStateWithValSet(t, app, genState)
	stateBytes, err := json.Marshal(genState)
	require.NoError(t, err)

	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:         testChainID,
		AppStateBytes:   stateBytes,
		ConsensusParams: simtestutil.DefaultConsensusParams,
	})
	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	h := &vaultTestHarness{
		t:             t,
		app:           app,
		govKeeper:     app.ZeroneGovKeeper,
		bankKeeper:    app.BankKeeper,
		accountKeeper: app.AccountKeeper,
		currentHeight: 1,
	}

	h.ctx = app.NewContext(true).
		WithBlockHeight(h.currentHeight).
		WithChainID(testChainID).
		WithBlockHeader(cmtproto.Header{
			Height:  h.currentHeight,
			ChainID: testChainID,
		})

	return h
}

func (h *vaultTestHarness) advanceBlocks(n int) {
	for i := 0; i < n; i++ {
		h.currentHeight++
		h.ctx = h.app.NewContext(true).
			WithBlockHeight(h.currentHeight).
			WithChainID(testChainID).
			WithBlockHeader(cmtproto.Header{
				Height:  h.currentHeight,
				ChainID: testChainID,
			})

		h.app.BeginBlocker(h.ctx)
		h.app.EndBlocker(h.ctx)
	}
}

func (h *vaultTestHarness) fundAccount(addr sdk.AccAddress, amount sdk.Coins) error {
	moduleName := zeroneauthtypes.ModuleName
	if err := h.app.BankKeeper.MintCoins(h.ctx, moduleName, amount); err != nil {
		return err
	}
	return h.app.BankKeeper.SendCoinsFromModuleToAccount(h.ctx, moduleName, addr, amount)
}

func (h *vaultTestHarness) setupVoters() (founder, aiVault string) {
	founder = testAddr("founder").String()
	aiVault = testAddr("ai-vault").String()

	// Set short periods for testing.
	params := h.govKeeper.GetParams(h.ctx)
	params.ResearchDiscussionBlocks = 5
	params.ResearchVotingBlocks = 10
	h.govKeeper.SetParams(h.ctx, params)

	// Configure the 2-of-2 voters.
	h.govKeeper.SetResearchFundVoters(h.ctx, &zeronegovtypes.ResearchFundVoters{
		Voter1: founder,
		Voter2: aiVault,
	})

	return founder, aiVault
}

func (h *vaultTestHarness) submitProposal(proposer string) uint64 {
	resp, err := h.govKeeper.SubmitResearchSpend(h.ctx, &zeronegovtypes.MsgSubmitResearchSpend{
		Proposer:      proposer,
		Title:         "Fund AI safety research",
		Description:   "Grant for adversarial testing framework",
		Recipient:     testAddr("researcher").String(),
		Amount:        "1000000000",
		Justification: "Critical safety work",
	})
	require.NoError(h.t, err)
	return resp.ProposalId
}

func (h *vaultTestHarness) advanceToVoting(proposalID uint64) {
	// Advance past discussion period.
	prop, found := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.True(h.t, found)
	prop.Stage = string(zeronegovtypes.ResearchStageVoting)
	h.govKeeper.SetResearchSpendProposal(h.ctx, prop)
}

// ---------- Genesis helpers (duplicated from harness for package isolation) ----------

func genesisStateWithValSet(
	t *testing.T,
	app *zeroneapp.ZeroneApp,
	genState zeroneapp.GenesisState,
) zeroneapp.GenesisState {
	t.Helper()

	privVal := cmted25519.GenPrivKey()
	pubKey := privVal.PubKey()
	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	senderPrivKey := cmted25519.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), nil, 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100_000_000_000))),
	}

	codec := app.AppCodec()

	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), []authtypes.GenesisAccount{acc})
	genState[authtypes.ModuleName] = codec.MustMarshalJSON(authGenesis)

	validators := make([]sdkstakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]sdkstakingtypes.Delegation, 0, len(valSet.Validators))
	bondAmt := sdk.DefaultPowerReduction

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromCmtPubKeyInterface(val.PubKey)
		require.NoError(t, err)
		pkAny, err := codectypes.NewAnyWithValue(pk)
		require.NoError(t, err)
		v := sdkstakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(val.Address).String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            sdkstakingtypes.Bonded,
			Tokens:            bondAmt,
			DelegatorShares:   sdkmath.LegacyOneDec(),
			Description:       sdkstakingtypes.Description{},
			UnbondingHeight:   0,
			Commission:        sdkstakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
			MinSelfDelegation: sdkmath.ZeroInt(),
		}
		validators = append(validators, v)
		delegations = append(delegations, sdkstakingtypes.NewDelegation(
			acc.GetAddress().String(),
			sdk.ValAddress(val.Address).String(),
			sdkmath.LegacyOneDec(),
		))
	}

	stakingGenesis := sdkstakingtypes.NewGenesisState(sdkstakingtypes.DefaultParams(), validators, delegations)
	genState[sdkstakingtypes.ModuleName] = codec.MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	totalSupply = totalSupply.Add(balance.Coins...)
	for range delegations {
		totalSupply = totalSupply.Add(sdk.NewCoin(sdk.DefaultBondDenom, bondAmt))
	}

	balances := []banktypes.Balance{balance}
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(sdkstakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, bondAmt)},
	})

	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultGenesisState().Params,
		balances,
		totalSupply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{},
	)
	genState[banktypes.ModuleName] = codec.MustMarshalJSON(bankGenesis)

	return genState
}

// ---------- Vault Governance E2E Tests ----------

// TestVaultGovernance_HappyPath verifies the complete 2-of-2 flow:
// founder proposes → advance past discussion → both vote yes → executed → funds transferred.
func TestVaultGovernance_HappyPath(t *testing.T) {
	h := newVaultTestHarness(t)
	founder, aiVault := h.setupVoters()

	// Fund the research fund (via vesting rewards module).
	// For this test we wire a mock vesting keeper that succeeds.
	mock := &mockVestingKeeper{}
	h.govKeeper.SetVestingKeeper(mock)

	// Founder submits proposal.
	proposalID := h.submitProposal(founder)
	require.Equal(t, uint64(1), proposalID)

	// Verify proposal in discussion stage.
	prop, found := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.True(t, found)
	require.Equal(t, string(zeronegovtypes.ResearchStageDiscussion), prop.Stage)

	// Advance to voting stage.
	h.advanceToVoting(proposalID)

	// Founder votes yes.
	_, err := h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      founder,
		ProposalId: proposalID,
		Vote:       "yes",
		Reasoning:  "Aligns with research roadmap",
	})
	require.NoError(t, err)

	// Proposal still in voting (needs both votes).
	prop, _ = h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, string(zeronegovtypes.ResearchStageVoting), prop.Stage)

	// AI vault votes yes.
	_, err = h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      aiVault,
		ProposalId: proposalID,
		Vote:       "yes",
		Reasoning:  "Verified budget and recipient",
	})
	require.NoError(t, err)

	// Proposal should be executed.
	prop, _ = h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, string(zeronegovtypes.ResearchStageExecuted), prop.Stage)
	require.True(t, mock.disburseCalled, "DisburseFromResearchFund should have been called")

	// Verify audit trail.
	require.Equal(t, "yes", prop.Voter1Vote)
	require.Equal(t, "yes", prop.Voter2Vote)
	require.Equal(t, "Aligns with research roadmap", prop.Voter1Reason)
	require.Equal(t, "Verified budget and recipient", prop.Voter2Reason)
}

// TestVaultGovernance_FounderAlone verifies that the founder alone cannot spend funds.
// Only founder votes yes → voting expires → proposal expired.
func TestVaultGovernance_FounderAlone(t *testing.T) {
	h := newVaultTestHarness(t)
	founder, _ := h.setupVoters()

	mock := &mockVestingKeeper{}
	h.govKeeper.SetVestingKeeper(mock)

	proposalID := h.submitProposal(founder)
	h.advanceToVoting(proposalID)

	// Founder votes yes.
	_, err := h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      founder,
		ProposalId: proposalID,
		Vote:       "yes",
		Reasoning:  "I approve",
	})
	require.NoError(t, err)

	// AI vault does NOT vote. Advance past voting period.
	// Need to update the proposal's voting end block to something we can reach.
	prop, _ := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	prop.VotingEndsAt = uint64(h.currentHeight) + 3
	h.govKeeper.SetResearchSpendProposal(h.ctx, prop)

	h.advanceBlocks(5)

	// Run expiry processor.
	h.govKeeper.ProcessResearchSpendExpiry(h.ctx, uint64(h.currentHeight))

	prop, _ = h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, string(zeronegovtypes.ResearchStageExpired), prop.Stage)
	require.False(t, mock.disburseCalled, "funds should NOT have been disbursed")
}

// TestVaultGovernance_AIAlone verifies that the AI vault alone cannot spend funds.
// Only AI votes yes → voting expires → proposal expired.
func TestVaultGovernance_AIAlone(t *testing.T) {
	h := newVaultTestHarness(t)
	founder, aiVault := h.setupVoters()

	mock := &mockVestingKeeper{}
	h.govKeeper.SetVestingKeeper(mock)

	proposalID := h.submitProposal(founder)
	h.advanceToVoting(proposalID)

	// Only AI vault votes yes.
	_, err := h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      aiVault,
		ProposalId: proposalID,
		Vote:       "yes",
		Reasoning:  "Looks good to me",
	})
	require.NoError(t, err)

	// Founder does NOT vote. Advance past voting period.
	prop, _ := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	prop.VotingEndsAt = uint64(h.currentHeight) + 3
	h.govKeeper.SetResearchSpendProposal(h.ctx, prop)

	h.advanceBlocks(5)

	h.govKeeper.ProcessResearchSpendExpiry(h.ctx, uint64(h.currentHeight))

	prop, _ = h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, string(zeronegovtypes.ResearchStageExpired), prop.Stage)
	require.False(t, mock.disburseCalled, "funds should NOT have been disbursed")
}

// TestVaultGovernance_AIRejects verifies that a NO vote from either party
// immediately rejects the proposal.
func TestVaultGovernance_AIRejects(t *testing.T) {
	h := newVaultTestHarness(t)
	founder, aiVault := h.setupVoters()

	mock := &mockVestingKeeper{}
	h.govKeeper.SetVestingKeeper(mock)

	proposalID := h.submitProposal(founder)
	h.advanceToVoting(proposalID)

	// Founder votes yes.
	_, err := h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      founder,
		ProposalId: proposalID,
		Vote:       "yes",
		Reasoning:  "I approve",
	})
	require.NoError(t, err)

	// AI vault votes NO → immediate rejection.
	_, err = h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      aiVault,
		ProposalId: proposalID,
		Vote:       "no",
		Reasoning:  "Budget concerns",
	})
	require.NoError(t, err)

	prop, _ := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, string(zeronegovtypes.ResearchStageRejected), prop.Stage)
	require.False(t, mock.disburseCalled, "funds should NOT have been disbursed")
}

// TestVaultGovernance_Timeout verifies that proposals expire when no one votes.
func TestVaultGovernance_Timeout(t *testing.T) {
	h := newVaultTestHarness(t)
	founder, _ := h.setupVoters()

	mock := &mockVestingKeeper{}
	h.govKeeper.SetVestingKeeper(mock)

	proposalID := h.submitProposal(founder)
	h.advanceToVoting(proposalID)

	// No one votes. Set voting end to be reachable.
	prop, _ := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	prop.VotingEndsAt = uint64(h.currentHeight) + 3
	h.govKeeper.SetResearchSpendProposal(h.ctx, prop)

	h.advanceBlocks(5)

	h.govKeeper.ProcessResearchSpendExpiry(h.ctx, uint64(h.currentHeight))

	prop, _ = h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, string(zeronegovtypes.ResearchStageExpired), prop.Stage)
	require.False(t, mock.disburseCalled, "funds should NOT have been disbursed")
}

// TestVaultGovernance_DoubleVote verifies that the same voter cannot vote twice.
func TestVaultGovernance_DoubleVote(t *testing.T) {
	h := newVaultTestHarness(t)
	founder, _ := h.setupVoters()

	mock := &mockVestingKeeper{}
	h.govKeeper.SetVestingKeeper(mock)

	proposalID := h.submitProposal(founder)
	h.advanceToVoting(proposalID)

	// Founder votes yes.
	_, err := h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      founder,
		ProposalId: proposalID,
		Vote:       "yes",
		Reasoning:  "First vote",
	})
	require.NoError(t, err)

	// Founder tries to vote again → should fail.
	_, err = h.govKeeper.VoteResearchSpend(h.ctx, &zeronegovtypes.MsgVoteResearchSpend{
		Voter:      founder,
		ProposalId: proposalID,
		Vote:       "no",
		Reasoning:  "Changed my mind",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, zeronegovtypes.ErrResearchAlreadyVoted)

	// Original vote should be preserved.
	prop, _ := h.govKeeper.GetResearchSpendProposal(h.ctx, proposalID)
	require.Equal(t, "yes", prop.Voter1Vote)
	require.Equal(t, "First vote", prop.Voter1Reason)
}

// ---------- Mock Vesting Keeper ----------

type mockVestingKeeper struct {
	disburseCalled bool
	disburseErr    error
}

func (m *mockVestingKeeper) DisburseFromResearchFund(_ sdk.Context, _ sdk.AccAddress, _ sdk.Coins) error {
	m.disburseCalled = true
	return m.disburseErr
}

// Ensure unused imports are consumed.
var (
	_ = fmt.Sprintf
	_ bankkeeper.Keeper
	_ authkeeper.AccountKeeper
)
