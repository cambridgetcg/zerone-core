package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"

	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit (the reviewer-entry design draft, pull request #190,
// step B) through the application's actual SDK bank, account and zerone_auth
// keepers and the knowledge message handlers. Reviewers here are synthetic
// accounts funded by test issuance; they show how the code behaves, not that
// any outside reviewer has come in. These tests do not run the ante or count
// gas.

const seatTrialChainID = "seat-deposit-bank-test"

// seatTrial is one review round admitted through a real handler, on native
// genesis with the seat-deposit marker on or off.
type seatTrial struct {
	app   *ZeroneApp
	db    dbm.DB
	ctx   sdk.Context
	claim *knowledgetypes.Claim
	round *knowledgetypes.VerificationRound
	payer sdk.AccAddress
	// seats lists each reviewer that committed, in commit order, with the
	// vote it committed to.
	seats []seatTrialSeat
	// admitted lists the accounts admit wrote to zerone_auth.
	admitted []sdk.AccAddress
}

type seatTrialSeat struct {
	address sdk.AccAddress
	vote    string
	salt    []byte
}

var seatTrialAttestation = &knowledgetypes.ReviewAttestation{Reason: "Synthetic operator-owned review of a stated test fixture.", Scope: "No independent scientific assurance is claimed."}

// newSeatTrial admits a claim through route ("ordinary", "conjecture",
// "contradiction", "explicit" or "provisional") on a chain whose seat-deposit
// marker is seat and whose min_reviewer_balance is figure (written straight to
// the params store, as governance would). The windows are two blocks each.
func newSeatTrial(t *testing.T, route string, seat bool, figure string) *seatTrial {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(100).WithChainID(seatTrialChainID).WithHeaderInfo(header.Info{Height: 100, ChainID: seatTrialChainID})
	if seat {
		require.NoError(t, app.KnowledgeKeeper.EnableSeatDeposit(ctx))
	}
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.MinReviewFee, params.MinChallengeStake = "101", "101"
	params.ChallengeConfidenceScalingBps = 0
	params.MinVerifiers, params.MinHeadcountAgreement, params.ConfidenceThreshold = 3, 2, 600000
	params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 2
	params.VerificationReward = "0"
	params.MinReviewerBalance = figure
	require.NoError(t, app.KnowledgeKeeper.SetParams(ctx, params))
	tr := &seatTrial{app: app, db: db, ctx: ctx, payer: settlementAddress(150)}
	tr.fund(t, tr.payer, 10_000)
	target := &knowledgetypes.Fact{Id: "seat-test-target", Domain: "physics", Category: "empirical", Content: "A scoped synthetic target observation.", Submitter: settlementAddress(151).String(), Status: knowledgetypes.FactStatus_FACT_STATUS_ACTIVE}
	if route == "provisional" {
		target.Status = knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL
	}
	if route != "ordinary" && route != "conjecture" {
		require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, target))
	}
	server := knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper)
	var claimID, roundID string
	switch route {
	case "ordinary":
		response, err := server.SubmitClaim(ctx, &knowledgetypes.MsgSubmitClaim{Submitter: tr.payer.String(), FactContent: "This synthetic local computation has a retained review fee.", Domain: "physics", Category: "empirical", Stake: "101", ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION})
		require.NoError(t, err)
		claimID = response.ClaimId
	case "conjecture":
		response, err := server.PostConjecture(ctx, &knowledgetypes.MsgPostConjecture{Proposer: tr.payer.String(), Statement: "A synthetic model predicts a repeated observation outside its sampled range.", FalsificationPredicate: "One controlled observation outside the range violates that prediction.", Domain: "physics", Category: "empirical", Stake: "101", ReasoningTrace: "Synthetic well-posedness fixture; acceptance does not establish truth."})
		require.NoError(t, err)
		claimID = response.ClaimId
	case "contradiction":
		response, err := server.SubmitContradiction(ctx, &knowledgetypes.MsgSubmitContradiction{Submitter: tr.payer.String(), FactId: target.Id, CounterClaim: "This synthetic counter-observation contradicts the target.", Category: "empirical", Stake: "101", Reason: "A scoped synthetic counterexample."})
		require.NoError(t, err)
		claimID = response.CounterFactId
	case "explicit":
		response, err := server.ChallengeFact(ctx, &knowledgetypes.MsgChallengeFact{Challenger: tr.payer.String(), FactId: target.Id, Stake: "101", Reason: "A scoped synthetic refutation."})
		require.NoError(t, err)
		roundID = response.RoundId
	case "provisional":
		response, err := server.ChallengeProvisionalFact(ctx, &knowledgetypes.MsgChallengeProvisionalFact{Challenger: tr.payer.String(), FactId: target.Id, Stake: "101", Reason: "A scoped synthetic provisional refutation.", CounterClaim: "The synthetic predicted observation was absent."})
		require.NoError(t, err)
		roundID = response.ChallengeId
	default:
		t.Fatalf("unknown seat trial route %s", route)
	}
	if roundID != "" {
		round, found := app.KnowledgeKeeper.GetVerificationRound(ctx, roundID)
		require.True(t, found)
		claimID = round.ClaimId
	}
	claim, found := app.KnowledgeKeeper.GetClaim(ctx, claimID)
	require.True(t, found)
	tr.claim = claim
	tr.round, found = app.KnowledgeKeeper.GetVerificationRound(ctx, claim.VerificationRoundId)
	require.True(t, found)
	require.Equal(t, knowledgetypes.ReviewPolicyNeutral, tr.round.ReviewPolicyVersion)
	require.Equal(t, knowledgetypes.CommitmentSchemeReviewV2, tr.round.CommitmentScheme)
	return tr
}

