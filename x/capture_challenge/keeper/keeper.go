package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// Keeper manages the capture_challenge module's state.
type Keeper struct {
	storeService         store.KVStoreService
	cdc                  codec.BinaryCodec
	authority            string
	bankKeeper           types.BankKeeper
	captureDefenseKeeper types.CaptureDefenseKeeper
	qualificationKeeper  types.DomainQualificationKeeper // nil-safe, set post-init
	knowledgeKeeper      types.KnowledgeKeeper           // nil-safe, set post-init
}

// NewKeeper creates a new capture_challenge module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bankKeeper types.BankKeeper,
) Keeper {
	return Keeper{
		storeService: storeService,
		cdc:          cdc,
		authority:    authority,
		bankKeeper:   bankKeeper,
	}
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string { return k.authority }

// SetCaptureDefenseKeeper sets the capture defense keeper post-initialization.
func (k *Keeper) SetCaptureDefenseKeeper(cdk types.CaptureDefenseKeeper) {
	k.captureDefenseKeeper = cdk
}

// SetQualificationKeeper sets the domain qualification keeper post-initialization.
func (k *Keeper) SetQualificationKeeper(qk types.DomainQualificationKeeper) {
	k.qualificationKeeper = qk
}

// SetKnowledgeKeeper sets the knowledge keeper post-initialization.
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) {
	k.knowledgeKeeper = kk
}

// GenerateChallengeID creates a deterministic ID from challenger + domain + block.
func GenerateChallengeID(challenger, domain string, blockHeight int64) string {
	data := fmt.Sprintf("%s:%s:%d", challenger, domain, blockHeight)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // 32-char hex
}

// ---------- Genesis ----------

// InitGenesis sets initial state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if gs.Params != nil {
		k.SetParams(ctx, gs.Params)
	}
	for _, ch := range gs.Challenges {
		k.SetChallenge(ctx, ch)
	}
	for _, bp := range gs.BountyPools {
		k.SetBountyPool(ctx, bp)
	}
	// Only an explicit true writes the marker; absence keeps legacy semantics.
	if gs.ChallengeAccountingEnabled {
		if err := k.EnableChallengeAccounting(ctx); err != nil {
			panic("invalid capture_challenge genesis: " + err.Error())
		}
	}
}

// ExportGenesis exports the current state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	challengeAccounting, err := k.ChallengeAccountingEnabled(ctx)
	if err != nil {
		panic(err)
	}
	return &types.GenesisState{
		Params:                     k.GetParams(ctx),
		Challenges:                 k.GetAllChallenges(ctx),
		BountyPools:                k.GetAllBountyPools(ctx),
		ChallengeAccountingEnabled: challengeAccounting,
	}
}

// ---------- BeginBlocker ----------

// BeginBlocker handles phase advancement, auto-fund, and risk analysis.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	accountingEnabled, err := k.ChallengeAccountingEnabled(ctx)
	if err != nil {
		return err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := k.GetParams(sdkCtx)
	height := uint64(sdkCtx.BlockHeight())

	// Advance challenge phases
	k.advanceChallengePhases(sdkCtx, height, accountingEnabled)

	// Periodic risk analysis
	if height > 0 && params.RiskAnalysisInterval > 0 && height%params.RiskAnalysisInterval == 0 {
		k.RunRiskAnalysis(sdkCtx)
	}

	return nil
}

// AdvanceChallengePhases transitions challenges based on deadlines.
// Marker errors fail closed before any challenge is changed.
func (k Keeper) AdvanceChallengePhases(ctx sdk.Context, height uint64) error {
	accountingEnabled, err := k.ChallengeAccountingEnabled(ctx)
	if err != nil {
		return err
	}
	k.advanceChallengePhases(ctx, height, accountingEnabled)
	return nil
}

// advanceChallengePhases uses the entry point's selection, never a per-record read.
func (k Keeper) advanceChallengePhases(ctx sdk.Context, height uint64, accountingEnabled bool) {
	if !accountingEnabled {
		k.advanceChallengePhasesLegacy(ctx, height)
		return
	}
	k.IterateChallenges(ctx, func(ch *types.CaptureChallenge) bool {
		switch ch.Status {
		case types.ChallengeStatus_CHALLENGE_STATUS_OPEN,
			types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE:
			// Transition to under_review at evidence deadline
			if height >= ch.EvidenceDeadline {
				ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW
				k.SetChallenge(ctx, ch)
			}
		case types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW:
			// With accounting enabled, overdue UNDER_REVIEW means refund-pending.
			// Keep that durable record until payment and EXPIRED commit together.
			if height >= ch.ReviewDeadline {
				if err := k.expireChallenge(ctx, ch); err != nil {
					// Retry every block, but announce only at the deadline and
					// every 100 blocks after it. The failed cache is discarded;
					// this parent-context event reports the retained obligation.
					if (height-ch.ReviewDeadline)%100 == 0 {
						k.Logger(ctx).Error("expiry refund pending", "challenge_id", ch.Id, "error", err)
						ctx.EventManager().EmitEvent(sdk.NewEvent(
							"zerone.capture_challenge.expiry_refund_pending",
							sdk.NewAttribute("challenge_id", ch.Id),
							sdk.NewAttribute("domain", ch.Domain),
							sdk.NewAttribute("stake", ch.Stake),
							sdk.NewAttribute("review_deadline", fmt.Sprintf("%d", ch.ReviewDeadline)),
						))
					}
				}
			}
		}
		return false
	})
}

// expireChallenge commits one refund and its terminal status atomically. A bank
// or record-write error discards that challenge's effects; other challenges can
// still advance and the original record will be retried on the next block.
func (k Keeper) expireChallenge(ctx sdk.Context, ch *types.CaptureChallenge) error {
	stakeAmt, ok := new(big.Int).SetString(ch.Stake, 10)
	if !ok || stakeAmt.Sign() < 0 || stakeAmt.BitLen() > sdkmath.MaxBitLen {
		return fmt.Errorf("invalid expiry stake: %q", ch.Stake)
	}

	cacheCtx, write := ctx.CacheContext()
	if stakeAmt.Sign() > 0 {
		challengerAddr, err := sdk.AccAddressFromBech32(ch.Challenger)
		if err != nil {
			return fmt.Errorf("invalid refund address: %w", err)
		}
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, challengerAddr, coins); err != nil {
			return fmt.Errorf("failed to refund expiry stake: %w", err)
		}
	}
	// Zero-stake automatic challenges have no refund obligation.
	ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED
	if err := k.setChallenge(cacheCtx, ch); err != nil {
		return fmt.Errorf("failed to record expiry: %w", err)
	}
	write()
	return nil
}

// RunRiskAnalysis queries capture defense keeper for flagged domains.
func (k Keeper) RunRiskAnalysis(ctx sdk.Context) {
	// No-op if defense keeper not wired
	if k.captureDefenseKeeper == nil {
		return
	}
	// Risk analysis is primarily done by capture_defense BeginBlocker.
	// This is a hook for challenge module to take action on flagged domains.
}
