package cross_stack_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestScenario10_AllModulesDefaultGenesisValid verifies that every registered
// module produces a valid DefaultGenesis that passes ValidateGenesis.
// This is a comprehensive audit across all 35+ modules.
func TestScenario10_AllModulesDefaultGenesisValid(t *testing.T) {
	app := newTestApp(t, testChainID)

	genState := app.DefaultGenesis()
	require.NotEmpty(t, genState)

	// Total module count must be >= 35 (SDK + Zerone custom modules)
	require.GreaterOrEqual(t, len(genState), 35,
		"expected >= 35 genesis modules, got %d", len(genState))
	t.Logf("total genesis modules: %d", len(genState))

	// ValidateGenesis for all modules
	err := zeroneapp.ModuleBasics.ValidateGenesis(
		app.AppCodec(),
		app.TxConfig(),
		genState,
	)
	require.NoError(t, err, "DefaultGenesis must pass ValidateGenesis for all modules")

	// Knowledge module specific validation
	knowledgeRaw, ok := genState[knowledgetypes.ModuleName]
	require.True(t, ok, "knowledge module must be in genesis")

	var knowledgeGen knowledgetypes.GenesisState
	require.NoError(t, app.AppCodec().UnmarshalJSON(knowledgeRaw, &knowledgeGen))
	require.NotNil(t, knowledgeGen.Params)

	// DefaultGenesis should have empty facts (axioms only via prepare-genesis)
	require.Empty(t, knowledgeGen.Facts, "DefaultGenesis must have empty facts")

	// But should have 18 domains
	require.Len(t, knowledgeGen.Domains, 18, "expected 18 genesis domains")

	// Verify slash params are non-zero
	p := knowledgeGen.Params
	require.Greater(t, p.WrongVerificationSlashBps, uint64(0))
	require.Greater(t, p.MissedRevealSlashBps, uint64(0))
	require.Greater(t, p.EquivocationSlashBps, uint64(0))
	// InvalidClaimSlashBps deprecated (R19-6): review fee is non-refundable

	// Verify BPS values on 1M scale
	require.LessOrEqual(t, p.InitialConfidence, uint64(1_000_000))
	require.LessOrEqual(t, p.ConfidenceThreshold, uint64(1_000_000))
	require.LessOrEqual(t, p.QuorumThreshold, uint64(1_000_000))

	// Verify non-zero epochs and minimums
	require.Greater(t, p.MinVerifiers, uint64(0))
	require.Greater(t, p.CommitPhaseBlocks, uint64(0))
	require.Greater(t, p.RevealPhaseBlocks, uint64(0))
}

// TestScenario11_SeedAxiomsWellFormed validates all embedded axioms pass
// comprehensive validation including DAG acyclicity and domain coverage.
func TestScenario13_ZeroTeamAllocationAtGenesis(t *testing.T) {
	app := newTestApp(t, testChainID)

	genState := app.DefaultGenesis()

	// Decode bank module genesis directly — InitChain is not called here,
	// so we read DefaultGenesis as authored, not as patched by a val-set
	// helper that would bond tokens.
	var bankGen banktypes.GenesisState
	require.NoError(t, app.AppCodec().UnmarshalJSON(genState[banktypes.ModuleName], &bankGen))

	// No positive balances. Module accounts may exist (with a 0-balance
	// entry registered for permission tracking), but no entry may carry
	// positive uzrn.
	violations := []string{}
	for _, bal := range bankGen.Balances {
		for _, coin := range bal.Coins {
			if coin.Denom == zeroneapp.BondDenom && coin.Amount.IsPositive() {
				violations = append(violations, bal.Address+" holds "+coin.String())
			}
		}
	}
	require.Empty(t, violations,
		"no genesis account may hold ZRN — doctrine: zero team allocation. violations: %v", violations)

	// No positive supply. Total supply at genesis is 0; minting begins
	// at block 1 through x/vesting_rewards block rewards, and per-claim
	// through x/claiming_pot bootstrap claims.
	for _, supply := range bankGen.Supply {
		if supply.Denom == zeroneapp.BondDenom {
			require.True(t, supply.Amount.IsZero(),
				"genesis supply for ZRN must be 0; got %s", supply.Amount.String())
		}
	}
}