func (tr *seatTrial) fund(t *testing.T, address sdk.AccAddress, amount int64) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
	require.NoError(t, tr.app.BankKeeper.MintCoins(tr.ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, tr.app.BankKeeper.SendCoinsFromModuleToAccount(tr.ctx, knowledgetypes.BootstrapFundModuleName, address, coins))
}

func (tr *seatTrial) balance(address sdk.AccAddress) int64 {
	return tr.app.BankKeeper.GetBalance(tr.ctx, address, "uzrn").Amount.Int64()
}

func (tr *seatTrial) custody() int64 {
	return tr.balance(authtypes.NewModuleAddress(knowledgetypes.SeatDepositModuleName))
}

var seatTrialRoutes = []string{"ordinary", "conjecture", "contradiction", "explicit", "provisional"}

// D1: every admission route records the round's seat terms while the marker
// is on, copied from min_reviewer_balance as it stood at creation; an empty
// parameter locks the built-in 100 ZRN. With the marker off no route records
// terms.
func TestSeatDepositTermsAtEveryCreationSite(t *testing.T) {
	for _, route := range seatTrialRoutes {
		for _, tc := range []struct {
			name   string
			seat   bool
			figure string
			want   string
		}{
			{"on/raised-figure", true, "150000000", "150000000"},
			{"on/empty-locks-100-ZRN", true, "", "100000000"},
			{"off", false, "", ""},
		} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				tr := newSeatTrial(t, route, tc.seat, tc.figure)
				if tc.want == "" {
					require.Nil(t, tr.round.SeatTerms)
					return
				}
				require.NotNil(t, tr.round.SeatTerms)
				require.Equal(t, tc.want, tr.round.SeatTerms.CountedSeatDeposit)
				// A later parameter change does not alter the open round.
				params, err := tr.app.KnowledgeKeeper.GetParams(tr.ctx)
				require.NoError(t, err)
				params.MinReviewerBalance = "999000000"
				require.NoError(t, tr.app.KnowledgeKeeper.SetParams(tr.ctx, params))
				round, found := tr.app.KnowledgeKeeper.GetVerificationRound(tr.ctx, tr.round.Id)
				require.True(t, found)
				require.Equal(t, tc.want, round.SeatTerms.CountedSeatDeposit)
				round.SeatTerms.CountedSeatDeposit = "999000000"
				require.ErrorContains(t, tr.app.KnowledgeKeeper.SetVerificationRound(tr.ctx, round), "seat terms are immutable")
			})
		}
	}
}

// D1: the marker on with its parameter predecessor off fails closed at round
// creation instead of locking a figure governance cannot set.
func TestSeatDepositTermsRequireTheParameterMarker(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	ctx = ctx.WithBlockHeight(100).WithChainID(seatTrialChainID).WithHeaderInfo(header.Info{Height: 100, ChainID: seatTrialChainID})
	require.NoError(t, app.KnowledgeKeeper.EnableSeatDeposit(ctx))
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete(knowledgetypes.ReviewerBalanceParamEnabledKey)
	payer := settlementAddress(152)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, payer, coins))
	params, err := app.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	_, err = knowledgekeeper.NewMsgServerImpl(app.KnowledgeKeeper).SubmitClaim(ctx, &knowledgetypes.MsgSubmitClaim{Submitter: payer.String(), FactContent: "This synthetic local computation has a retained review fee.", Domain: "physics", Category: "empirical", Stake: params.MinReviewFee, ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION})
	require.ErrorContains(t, err, "seat deposits require the reviewer-balance-param activation")
}

