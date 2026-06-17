package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/qualification/types"
)

// Keeper manages the qualification module's state.
type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	authority    string

	bankKeeper           types.BankKeeper
	stakingKeeper        types.StakingKeeper
	captureDefenseKeeper types.CaptureDefenseKeeper // nil-safe, set post-init
	ontologyKeeper       types.OntologyKeeper       // nil-safe, set post-init
}

// NewKeeper creates a new qualification module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	bk types.BankKeeper,
	sk types.StakingKeeper,
) Keeper {
	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		authority:     authority,
		bankKeeper:    bk,
		stakingKeeper: sk,
	}
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetCaptureDefenseKeeper sets the capture defense keeper post-initialization.
func (k *Keeper) SetCaptureDefenseKeeper(cdk types.CaptureDefenseKeeper) {
	k.captureDefenseKeeper = cdk
}

// SetOntologyKeeper sets the ontology keeper post-initialization.
func (k *Keeper) SetOntologyKeeper(ok types.OntologyKeeper) {
	k.ontologyKeeper = ok
}

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Genesis ----------

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	for _, q := range genState.Qualifications {
		k.SetQualification(ctx, q)
	}
	for _, e := range genState.Endorsements {
		k.SetEndorsement(ctx, e)
	}
	if genState.NextEndorsementId > 0 {
		k.setEndorsementCounter(ctx, genState.NextEndorsementId)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	qualifications := k.GetAllQualifications(ctx)
	endorsements := k.GetAllEndorsements(ctx)
	nextID := k.getEndorsementCounter(ctx)
	return &types.GenesisState{
		Params:            params,
		Qualifications:    qualifications,
		Endorsements:      endorsements,
		NextEndorsementId: nextID,
	}
}

// ---------- Cross-module interface methods ----------

// IsQualified returns true if the validator holds an active qualification in the domain.
func (k Keeper) IsQualified(ctx context.Context, validator string, domain string) bool {
	q, found := k.GetQualification(ctx, validator, domain)
	if !found {
		return false
	}
	return q.Status == types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE
}

// GetQualificationWeight returns the EFFECTIVE weight of a validator's
// qualification in a domain — the base Weight reduced by any active
// penalty written by capture_challenge on UPHELD cartel allegations.
//
// The penalty pathway is the soft, time-bounded counterpart to outright
// suspension. A validator implicated in a cartel doesn't necessarily
// lose their qualification entirely (UPHELDs may be partial; evidence
// may be ambiguous in scope); the penalty halves their weight for a
// configurable expiry window. After expiry the penalty is naturally
// ignored. Hard suspension (full status transition) remains an option
// for severe cases.
//
// Wiring this read closed a latent integration: ReduceQualificationWeight
// has been writing penalties since R28-8 but no consumer read them, so
// confirmed cartel members continued voting at full strength on the
// next panel.
func (k Keeper) GetQualificationWeight(ctx context.Context, validator string, domain string) uint32 {
	q, found := k.GetQualification(ctx, validator, domain)
	if !found {
		return 0
	}
	if q.Status != types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE {
		return 0
	}
	base := q.Weight
	penalty, ok := k.GetActiveQualificationPenalty(ctx, validator, domain)
	if !ok || penalty == nil {
		return base
	}
	// reduction is in BPS (0..1_000_000); apply: effective = base × (BPS - reduction) / BPS
	const bps uint64 = 1_000_000
	if penalty.ReductionBps >= bps {
		return 0
	}
	remaining := bps - penalty.ReductionBps
	effective := uint64(base) * remaining / bps
	if effective > uint64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(effective)
}

// GetActiveQualificationPenalty returns the current penalty for a
// (validator, domain) pair if one exists and has not expired. Returns
// nil if no penalty is recorded or the recorded one has expired.
func (k Keeper) GetActiveQualificationPenalty(ctx context.Context, validator, domain string) (*types.QualificationPenalty, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(append([]byte{}, types.QualificationPenaltyKeyPrefix...), []byte(validator+"/"+domain)...)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var p types.QualificationPenalty
	if err := json.Unmarshal(bz, &p); err != nil {
		return nil, false
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if p.ExpiryHeight > 0 && uint64(sdkCtx.BlockHeight()) >= p.ExpiryHeight {
		// Expired — caller will ignore. (Cleanup happens in BeginBlocker
		// or lazily on the next write; we don't delete here to keep the
		// read path read-only.)
		return nil, false
	}
	return &p, true
}

// GetQualifiedValidators returns all validators with active qualifications in a domain.
func (k Keeper) GetQualifiedValidators(ctx context.Context, domain string) []string {
	validators := k.GetValidatorsByDomain(ctx, domain)
	var qualified []string
	for _, v := range validators {
		if k.IsQualified(ctx, v, domain) {
			qualified = append(qualified, v)
		}
	}
	return qualified
}

// RecordVerificationOutcome records a verification outcome for a validator in a domain.
//
// Honest limit (named, not hidden): `correct` means "the validator's vote
// agreed with the panel's finalized verdict" — the chain grades a verifier
// against the panel's *own* output, not against an external truth. The chain
// has no external truth-oracle; it witnesses and keeps, it does not certify.
// So AccuracyBps is a consensus-coherence signal (how often a verifier
// agreed with the record the chain kept), not a truth-tracking signal. The
// chain cannot self-certify truth from its own consensus; it keeps a record
// of agreement. The circularity is by design — there is no outside oracle to
// break it, and naming it is the honesty.
func (k Keeper) RecordVerificationOutcome(ctx context.Context, validator string, domain string, correct bool) error {
	q, found := k.GetQualification(ctx, validator, domain)
	if !found {
		return fmt.Errorf("%w: %s/%s", types.ErrQualificationNotFound, validator, domain)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if q.Metrics == nil {
		q.Metrics = &types.QualificationMetrics{}
	}
	q.Metrics.TotalVerifications++
	if correct {
		q.Metrics.CorrectVerifications++
	}
	if q.Metrics.TotalVerifications > 0 {
		q.Metrics.AccuracyBps = (q.Metrics.CorrectVerifications * 1000000) / q.Metrics.TotalVerifications
	}
	q.Metrics.LastVerificationBlock = uint64(sdkCtx.BlockHeight())
	k.SetQualification(ctx, q)
	return nil
}

// ReduceQualificationWeight temporarily reduces a validator's qualification weight
// in a domain. The reduction expires at expiryHeight.
func (k Keeper) ReduceQualificationWeight(ctx context.Context, validator, domain string, reductionBps, expiryHeight uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	kvStore := k.storeService.OpenKVStore(ctx)

	key := append(append([]byte{}, types.QualificationPenaltyKeyPrefix...), []byte(validator+"/"+domain)...)
	penalty := &types.QualificationPenalty{
		Validator:    validator,
		Domain:       domain,
		ReductionBps: reductionBps,
		ExpiryHeight: expiryHeight,
		CreatedAt:    uint64(sdkCtx.BlockHeight()),
	}
	bz, err := json.Marshal(penalty)
	if err != nil {
		return fmt.Errorf("failed to marshal qualification penalty: %w", err)
	}
	return kvStore.Set(key, bz)
}

// prefixEndBytes returns the end key for prefix iteration (exclusive upper bound).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}