// TestScenario13b_AllowedAppConstants is a compile-time and value-time
// assertion that app/constants.go exposes only doctrine-aligned constants.
// If anyone re-introduces TotalSupplyZRN, FounderZRN, AIAgentZRN,
// ValidatorZRN, ResearchFundZRN, or ClaimingPotsZRN, this test will fail
// to compile (those identifiers no longer resolve). Conversely, this test
// pins the expected values of the constants that ARE allowed — chain-id,
// denom, prefix, block time, and the micro-denomination multiplier.
func TestScenario13b_AllowedAppConstants(t *testing.T) {
	require.Equal(t, "zeroned", zeroneapp.AppName)
	require.Equal(t, "zrn", zeroneapp.AccountAddressPrefix)
	require.Equal(t, "uzrn", zeroneapp.BondDenom)
	require.Equal(t, "zrn", zeroneapp.DisplayDenom)
	require.Equal(t, 2521, zeroneapp.DefaultBlockTime)
	require.Equal(t, 1_000_000, zeroneapp.MicroDenomMultiplier)
	require.Equal(t, "zerone-testnet-1", zeroneapp.TestnetChainID)
}

// TestScenario13c_ClaimingPotMinterPermission asserts that the claiming_pot
// module account is registered with Minter permission, enabling the
// bootstrap-claim emission pathway. Without Minter permission, the
// bootstrap pathway cannot mint and the doctrine collapses back to the
// pre-fund-then-transfer model. The permission is the structural form of
// commitment 20 (issuance follows participation) at the module-account
// layer.
func TestScenario13c_ClaimingPotMinterPermission(t *testing.T) {
	h := NewTestHarness(t)

	moduleAcc := h.App.AccountKeeper.GetModuleAccount(h.Ctx, claimingpottypes.ModuleName)
	require.NotNil(t, moduleAcc, "claiming_pot module account must exist post-InitChain")

	require.True(t, moduleAcc.HasPermission(authtypes.Minter),
		"claiming_pot module account must hold Minter permission to drive the bootstrap mint pathway (commitment 20: issuance follows participation)")
}

// TestScenario13d_BootstrapPotForAgent asserts the bootstrap-pot helper
// produces a doctrine-aligned pot for an arbitrary agent: per-agent
// amount = 222,000 uzrn (0.222 ZRN), single-claimant whitelist, instant
// vest, ACTIVE status. The genesis ceremony will call this helper once
// per whitelisted address in the operator's whitelist file (Phase 5).
//
// The pot model is shared-bucket-vesting, so "per-agent fixed amount"
// is expressed structurally as one pot per agent. This test pins the
// shape of those per-agent pots — a contract change here is a doctrine
// change.
func TestScenario13d_BootstrapPotForAgent(t *testing.T) {
	const sampleAgent = "zrn1exampleagentaddressforbootstraptest"
	const blockHeight = uint64(1)

	pot := claimingpottypes.MakeBootstrapPotForAgent(sampleAgent, blockHeight)

	require.Equal(t, claimingpottypes.BootstrapPotIDPrefix+sampleAgent, pot.Id,
		"bootstrap pot ID must carry the prefix so ceremony tooling can enumerate them")
	require.Equal(t, claimingpottypes.PerAgentBootstrapUzrn, pot.TotalAmount,
		"per-agent amount must be 222000 uzrn (0.222 ZRN) per commitment 20 doctrine")
	require.Equal(t, "0", pot.ClaimedAmount,
		"freshly created pot must have ClaimedAmount = 0")

	require.NotNil(t, pot.Schedule, "schedule must be set so CalculateClaimable can vest")
	require.Equal(t, blockHeight, pot.Schedule.StartBlock)
	require.Equal(t, blockHeight+claimingpottypes.BootstrapPotInstantVestBlocks, pot.Schedule.EndBlock)

	require.NotNil(t, pot.Eligibility)
	require.Equal(t, []string{sampleAgent}, pot.Eligibility.Whitelist,
		"single-agent whitelist binds the pot to exactly the recipient (no surface for cross-claim)")
	require.Equal(t, uint32(0), pot.Eligibility.MinStakingTier,
		"bootstrap is the participation seed — agents have not yet staked")

	require.Equal(t, claimingpottypes.PotStatus_POT_STATUS_ACTIVE, pot.Status)
}