// admit registers address with zerone_auth as an agent with flags.
func (tr *seatTrial) admit(t *testing.T, address sdk.AccAddress, flags *zeroneauthtypes.AccountFlags) {
	t.Helper()
	tr.app.ZeroneAuthKeeper.SetAccount(tr.ctx, &zeroneauthtypes.Account{Address: address.String(), AccountType: "agent", Flags: flags})
	tr.admitted = append(tr.admitted, address)
}

// reviewer is a registered, funded account that may review.
func (tr *seatTrial) reviewer(t *testing.T, n byte, balance int64) sdk.AccAddress {
	t.Helper()
	address := settlementAddress(n)
	if balance > 0 {
		tr.fund(t, address, balance)
	}
	tr.admit(t, address, &zeroneauthtypes.AccountFlags{CanSubmitClaims: true})
	return address
}

func (tr *seatTrial) server() knowledgetypes.MsgServer {
	return knowledgekeeper.NewMsgServerImpl(tr.app.KnowledgeKeeper)
}

// commit submits address's commitment to vote in the trial's round, as a
// signed MsgSubmitCommitment would after the ante. A seat that is stored is
// remembered for reveal.
func (tr *seatTrial) commit(t *testing.T, address sdk.AccAddress, vote string) error {
	t.Helper()
	salt := bytes.Repeat([]byte{byte(len(tr.seats) + 1)}, 16)
	hash, err := knowledgetypes.ComputeReviewCommitmentV2(tr.round.CommitmentChainId, tr.round.Id, address.String(), vote, 800000, salt, seatTrialAttestation)
	require.NoError(t, err)
	_, err = tr.server().SubmitCommitment(tr.ctx, &knowledgetypes.MsgSubmitCommitment{Verifier: address.String(), RoundId: tr.round.Id, CommitHash: hash})
	if err == nil {
		tr.seats = append(tr.seats, seatTrialSeat{address: address, vote: vote, salt: salt})
	}
	return err
}

func (tr *seatTrial) stored(t *testing.T) *knowledgetypes.VerificationRound {
	t.Helper()
	round, found := tr.app.KnowledgeKeeper.GetVerificationRound(tr.ctx, tr.round.Id)
	require.True(t, found)
	return round
}

func (tr *seatTrial) height(h int64) {
	tr.ctx = tr.ctx.WithBlockHeight(h).WithHeaderInfo(header.Info{Height: h, ChainID: tr.ctx.ChainID()})
}

// requireRefusedCommit checks that a commitment is refused with want and that
// it moved nothing and stored nothing.
func (tr *seatTrial) requireRefusedCommit(t *testing.T, address sdk.AccAddress, want string) {
	t.Helper()
	before, custody := tr.balance(address), tr.custody()
	seats := len(tr.stored(t).Commits)
	require.ErrorContains(t, tr.commit(t, address, "accept"), want)
	require.Equal(t, before, tr.balance(address))
	require.Equal(t, custody, tr.custody())
	require.Len(t, tr.stored(t).Commits, seats)
}

func countEvents(ctx sdk.Context, kind string) int {
	n := 0
	for _, event := range ctx.EventManager().Events() {
		if event.Type == kind {
			n++
		}
	}
	return n
}