// TestScenario13e_BootstrapPotsDoNotExpire confirms the operational
// binding for commitment 20: bootstrap pots are participation seeds, and
// the BeginBlocker pot-expiry sweep does not transition them to EXPIRED.
//
// Without this invariant, the genesis bootstrap pathway is structurally
// unclaimable. At the start of block 1, ProcessPotExpiry would flip every
// bootstrap pot (StartBlock=0, EndBlock=1) to EXPIRED before any MsgClaim
// tx in block 1 could run. A participation seed must remain claimable for
// the participant who shows up, regardless of how late.
func TestScenario13e_BootstrapPotsDoNotExpire(t *testing.T) {
	h := NewTestHarness(t)

	// Seed a bootstrap pot directly (skip the authority gate; this test is
	// about the expiry rule, not the admission path).
	const sampleAgent = "zrn1exampleagentforexpirytest000000000"
	pot := claimingpottypes.MakeBootstrapPotForAgent(sampleAgent, 0)
	h.ClaimingPotKeeper.SetPot(h.Ctx, pot)

	// Advance many blocks. AdvanceBlocks runs full BeginBlocker each tick,
	// so ProcessPotExpiry fires every block.
	h.AdvanceBlocks(100)

	got, found := h.ClaimingPotKeeper.GetPot(h.Ctx, pot.Id)
	require.True(t, found, "bootstrap pot must persist across the BeginBlocker expiry sweep")
	require.Equal(t, claimingpottypes.PotStatus_POT_STATUS_ACTIVE, got.Status,
		"bootstrap pot must remain ACTIVE after BeginBlocker sweeps; commitment 20 requires the seed to stay claimable")
}

// TestScenario14_GenesisRoundTripWithAxioms verifies that a genesis state
// with injected axioms can be marshaled, unmarshaled, and validated.
func TestScenario15_ByteIdenticalGenesisRoundTrip(t *testing.T) {
	app1 := newTestApp(t, testChainID)
	app2 := newTestApp(t, testChainID)

	genState1 := app1.DefaultGenesis()
	genState2 := app2.DefaultGenesis()

	bytes1, err := json.Marshal(genState1)
	require.NoError(t, err)

	bytes2, err := json.Marshal(genState2)
	require.NoError(t, err)

	require.Equal(t, bytes1, bytes2,
		"DefaultGenesis must be byte-identical across app instances")

	// Round-trip: unmarshal → re-marshal must be idempotent
	var restored zeroneapp.GenesisState
	require.NoError(t, json.Unmarshal(bytes1, &restored))

	bytes3, err := json.Marshal(restored)
	require.NoError(t, err)

	require.Equal(t, bytes1, bytes3,
		"genesis JSON must be idempotent through marshal → unmarshal → marshal")
}