// D2: a commitment in a round with seat terms moves the round's deposit from
// the signer into the seat-deposit account, which the first deposit creates
// as a module account, and records it with the seat.
func TestSeatDepositLocksIntoModuleAccount(t *testing.T) {
	tr := newSeatTrial(t, "ordinary", true, "")
	const deposit = 100_000_000
	tr.ctx = tr.ctx.WithEventManager(sdk.NewEventManager())
	for i := byte(0); i < 3; i++ {
		reviewer := tr.reviewer(t, 160+i, deposit+7)
		require.NoError(t, tr.commit(t, reviewer, "accept"))
		require.Equal(t, int64(7), tr.balance(reviewer), "the deposit left the signer")
	}
	require.Equal(t, int64(3*deposit), tr.custody())
	round := tr.stored(t)
	require.Len(t, round.Commits, 3)
	for _, commit := range round.Commits {
		require.Equal(t, "100000000", commit.SeatDeposit)
	}
	account := tr.app.AccountKeeper.GetAccount(tr.ctx, authtypes.NewModuleAddress(knowledgetypes.SeatDepositModuleName))
	module, ok := account.(sdk.ModuleAccountI)
	require.True(t, ok)
	require.Equal(t, knowledgetypes.SeatDepositModuleName, module.GetName())
	require.Empty(t, module.GetPermissions())
	require.Equal(t, 3, countEvents(tr.ctx, "zerone.knowledge.seat_deposit_locked"))
	require.Equal(t, 3, countEvents(tr.ctx, "zerone.knowledge.submit_commitment"))
}

// D2: a signer that cannot lock the deposit is refused, and nothing moves or
// is stored. The transfer needs spendable coins: a delayed-vesting account
// whose whole balance covers the deposit, which the balance check it replaces
// counted, is refused while its coins are still vesting.
func TestSeatDepositSignerThatCannotLockIsRefused(t *testing.T) {
	tr := newSeatTrial(t, "ordinary", true, "")
	poor := tr.reviewer(t, 170, 99_999_999)
	tr.requireRefusedCommit(t, poor, "seat deposit required")

	tr.ctx = tr.ctx.WithBlockTime(time.Unix(1_800_000_000, 0))
	vesting := settlementAddress(171)
	tr.fund(t, vesting, 100_000_000)
	base := tr.app.AccountKeeper.GetAccount(tr.ctx, vesting)
	baseAccount, ok := base.(*authtypes.BaseAccount)
	require.True(t, ok)
	delayed, err := vestingtypes.NewDelayedVestingAccount(baseAccount, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 60_000_000)), tr.ctx.BlockTime().Unix()+86_400)
	require.NoError(t, err)
	tr.app.AccountKeeper.SetAccount(tr.ctx, delayed)
	tr.admit(t, vesting, &zeroneauthtypes.AccountFlags{CanSubmitClaims: true})
	require.Equal(t, int64(100_000_000), tr.balance(vesting), "the whole balance covers the deposit")
	tr.requireRefusedCommit(t, vesting, "seat deposit required")
	require.Zero(t, tr.custody())
}

// D3: the handler refuses a signer that zerone_auth does not admit to review,
// whatever its balance: unregistered, without CanSubmitClaims, or frozen.
func TestSeatDepositHandlerRefusesUnadmittedSigner(t *testing.T) {
	tr := newSeatTrial(t, "ordinary", true, "")
	unregistered := settlementAddress(172)
	tr.fund(t, unregistered, 200_000_000)
	tr.requireRefusedCommit(t, unregistered, "not registered with zerone_auth")
	for i, flags := range []*zeroneauthtypes.AccountFlags{nil, {CanSubmitClaims: false, CanChallenge: true}, {CanSubmitClaims: true, Frozen: true}} {
		address := settlementAddress(173 + byte(i))
		tr.fund(t, address, 200_000_000)
		tr.admit(t, address, flags)
		want := "cannot review"
		if flags != nil && flags.Frozen {
			want = "is frozen"
		}
		tr.requireRefusedCommit(t, address, want)
	}
	require.Zero(t, tr.custody())
}

// D3: no route but a signed, deposited commitment stores a seat in a round
// with seat terms. The keeper's StoreCommitmentInRound refuses, and so does
// the vote-extension writer, which names an account that did not sign; it
// logs the refusal and the block goes on.
func TestSeatDepositNoRouteStoresSeatWithoutDeposit(t *testing.T) {
	tr := newSeatTrial(t, "ordinary", true, "")
	reviewer := tr.reviewer(t, 176, 200_000_000)
	hash := bytes.Repeat([]byte{9}, 32)
	entry := &knowledgetypes.CommitEntry{Verifier: reviewer.String(), CommitHash: hash, CommittedAtBlock: 100}
	require.ErrorIs(t, tr.app.KnowledgeKeeper.StoreCommitmentInRound(tr.ctx, tr.round.Id, entry), knowledgetypes.ErrSeatDepositRequired)
	entry.SeatDeposit = "100000000"
	require.ErrorIs(t, tr.app.KnowledgeKeeper.StoreCommitmentInRound(tr.ctx, tr.round.Id, entry), knowledgetypes.ErrSeatDepositRequired, "a claimed deposit is not a locked one")
	var entries []recordedLog
	require.False(t, tr.app.storeInjectedCommitment(tr.ctx, recordingLogger{&entries}, tr.round.Id, &knowledgetypes.CommitEntry{Verifier: reviewer.String(), CommitHash: hash, CommittedAtBlock: 100}))
	require.Len(t, entries, 1)
	require.Equal(t, "skipped commitment from vote extension", entries[0].msg)
	require.Empty(t, tr.stored(t).Commits)
	require.Equal(t, int64(200_000_000), tr.balance(reviewer))
	require.Zero(t, tr.custody())
}

// A plain account at the seat-deposit address, which a send could create
// before this binary blocked the address, refuses every deposit with an error
// instead of the bank's panic, and nothing moves.
func TestSeatDepositRefusesAnAccountSquattingTheModuleAddress(t *testing.T) {
	tr := newSeatTrial(t, "ordinary", true, "")
	address := authtypes.NewModuleAddress(knowledgetypes.SeatDepositModuleName)
	tr.app.AccountKeeper.SetAccount(tr.ctx, tr.app.AccountKeeper.NewAccountWithAddress(tr.ctx, address))
	reviewer := tr.reviewer(t, 177, 200_000_000)
	tr.requireRefusedCommit(t, reviewer, "not its module account")
}

// The design's section 1, item 2, as a named run: one 100 ZRN balance passed
// from address to address inside one commit window. With the seat-deposit
// marker off it seats every address (five here), since the balance is only
// checked. With it on the first seat locks the balance, so the next address
// has nothing to be passed and is refused: N seats held at once need N
// deposits.
func TestSeatDepositRunOneBalancePassedAlong(t *testing.T) {
	for _, seat := range []bool{false, true} {
		t.Run(map[bool]string{false: "marker-off", true: "marker-on"}[seat], func(t *testing.T) {
			tr := newSeatTrial(t, "ordinary", seat, "")
			bank := tr.app.BankKeeper
			holder := tr.reviewer(t, 180, 100_000_000)
			seated := 0
			for i := byte(0); i < 5; i++ {
				if err := tr.commit(t, holder, "accept"); err != nil {
					require.True(t, seat, "with the marker off every pass-along seats")
					require.ErrorContains(t, err, "seat deposit required")
					break
				}
				seated++
				next := tr.reviewer(t, 181+i, 0)
				if spendable := bank.SpendableCoins(tr.ctx, holder); !spendable.IsZero() {
					require.NoError(t, bank.SendCoins(tr.ctx, holder, next, spendable))
				}
				holder = next
			}
			if seat {
				require.Equal(t, 1, seated)
				require.Equal(t, int64(100_000_000), tr.custody())
			} else {
				require.Equal(t, 5, seated)
				require.Zero(t, tr.custody())
			}
			require.Len(t, tr.stored(t).Commits, seated)
		})
	}
}

// reveal reveals every committed seat whose vote is not "", in commit order.
func (tr *seatTrial) revealAll(t *testing.T) {
	t.Helper()
	for _, seat := range tr.seats {
		if seat.vote == "" {
			continue
		}
		_, err := tr.server().SubmitReveal(tr.ctx, &knowledgetypes.MsgSubmitReveal{Verifier: seat.address.String(), RoundId: tr.round.Id, Vote: seat.vote, Confidence: 800000, Salt: seat.salt, Attestation: seatTrialAttestation})
		require.NoError(t, err)
	}
}

func (tr *seatTrial) beginBlock(t *testing.T, h int64) {
	t.Helper()
	tr.height(h)
	require.NoError(t, tr.app.KnowledgeKeeper.BeginBlocker(tr.ctx))
}