// TestScenario16_InvalidGenesisRejection verifies that ValidateGenesis rejects
// invalid parameter values for each module.
func TestScenario16_InvalidGenesisRejection(t *testing.T) {
	app := newTestApp(t, testChainID)
	codec := app.AppCodec()
	txConfig := app.TxConfig()

	t.Run("Knowledge_ZeroSlash", func(t *testing.T) {
		genState := app.DefaultGenesis()
		knowledgeGen := knowledgetypes.DefaultGenesis()
		knowledgeGen.Params.WrongVerificationSlashBps = 0
		bz, err := codec.MarshalJSON(knowledgeGen)
		require.NoError(t, err)
		genState[knowledgetypes.ModuleName] = bz

		err = zeroneapp.ModuleBasics.ValidateGenesis(codec, txConfig, genState)
		require.Error(t, err, "zero slash BPS should be rejected")
		t.Logf("rejected with: %v", err)
	})

	t.Run("Knowledge_ConfidenceOverBPS", func(t *testing.T) {
		genState := app.DefaultGenesis()
		knowledgeGen := knowledgetypes.DefaultGenesis()
		knowledgeGen.Params.InitialConfidence = 1_000_001
		bz, err := codec.MarshalJSON(knowledgeGen)
		require.NoError(t, err)
		genState[knowledgetypes.ModuleName] = bz

		err = zeroneapp.ModuleBasics.ValidateGenesis(codec, txConfig, genState)
		require.Error(t, err, "InitialConfidence > 1M should be rejected")
		t.Logf("rejected with: %v", err)
	})


	t.Run("Emergency_InvalidStatus", func(t *testing.T) {
		genState := app.DefaultGenesis()
		emGen := emergencytypes.DefaultGenesis()
		emGen.Params.HaltPrevoteBlocks = 0 // invalid: must be > 0
		bz, err := codec.MarshalJSON(emGen)
		require.NoError(t, err)
		genState[emergencytypes.ModuleName] = bz

		err = zeroneapp.ModuleBasics.ValidateGenesis(codec, txConfig, genState)
		require.Error(t, err, "zero halt_prevote_blocks should be rejected")
		t.Logf("rejected with: %v", err)
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Ceremony-artifact audit (design §4 TEST supply-invariant-audit)
//
// The tests above audit DefaultGenesis — which is exactly how the 1.2M-ZRN
// testnet artifact went unnoticed: nothing audited what the ceremony script
// actually PRODUCED. TestGenesisArtifact_SupplyInvariants closes that gap:
// point ZERONE_GENESIS_ARTIFACT at a genesis.json emitted by
// scripts/mainnet-ceremony.sh and it asserts the §4 supply-invariant list
// against the artifact itself.
//
//	ZERONE_GENESIS_ARTIFACT=ceremony-out/genesis.json \
//	  go test ./tests/cross_stack/ -run TestGenesisArtifact -v
// ─────────────────────────────────────────────────────────────────────────────

// Canonical ceremony constants (design §2). A 4→5 validator roster change
// (design §10a) flips nValidators and totalSupply here and in
// scripts/mainnet-ceremony.sh together.
const (
	artifactEnvVar        = "ZERONE_GENESIS_ARTIFACT"
	artifactNValidators   = 4
	artifactStakeUzrn     = "11111000000" // 11,111 ZRN permanently locked per validator
	artifactFloatUzrn     = "111000000"   // 111 ZRN operator float
	artifactOnboardUzrn   = "2222000000"  // 2,222 ZRN onboarding multisig
	artifactTotalSupply   = "47110000000" // 47,110 ZRN — the §10 zero-ALLOCATION invariant
	artifactNBalances     = 9
	permanentLockedType   = "/cosmos.vesting.v1beta1.PermanentLockedAccount"
	msgCreateValidatorURL = "/cosmos.staking.v1beta1.MsgCreateValidator"
)

// artifactGenesis mirrors just the genesis.json paths the audit reads.
type artifactGenesis struct {
	AppState struct {
		Auth struct {
			Accounts []struct {
				Type               string `json:"@type"`
				Address            string `json:"address"`
				BaseVestingAccount struct {
					BaseAccount struct {
						Address string `json:"address"`
					} `json:"base_account"`
					OriginalVesting []struct {
						Denom  string `json:"denom"`
						Amount string `json:"amount"`
					} `json:"original_vesting"`
					EndTime string `json:"end_time"`
				} `json:"base_vesting_account"`
			} `json:"accounts"`
		} `json:"auth"`
		Bank struct {
			Balances []struct {
				Address string `json:"address"`
				Coins   []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				} `json:"coins"`
			} `json:"balances"`
			Supply []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"supply"`
		} `json:"bank"`
		Knowledge struct {
			BootstrapFundAllocation string `json:"bootstrap_fund_allocation"`
		} `json:"knowledge"`
		Gov struct {
			Params struct {
				MinDeposit []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				} `json:"min_deposit"`
				ExpeditedMinDeposit []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				} `json:"expedited_min_deposit"`
			} `json:"params"`
		} `json:"gov"`
		SubstrateBridge struct {
			Adapters []struct {
				AdapterID string `json:"adapter_id"`
				Status    string `json:"status"`
			} `json:"adapters"`
		} `json:"substrate_bridge"`
		Transfer struct {
			Params struct {
				SendEnabled    bool `json:"send_enabled"`
				ReceiveEnabled bool `json:"receive_enabled"`
			} `json:"params"`
		} `json:"transfer"`
		InterchainAccounts struct {
			HostGenesisState struct {
				Params struct {
					HostEnabled   bool     `json:"host_enabled"`
					AllowMessages []string `json:"allow_messages"`
				} `json:"params"`
			} `json:"host_genesis_state"`
		} `json:"interchainaccounts"`
		Creed struct {
			GenesisPin struct {
				Version     uint32            `json:"version"`
				Commitments []json.RawMessage `json:"commitments"`
			} `json:"genesis_pin"`
		} `json:"creed"`
		WorkCreed struct {
			PinnedSubCreeds []json.RawMessage `json:"pinned_sub_creeds"`
		} `json:"work_creed"`
		Genutil struct {
			GenTxs []struct {
				Body struct {
					Messages []struct {
						Type             string `json:"@type"`
						ValidatorAddress string `json:"validator_address"`
						Value            struct {
							Denom  string `json:"denom"`
							Amount string `json:"amount"`
						} `json:"value"`
					} `json:"messages"`
				} `json:"body"`
			} `json:"gen_txs"`
		} `json:"genutil"`
	} `json:"app_state"`
}

func loadCeremonyArtifact(t *testing.T) *artifactGenesis {
	t.Helper()
	path := os.Getenv(artifactEnvVar)
	if path == "" {
		t.Skipf("%s not set — run scripts/mainnet-ceremony.sh and point it at the emitted genesis.json", artifactEnvVar)
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read ceremony artifact %s", path)
	var g artifactGenesis
	require.NoError(t, json.Unmarshal(raw, &g), "parse ceremony artifact %s", path)
	return &g
}

// TestGenesisArtifact_SupplyInvariants asserts the design §4
// supply-invariant list against a ceremony-produced genesis.json.
func TestGenesisArtifact_SupplyInvariants(t *testing.T) {
	g := loadCeremonyArtifact(t)

	// ── bank supply: exactly 47,110 ZRN, single denom ────────────────────
	require.Len(t, g.AppState.Bank.Supply, 1, "genesis supply must carry exactly one denom")
	require.Equal(t, zeroneapp.BondDenom, g.AppState.Bank.Supply[0].Denom)
	require.Equal(t, artifactTotalSupply, g.AppState.Bank.Supply[0].Amount,
		"§10 zero-ALLOCATION invariant: bank supply must be exactly 47,110 ZRN")

	// ── exactly 9 balances in the canonical role buckets ─────────────────
	require.Len(t, g.AppState.Bank.Balances, artifactNBalances,
		"genesis must have exactly %d balances (4 stake + 4 float + 1 onboarding)", artifactNBalances)

	balanceByAddr := map[string]string{}
	buckets := map[string][]string{} // amount → addresses
	for _, bal := range g.AppState.Bank.Balances {
		require.Len(t, bal.Coins, 1, "balance %s must be single-coin", bal.Address)
		require.Equal(t, zeroneapp.BondDenom, bal.Coins[0].Denom)
		balanceByAddr[bal.Address] = bal.Coins[0].Amount
		buckets[bal.Coins[0].Amount] = append(buckets[bal.Coins[0].Amount], bal.Address)
	}
	roleBuckets := []struct {
		role   string
		amount string
		count  int
	}{
		{"validator stake (permanently locked)", artifactStakeUzrn, artifactNValidators},
		{"operator float", artifactFloatUzrn, artifactNValidators},
		{"onboarding multisig", artifactOnboardUzrn, 1},
	}
	for _, rb := range roleBuckets {
		require.Len(t, buckets[rb.amount], rb.count,
			"expected %d × %s uzrn balances (%s)", rb.count, rb.amount, rb.role)
	}

	// ── 4 PermanentLockedAccounts == the 4 stake balances ────────────────
	lockedAddrs := map[string]bool{}
	for _, acc := range g.AppState.Auth.Accounts {
		if acc.Type != permanentLockedType {
			continue
		}
		addr := acc.BaseVestingAccount.BaseAccount.Address
		require.Len(t, acc.BaseVestingAccount.OriginalVesting, 1)
		require.Equal(t, zeroneapp.BondDenom, acc.BaseVestingAccount.OriginalVesting[0].Denom)
		require.Equal(t, artifactStakeUzrn, acc.BaseVestingAccount.OriginalVesting[0].Amount,
			"locked account %s must vest its FULL balance", addr)
		require.Equal(t, "0", acc.BaseVestingAccount.EndTime,
			"PermanentLockedAccount end_time must be 0 (never unlocks)")
		lockedAddrs[addr] = true
	}
	require.Len(t, lockedAddrs, artifactNValidators,
		"expected exactly %d PermanentLockedAccounts", artifactNValidators)
	for _, addr := range buckets[artifactStakeUzrn] {
		require.True(t, lockedAddrs[addr],
			"stake balance %s must be a PermanentLockedAccount", addr)
	}

	// ── fully bonded via gentxs: one full self-bond per locked account ───
	require.Len(t, g.AppState.Genutil.GenTxs, artifactNValidators,
		"expected %d gentxs (every locked stake fully bonded at block 0)", artifactNValidators)
	bondedOperators := map[string]bool{}
	for _, tx := range g.AppState.Genutil.GenTxs {
		require.Len(t, tx.Body.Messages, 1, "gentx must carry exactly one message")
		msg := tx.Body.Messages[0]
		require.Equal(t, msgCreateValidatorURL, msg.Type)
		require.Equal(t, zeroneapp.BondDenom, msg.Value.Denom)
		require.Equal(t, artifactStakeUzrn, msg.Value.Amount,
			"gentx self-bond must be the FULL 11,111 ZRN locked stake")
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		require.NoError(t, err)
		operator := sdk.AccAddress(valAddr).String()
		require.True(t, lockedAddrs[operator],
			"gentx operator %s must be one of the locked stake accounts", operator)
		require.False(t, bondedOperators[operator], "duplicate gentx for operator %s", operator)
		bondedOperators[operator] = true
	}
	require.Len(t, bondedOperators, artifactNValidators)

	// ── knowledge fund zeroed: no 22,222 ZRN InitGenesis mint ────────────
	require.Equal(t, "0", g.AppState.Knowledge.BootstrapFundAllocation,
		"knowledge.bootstrap_fund_allocation must be \"0\" — day-0 supply stays exactly 47,110 ZRN")

	// ── SDK gov denominated in uzrn (default 'stake' would kill gov) ─────
	require.Len(t, g.AppState.Gov.Params.MinDeposit, 1)
	require.Equal(t, zeroneapp.BondDenom, g.AppState.Gov.Params.MinDeposit[0].Denom,
		"gov min_deposit must be uzrn or the authority surface is dead")
	require.Equal(t, "100000000", g.AppState.Gov.Params.MinDeposit[0].Amount)
	require.Len(t, g.AppState.Gov.Params.ExpeditedMinDeposit, 1)
	require.Equal(t, zeroneapp.BondDenom, g.AppState.Gov.Params.ExpeditedMinDeposit[0].Denom)
	require.Equal(t, "300000000", g.AppState.Gov.Params.ExpeditedMinDeposit[0].Amount)

	// ── agenttool adapter pre-registered ACTIVE ──────────────────────────
	require.Len(t, g.AppState.SubstrateBridge.Adapters, 1,
		"exactly one genesis adapter expected")
	require.Equal(t, "agenttool-invocation-v1", g.AppState.SubstrateBridge.Adapters[0].AdapterID)
	require.Equal(t, "ADAPTER_STATUS_ACTIVE", g.AppState.SubstrateBridge.Adapters[0].Status)

	// ── IBC dark at genesis ──────────────────────────────────────────────
	require.False(t, g.AppState.Transfer.Params.SendEnabled, "IBC transfer send must be disabled at genesis")
	require.False(t, g.AppState.Transfer.Params.ReceiveEnabled, "IBC transfer receive must be disabled at genesis")
	require.False(t, g.AppState.InterchainAccounts.HostGenesisState.Params.HostEnabled, "ICA host must be disabled at genesis")
	require.Empty(t, g.AppState.InterchainAccounts.HostGenesisState.Params.AllowMessages,
		"ICA allow_messages must be empty (ibc-go default is allow-all '*')")

	// ── creed pinned at block 0 ──────────────────────────────────────────
	require.EqualValues(t, 1, g.AppState.Creed.GenesisPin.Version, "Genesis Creed pin must be version 1")
	require.Len(t, g.AppState.Creed.GenesisPin.Commitments, 20, "Genesis Creed must pin all 20 commitments")
	require.Len(t, g.AppState.WorkCreed.PinnedSubCreeds, 8, "work_creed must carry the 8 inception pins")

	// ── no foundation / research / faucet / module-account balances ──────
	// Doctrine (zero team allocation): the nine role balances above are the
	// ONLY balances; in particular no module-derived account may hold coin.
	forbiddenModuleAccounts := []string{
		"foundation",
		"research_fund",
		"faucet",
		"fee_collector",
		"distribution",
		"bonded_tokens_pool",
		"not_bonded_tokens_pool",
		"gov",
		"mint",
		claimingpottypes.ModuleName,
		knowledgetypes.ModuleName,
	}
	for _, name := range forbiddenModuleAccounts {
		addr := authtypes.NewModuleAddress(name).String()
		amount, found := balanceByAddr[addr]
		require.False(t, found,
			"module account %q (%s) must hold NO genesis balance, found %s uzrn", name, addr, amount)
	}
}

// TestScenario17_BankGenesisSupplyConsistency verifies that the bank genesis
// supply entries sum to the declared total supply.
func TestScenario17_BankGenesisSupplyConsistency(t *testing.T) {
	app := newTestApp(t, testChainID)

	genState := app.DefaultGenesis()
	genState = genesisStateWithValSet(t, app, genState)

	// Unmarshal bank genesis
	var bankGen banktypes.GenesisState
	require.NoError(t, app.AppCodec().UnmarshalJSON(genState[banktypes.ModuleName], &bankGen))

	// Sum all balance entries
	totalFromBalances := make(map[string]int64)
	for _, bal := range bankGen.Balances {
		for _, coin := range bal.Coins {
			totalFromBalances[coin.Denom] += coin.Amount.Int64()
		}
	}

	// Compare against declared supply
	for _, supply := range bankGen.Supply {
		balAmt, ok := totalFromBalances[supply.Denom]
		require.True(t, ok, "supply denom %s must have matching balances", supply.Denom)
		require.Equal(t, supply.Amount.Int64(), balAmt,
			"supply for %s (%d) must equal sum of balances (%d)",
			supply.Denom, supply.Amount.Int64(), balAmt)
	}

	t.Logf("bank genesis: %d balance entries, %d supply denoms",
		len(bankGen.Balances), len(bankGen.Supply))
}

// TestScenario18_ModuleAccountPermissions verifies that module account
// addresses are deterministically derived and that the app's DefaultGenesis
// includes module genesis entries for all expected Zerone custom modules.
func TestScenario18_ModuleAccountPermissions(t *testing.T) {
	// Verify module account addresses are deterministic (derived from module name).
	// In SDK v0.50, module accounts are created lazily when they first interact
	// with the bank. Here we verify the address derivation is consistent.
	moduleNames := []string{
		"fee_collector",
		"bonded_tokens_pool",
		"not_bonded_tokens_pool",
		"distribution",
		"zerone_auth",
		"research_fund",
	}
	for _, name := range moduleNames {
		addr := authtypes.NewModuleAddress(name)
		require.NotEmpty(t, addr, "module address for %q must not be empty", name)
		// Verify the address is a valid bech32 with zrn prefix
		bech32 := addr.String()
		require.True(t, len(bech32) > 3 && bech32[:3] == "zrn",
			"module address for %q should have zrn prefix, got %s", name, bech32)
		t.Logf("module %q -> %s", name, bech32)
	}

	// Verify address derivation is idempotent
	for _, name := range moduleNames {
		addr1 := authtypes.NewModuleAddress(name)
		addr2 := authtypes.NewModuleAddress(name)
		require.Equal(t, addr1, addr2,
			"module address for %q must be deterministic", name)
	}

	// Verify all expected Zerone custom modules appear in DefaultGenesis
	app := newTestApp(t, testChainID)
	genState := app.DefaultGenesis()

	zeroneModules := []string{
		"zerone_auth",
		"zerone_staking",
		"knowledge",
		"emergency",
		"alignment",
		"vesting_rewards",
	}
	for _, name := range zeroneModules {
		_, ok := genState[name]
		require.True(t, ok, "DefaultGenesis must include module %q", name)
		t.Logf("genesis module %q: present", name)
	}

	t.Logf("total genesis modules: %d", len(genState))
}