// D4, D5 with real bank transfers: on every admission route and every
// outcome, every seat's deposit comes back once, to the address that locked
// it, whatever it voted and whether it revealed; the seat-deposit account
// ends empty; and later blocks repeat nothing. Each
// reviewer ends with its starting balance plus its reviewer pay, which is
// unchanged by the deposit.
func TestSeatDepositEveryOutcomeReturnsEveryDepositOnce(t *testing.T) {
	const deposit, funded = 1000, 100_000
	for _, route := range seatTrialRoutes {
		// A round whose every seat reveals completes early, on the block
		// after the reveals (103); one with a silent seat waits for its
		// reveal deadline (104).
		for _, tc := range []struct {
			name     string
			votes    []string // "" commits to accept and never reveals
			complete int64    // height at which the round completes
			verdict  knowledgetypes.Verdict
		}{
			{"early-accept", []string{"accept", "accept", "accept"}, 103, knowledgetypes.Verdict_VERDICT_ACCEPT},
			{"early-reject-with-dissent", []string{"reject", "reject", "accept"}, 103, knowledgetypes.Verdict_VERDICT_REJECT},
			{"early-malformed", []string{"malformed", "malformed", "accept"}, 103, knowledgetypes.Verdict_VERDICT_MALFORMED},
			{"early-inconclusive-split", []string{"accept", "reject", "malformed"}, 103, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
			{"deadline-accept-with-silent-seat", []string{"accept", "accept", "accept", ""}, 104, knowledgetypes.Verdict_VERDICT_ACCEPT},
			{"deadline-under-quorum-silent-seat", []string{"accept", "reject", ""}, 104, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
			{"deadline-no-reveal", []string{"", "", ""}, 104, knowledgetypes.Verdict_VERDICT_INCONCLUSIVE},
		} {
			t.Run(route+"/"+tc.name, func(t *testing.T) {
				tr := newSeatTrial(t, route, true, "1000")
				require.Equal(t, "1000", tr.round.SeatTerms.CountedSeatDeposit)
				tr.ctx = tr.ctx.WithEventManager(sdk.NewEventManager())
				for i, vote := range tc.votes {
					reviewer := tr.reviewer(t, 190+byte(i), funded)
					if vote == "" {
						require.NoError(t, tr.commit(t, reviewer, "accept"))
						tr.seats[len(tr.seats)-1].vote = ""
					} else {
						require.NoError(t, tr.commit(t, reviewer, vote))
					}
					require.Equal(t, int64(funded-deposit), tr.balance(reviewer))
				}
				require.Equal(t, int64(len(tc.votes)*deposit), tr.custody())
				tr.beginBlock(t, 102)
				tr.revealAll(t)
				for h := int64(103); h <= tc.complete; h++ {
					tr.beginBlock(t, h)
				}
				round := tr.stored(t)
				require.Equal(t, knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, round.Phase)
				require.Equal(t, uint64(tc.complete), round.VerdictBlock)
				require.Equal(t, tc.verdict, round.Verdict)
				require.NotNil(t, round.SeatDepositReturns)
				require.Equal(t, round.VerdictBlock, round.SeatDepositReturns.CreatedAtBlock)

				tr.beginBlock(t, tc.complete+1)
				valid := 0
				for _, vote := range tc.votes {
					if vote != "" {
						valid++
					}
				}
				for i, seat := range tr.seats {
					pay := int64(0)
					if seat.vote != "" {
						pay = int64(55 / valid)
						if i == 0 {
							pay += int64(55 % valid)
						}
					}
					require.Equal(t, int64(funded)+pay, tr.balance(seat.address), "seat %d gets its whole deposit back", i)
				}
				require.Zero(t, tr.custody(), "the seat-deposit account ends empty")
				require.Equal(t, len(tc.votes), countEvents(tr.ctx, "zerone.knowledge.seat_deposit_returned"), "one transfer per seat")

				settled := tr.stored(t)
				for h := tc.complete + 2; h <= tc.complete+4; h++ {
					tr.beginBlock(t, h)
				}
				require.Equal(t, len(tc.votes), countEvents(tr.ctx, "zerone.knowledge.seat_deposit_returned"), "nothing is returned twice")
				require.Equal(t, settled.SeatDepositReturns.String(), tr.stored(t).SeatDepositReturns.String())
			})
		}
	}
}

// seatCustodyTrial is a round with three 1,000 uzrn seats committed, one each
// from reviewers holding 100,000 uzrn, on a chain with seat deposits on.
func seatCustodyTrial(t *testing.T) *seatTrial {
	t.Helper()
	tr := newSeatTrial(t, "ordinary", true, "1000")
	for i := byte(0); i < 3; i++ {
		require.NoError(t, tr.commit(t, tr.reviewer(t, 200+i, 100_000), "accept"))
	}
	require.Equal(t, int64(3000), tr.custody())
	return tr
}

// moveCustody moves amount uzrn out of (negative) or into (positive) the
// seat-deposit account through the bootstrap fund. It is a test-only
// movement that stands in for a shortfall or a stray coin, neither of which
// any message can produce.
func (tr *seatTrial) moveCustody(t *testing.T, amount int64) {
	t.Helper()
	bank := tr.app.BankKeeper
	if amount < 0 {
		coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", -amount))
		require.NoError(t, bank.SendCoinsFromModuleToModule(tr.ctx, knowledgetypes.SeatDepositModuleName, knowledgetypes.BootstrapFundModuleName, coins))
		return
	}
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
	require.NoError(t, bank.MintCoins(tr.ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, bank.SendCoinsFromModuleToModule(tr.ctx, knowledgetypes.BootstrapFundModuleName, knowledgetypes.SeatDepositModuleName, coins))
}

// exportImport exports every module's genesis from the trial's state and
// imports it with InitChain into a fresh application. On success it returns
// a trial on the imported chain, at the same height, with the same seats.
func (tr *seatTrial) exportImport(t *testing.T, logger log.Logger) (*seatTrial, error) {
	t.Helper()
	// admit writes a bare zerone_auth record (no DID or identity key), which
	// the auth export rightly refuses; registration is not what D6 carries,
	// so the export is taken from a cache without those records. Reveals and
	// returns on the imported chain need no registration.
	exportCtx, _ := tr.ctx.CacheContext()
	for _, address := range tr.admitted {
		exportCtx.KVStore(tr.app.keys[zeroneauthtypes.StoreKey]).Delete(zeroneauthtypes.AccountKey(address.String()))
	}
	exported, err := tr.app.ModuleManager.ExportGenesisForModules(exportCtx, tr.app.AppCodec(), nil)
	require.NoError(t, err)
	var knowledge knowledgetypes.GenesisState
	require.NoError(t, tr.app.AppCodec().UnmarshalJSON(exported[knowledgetypes.ModuleName], &knowledge))
	require.True(t, knowledge.SeatDepositEnabled, "the export carries the marker")
	raw, err := json.Marshal(exported)
	require.NoError(t, err)
	fresh := NewZeroneApp(logger, dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(seatTrialChainID))
	ctx := fresh.NewUncachedContext(false, cmtproto.Header{ChainID: seatTrialChainID})
	// The module manager panics when a module's InitGenesis refuses; a
	// refused import is returned here as that error.
	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		_, err = fresh.InitChainer(ctx, &abci.RequestInitChain{ChainId: seatTrialChainID, AppStateBytes: raw})
		return err
	}()
	if err != nil {
		return nil, err
	}
	h := tr.ctx.BlockHeight()
	imported := &seatTrial{app: fresh, ctx: ctx.WithBlockHeight(h).WithHeaderInfo(header.Info{Height: h, ChainID: seatTrialChainID}).WithEventManager(sdk.NewEventManager()), claim: tr.claim, round: tr.round, payer: tr.payer, seats: tr.seats}
	return imported, nil
}

// D6: the account balance covers locked deposits across export and import,
// at every stage of a round: seats locked, returns frozen and unpaid, and
// returns partly paid. Import rebuilds the payment queue from the round
// records, and afterwards every deposit comes back exactly once.
func TestSeatDepositCustodyAcrossExportImport(t *testing.T) {
	for _, stage := range []string{"seats-locked", "returns-frozen", "returns-partly-paid"} {
		t.Run(stage, func(t *testing.T) {
			tr := seatCustodyTrial(t)
			paidBefore := 0
			if stage != "seats-locked" {
				tr.beginBlock(t, 102)
				tr.revealAll(t)
				tr.beginBlock(t, 103)
				require.NotNil(t, tr.stored(t).SeatDepositReturns)
			}
			if stage == "returns-partly-paid" {
				// One deposit short: the first two returns are paid, the
				// third stays pending, and the block does not fail.
				tr.moveCustody(t, -1000)
				tr.beginBlock(t, 104)
				plan := tr.stored(t).SeatDepositReturns
				require.NotZero(t, plan.Returns[0].PaidAtBlock)
				require.NotZero(t, plan.Returns[1].PaidAtBlock)
				require.Zero(t, plan.Returns[2].PaidAtBlock)
				require.Error(t, tr.app.KnowledgeKeeper.ValidateSeatDepositCustody(tr.ctx), "a shortfall is visible")
				tr.moveCustody(t, 1000)
				paidBefore = 2
			}
			require.NoError(t, tr.app.KnowledgeKeeper.ValidateSeatDepositCustody(tr.ctx))
			before := tr.stored(t)
			custody := tr.custody()

			imported, err := tr.exportImport(t, log.NewNopLogger())
			require.NoError(t, err)
			require.Equal(t, custody, imported.custody())
			require.Equal(t, before.String(), imported.stored(t).String(), "the round, its seats and its returns travel exactly")
			require.NoError(t, imported.app.KnowledgeKeeper.ValidateSeatDepositCustody(imported.ctx))

			h := imported.ctx.BlockHeight()
			if stage == "seats-locked" {
				imported.beginBlock(t, 102)
				imported.revealAll(t)
				imported.beginBlock(t, 103)
				h = 103
			}
			imported.beginBlock(t, h+1)
			imported.beginBlock(t, h+2)
			require.Zero(t, imported.custody())
			require.Equal(t, 3-paidBefore, countEvents(imported.ctx, "zerone.knowledge.seat_deposit_returned"), "each return is paid once, across the import")
			for _, seat := range imported.seats {
				require.Equal(t, int64(100_000+55/3)+map[bool]int64{true: 55 % 3}[seat.address.Equals(imported.seats[0].address)], imported.balance(seat.address))
			}
		})
	}
}

// D6: import refuses a state whose seat-deposit account holds less than the
// locked, unreturned deposits, before any knowledge write. At run time the
// same shortfall is not a halt: blocks go on and the returns it cannot cover
// stay pending.
func TestSeatDepositImportRefusesShortfall(t *testing.T) {
	tr := seatCustodyTrial(t)
	tr.moveCustody(t, -1)
	_, err := tr.exportImport(t, log.NewNopLogger())
	require.ErrorContains(t, err, "less than the 3000 uzrn of locked, unreturned deposits")

	tr.beginBlock(t, 102)
	tr.revealAll(t)
	tr.beginBlock(t, 103)
	tr.beginBlock(t, 104)
	tr.beginBlock(t, 105)
	plan := tr.stored(t).SeatDepositReturns
	require.NotZero(t, plan.Returns[0].PaidAtBlock)
	require.NotZero(t, plan.Returns[1].PaidAtBlock)
	require.Zero(t, plan.Returns[2].PaidAtBlock, "the return the account cannot cover stays pending")
	require.Equal(t, int64(999), tr.custody())
}

// D6: a surplus in the seat-deposit account does not block an import. It is
// reported in the node log and not swept.
func TestSeatDepositImportReportsSurplusWithoutSweeping(t *testing.T) {
	tr := seatCustodyTrial(t)
	tr.moveCustody(t, 7)
	var logs bytes.Buffer
	imported, err := tr.exportImport(t, log.NewLogger(&logs, log.ColorOption(false)))
	require.NoError(t, err)
	require.Equal(t, int64(3007), imported.custody())
	require.Contains(t, logs.String(), "seat-deposit account surplus at import, reported and not swept")
	require.Contains(t, logs.String(), "surplus_uzrn=7")
}

// D6: the account balance covers locked deposits across a restart on the
// same database, and the round goes on to return every deposit.
func TestSeatDepositCustodyAcrossRestart(t *testing.T) {
	tr := seatCustodyTrial(t)
	before := tr.stored(t)
	tr.app.CommitMultiStore().Commit()
	reopened := NewZeroneApp(log.NewNopLogger(), tr.db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(seatTrialChainID))
	ctx := reopened.NewUncachedContext(false, cmtproto.Header{ChainID: seatTrialChainID})
	restarted := &seatTrial{app: reopened, ctx: ctx.WithBlockHeight(100).WithHeaderInfo(header.Info{Height: 100, ChainID: seatTrialChainID}).WithEventManager(sdk.NewEventManager()), round: tr.round, seats: tr.seats}
	require.Equal(t, int64(3000), restarted.custody())
	require.Equal(t, before.String(), restarted.stored(t).String())
	enabled, err := reopened.KnowledgeKeeper.SeatDepositEnabled(restarted.ctx)
	require.NoError(t, err)
	require.True(t, enabled)
	require.NoError(t, reopened.KnowledgeKeeper.ValidateSeatDepositCustody(restarted.ctx))
	restarted.beginBlock(t, 102)
	restarted.revealAll(t)
	restarted.beginBlock(t, 103)
	restarted.beginBlock(t, 104)
	require.Zero(t, restarted.custody())
	require.Equal(t, 3, countEvents(restarted.ctx, "zerone.knowledge.seat_deposit_returned"))
}
