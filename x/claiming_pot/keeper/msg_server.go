package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func checkedAddUint64(left, right uint64) (uint64, bool) {
	if right > ^uint64(0)-left {
		return 0, false
	}
	return left + right, true
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreatePot creates a legacy general claiming pot (authority-gated). General
// pots share the bootstrap lifetime issuance budget in fixed-size commitment
// units so this API cannot schedule uncapped native issuance.
func (m msgServer) CreatePot(goCtx context.Context, msg *types.MsgCreatePot) (*types.MsgCreatePotResponse, error) {
	if msg == nil {
		return nil, fmt.Errorf("%w: message must not be nil", types.ErrInvalidConfig)
	}
	if m.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, m.GetAuthority(), msg.Authority)
	}
	if msg.Name == "" {
		return nil, fmt.Errorf("%w: name must not be empty", types.ErrInvalidConfig)
	}
	if msg.Schedule == nil {
		return nil, fmt.Errorf("%w: schedule must not be nil", types.ErrInvalidConfig)
	}
	if msg.Schedule.EndBlock <= msg.Schedule.StartBlock {
		return nil, fmt.Errorf("%w: schedule end_block must be greater than start_block", types.ErrInvalidConfig)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check max active pots
	params := m.GetParams(ctx)
	if uint32(m.CountActivePots(ctx)) >= params.MaxPotsActive {
		return nil, types.ErrMaxPotsReached
	}

	units, err := types.PotCommitmentUnits(msg.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}

	// Charge ceil(total/per-agent-bootstrap) lifetime units. This intentionally
	// rounds up: the shared counter remains a conservative cap even though a
	// general pot may not have the standard bootstrap shape.
	perUnit, _ := new(big.Int).SetString(types.PerAgentBootstrapUzrn, 10)
	unitsBig := new(big.Int).SetUint64(units)
	committedUnits := m.GetBootstrapMintedEntries(ctx)
	if ^uint64(0)-committedUnits < units {
		return nil, types.ErrBootstrapEmissionCapExceeded
	}
	projectedUnits := new(big.Int).Add(
		new(big.Int).SetUint64(committedUnits),
		unitsBig,
	)
	projectedCommitment := new(big.Int).Mul(projectedUnits, perUnit)
	if projectedCommitment.Cmp(params.BootstrapEmissionCap()) > 0 {
		return nil, fmt.Errorf(
			"%w: general pot commits %s uzrn (%s fixed-size units) above remaining lifetime budget",
			types.ErrBootstrapEmissionCapExceeded,
			msg.TotalAmount,
			unitsBig,
		)
	}
	m.SetBootstrapMintedEntries(ctx, committedUnits+units)

	currentBlock := uint64(ctx.BlockHeight())
	nextPotID, err := m.GetNextPotID(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}
	potID := fmt.Sprintf("pot-%d", nextPotID)

	pot := &types.ClaimingPot{
		Id:             potID,
		Name:           msg.Name,
		TotalAmount:    msg.TotalAmount,
		ClaimedAmount:  "0",
		Schedule:       msg.Schedule,
		Eligibility:    msg.Eligibility,
		CreatedAtBlock: currentBlock,
		Status:         types.PotStatus_POT_STATUS_ACTIVE,
	}

	m.SetPot(ctx, pot)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.claiming_pot.pot_created",
			sdk.NewAttribute("pot_id", potID),
			sdk.NewAttribute("name", msg.Name),
			sdk.NewAttribute("total_amount", msg.TotalAmount),
		),
	)

	return &types.MsgCreatePotResponse{PotId: potID}, nil
}

// Claim processes a claim from an eligible claimant.
func (m msgServer) Claim(goCtx context.Context, msg *types.MsgClaim) (*types.MsgClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pot, found := m.GetPot(ctx, msg.PotId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrPotNotFound, msg.PotId)
	}

	if pot.Status != types.PotStatus_POT_STATUS_ACTIVE {
		return nil, fmt.Errorf("%w: pot status is %s", types.ErrPotNotActive, pot.Status.String())
	}

	// Check eligibility
	if err := m.checkEligibility(ctx, msg.Claimant, pot); err != nil {
		return nil, err
	}

	// Check not already claimed
	if _, exists := m.GetClaim(ctx, msg.PotId, msg.Claimant); exists {
		return nil, types.ErrAlreadyClaimed
	}

	// Calculate vested-but-unclaimed
	currentBlock := uint64(ctx.BlockHeight())
	claimable := CalculateClaimable(pot, currentBlock)
	if claimable.Sign() <= 0 {
		return nil, types.ErrCliffNotReached
	}

	// Check min claim amount
	params := m.GetParams(ctx)
	minClaim := new(big.Int)
	minClaim.SetString(params.MinClaimAmount, 10)
	if claimable.Cmp(minClaim) < 0 {
		return nil, fmt.Errorf("%w: claimable %s < min %s", types.ErrBelowMinClaim, claimable.String(), params.MinClaimAmount)
	}

	// Check pot has sufficient remaining funds
	totalAmount := new(big.Int)
	totalAmount.SetString(pot.TotalAmount, 10)
	claimedAmount := new(big.Int)
	claimedAmount.SetString(pot.ClaimedAmount, 10)
	remaining := new(big.Int).Sub(totalAmount, claimedAmount)
	if claimable.Cmp(remaining) > 0 {
		claimable = remaining
	}
	if claimable.Sign() <= 0 {
		return nil, types.ErrInsufficientPotFunds
	}

	// Mint into the claiming_pot module account through the chain's
	// single cap-gated mint entry point, then forward to the claimer
	// in the same transaction. The module account is a transient
	// conduit — it never holds funds across blocks.
	//
	// Doctrine (commitment 20, issuance follows participation): the
	// chain mints when an agent participates (this MsgClaim) — not
	// from a pre-funded module account.
	claimantAddr, err := sdk.AccAddressFromBech32(msg.Claimant)
	if err != nil {
		return nil, fmt.Errorf("invalid claimant address: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	actualMinted, err := m.vestingRewardsKeeper.MintWithCap(sdkCtx, types.ModuleName, claimable)
	if err != nil {
		return nil, fmt.Errorf("mint with cap: %w", err)
	}
	if actualMinted.Sign() <= 0 {
		return nil, types.ErrCapReached
	}
	// MintWithCap may have clipped to remaining cap headroom; honour
	// that clip rather than over-promising the claimer.
	if actualMinted.Cmp(claimable) < 0 {
		claimable = actualMinted
	}

	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(claimable)))
	if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimantAddr, coins); err != nil {
		return nil, fmt.Errorf("forward minted bootstrap claim to claimer: %w", err)
	}

	// Record claim
	claim := &types.Claim{
		PotId:     msg.PotId,
		Claimant:  msg.Claimant,
		Amount:    claimable.String(),
		ClaimedAt: currentBlock,
	}
	m.SetClaim(ctx, claim)

	// Update pot claimed amount
	claimedAmount.Add(claimedAmount, claimable)
	pot.ClaimedAmount = claimedAmount.String()
	if claimedAmount.Cmp(totalAmount) >= 0 {
		pot.Status = types.PotStatus_POT_STATUS_DEPLETED
	}
	m.SetPot(ctx, pot)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.claiming_pot.pot_claimed",
			sdk.NewAttribute("pot_id", msg.PotId),
			sdk.NewAttribute("claimant", msg.Claimant),
			sdk.NewAttribute("amount", claimable.String()),
			// Commitment 20: issuance follows participation. Every
			// uzrn this event announces was minted on demand through
			// MintWithCap when the agent called MsgClaim — never
			// pre-funded.
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgClaimResponse{Amount: claimable.String()}, nil
}

// AddBootstrapEntry creates one bootstrap pot per address in msg.Addresses.
// Idempotent: addresses already represented by a bootstrap pot are silently
// skipped (and consume no cap budget). The doctrine is commitment 20
// extended to continuous, bounded authority-gated entry — the participant set is
// plural and growing, not closed at genesis.
//
// Two signers are accepted:
//   - the gov authority (always), and
//   - Params.BootstrapRegistrar when non-empty (the agenttool ops multisig;
//     revocable by a single param change back to "").
//
// Compromise bounds live IN CONSENSUS, not off-chain policy:
//   - lifetime: (entries ever created) x 222,000 uzrn must never exceed
//     Params.BootstrapEmissionCapUzrn — enforced for gov AND registrar,
//     since the cap is a supply commitment, not a rate limit;
//   - rate: registrar admissions are capped at
//     Params.BootstrapDailyAdmissionCap per 34,272-block window. Gov
//     admissions BYPASS the window and do not consume it — governance is
//     already throttled by the proposal process, and a stolen registrar
//     must not be able to starve gov's own admission path.
//
// A batch that would breach either cap fails atomically (no partial
// admission), so the operator can split or wait for the window to roll.
//
// Each created pot is shaped by MakeBootstrapPotForAgent with the current
// block height, so vesting starts now and the pot remains claimable until
// claimed (bootstrap pots do not auto-expire — see ProcessPotExpiry).
func (m msgServer) AddBootstrapEntry(goCtx context.Context, msg *types.MsgAddBootstrapEntry) (*types.MsgAddBootstrapEntryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)

	isGov := m.GetAuthority() == msg.Authority
	isRegistrar := !isGov && params.BootstrapRegistrar != "" && params.BootstrapRegistrar == msg.Authority
	if !isGov && !isRegistrar {
		return nil, fmt.Errorf("%w: expected gov authority %s or bootstrap registrar %q, got %s",
			types.ErrUnauthorized, m.GetAuthority(), params.BootstrapRegistrar, msg.Authority)
	}

	// Defensive re-check of the ValidateBasic bound: msg-server is the last
	// line, and an unbounded batch means unbounded state writes.
	if len(msg.Addresses) > types.MaxBootstrapAddressesPerMsg {
		return nil, fmt.Errorf("too many addresses: %d > max %d per message", len(msg.Addresses), types.MaxBootstrapAddressesPerMsg)
	}

	currentBlock := uint64(ctx.BlockHeight())

	// First pass: validate and split into new-vs-existing, so cap budgets
	// are charged only for entries that will actually be created.
	toAdd := make([]string, 0, len(msg.Addresses))
	skippedCount := uint32(0)
	for i, addr := range msg.Addresses {
		// Defensive: ValidateBasic catches these too, but msg-server is the
		// last line and we validate explicitly so a malformed entry can't
		// reach SetPot.
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return nil, fmt.Errorf("addresses[%d] (%q): invalid bech32: %w", i, addr, err)
		}
		if _, exists := m.GetPot(ctx, types.BootstrapPotIDPrefix+addr); exists {
			skippedCount++
			continue
		}
		toAdd = append(toAdd, addr)
	}

	if len(toAdd) > 0 {
		wouldAdd := uint64(len(toAdd))

		// Lifetime emission cap (gov and registrar alike).
		perEntry, _ := new(big.Int).SetString(types.PerAgentBootstrapUzrn, 10)
		minted := m.GetBootstrapMintedEntries(ctx)
		nextMinted, ok := checkedAddUint64(minted, wouldAdd)
		if !ok {
			return nil, fmt.Errorf(
				"%w: %d existing + %d new entries exceed the uint64 lifetime-accounting range",
				types.ErrBootstrapEmissionCapExceeded,
				minted,
				wouldAdd,
			)
		}
		projected := new(big.Int).Mul(new(big.Int).SetUint64(nextMinted), perEntry)
		if projected.Cmp(params.BootstrapEmissionCap()) > 0 {
			return nil, fmt.Errorf("%w: %d existing + %d new entries commit %s uzrn > cap %s",
				types.ErrBootstrapEmissionCapExceeded, minted, wouldAdd, projected, params.BootstrapEmissionCap())
		}

		// Per-window rate cap (registrar only; gov bypasses, see doc above).
		if isRegistrar {
			window := currentBlock / types.BootstrapAdmissionWindowBlocks
			windowCount := m.GetBootstrapWindowCount(ctx, window)
			nextWindowCount, ok := checkedAddUint64(windowCount, wouldAdd)
			if !ok {
				return nil, fmt.Errorf(
					"%w: %d already admitted + %d new exceed the uint64 window-accounting range (window %d)",
					types.ErrBootstrapDailyCapExceeded,
					windowCount,
					wouldAdd,
					window,
				)
			}
			if nextWindowCount > params.BootstrapDailyAdmissionCap {
				return nil, fmt.Errorf("%w: %d already admitted + %d new > cap %d (window %d)",
					types.ErrBootstrapDailyCapExceeded, windowCount, wouldAdd, params.BootstrapDailyAdmissionCap, window)
			}
			m.SetBootstrapWindowCount(ctx, window, nextWindowCount)
		}

		m.SetBootstrapMintedEntries(ctx, nextMinted)
	}

	addedCount := uint32(0)
	for _, addr := range toAdd {
		potID := types.BootstrapPotIDPrefix + addr
		pot := types.MakeBootstrapPotForAgent(addr, currentBlock)
		m.SetPot(ctx, pot)
		addedCount++

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.claiming_pot.bootstrap_entry_added",
				sdk.NewAttribute("address", addr),
				sdk.NewAttribute("pot_id", potID),
				sdk.NewAttribute("block", fmt.Sprintf("%d", currentBlock)),
				// Commitment 20 (extended): issuance follows participation,
				// continuously and authority-gated. This event announces a
				// new participant admitted post-genesis via gov authority
				// or the configured, capped registrar.
				sdk.NewAttribute("creed_commitment", "20"),
			),
		)
	}

	return &types.MsgAddBootstrapEntryResponse{
		AddedCount:   addedCount,
		SkippedCount: skippedCount,
	}, nil
}

// UpdatePotParams updates module parameters (authority-gated).
func (m msgServer) UpdatePotParams(goCtx context.Context, msg *types.MsgUpdatePotParams) (*types.MsgUpdatePotParamsResponse, error) {
	if msg == nil {
		return nil, fmt.Errorf("%w: message must not be nil", types.ErrInvalidConfig)
	}
	if m.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, m.GetAuthority(), msg.Authority)
	}
	if msg.Params == nil {
		return nil, fmt.Errorf("%w: params must not be nil", types.ErrInvalidConfig)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	perUnit, _ := new(big.Int).SetString(types.PerAgentBootstrapUzrn, 10)
	committed := new(big.Int).Mul(
		new(big.Int).SetUint64(m.GetBootstrapMintedEntries(ctx)),
		perUnit,
	)
	if committed.Cmp(msg.Params.BootstrapEmissionCap()) > 0 {
		return nil, fmt.Errorf(
			"%w: bootstrap_emission_cap_uzrn %s is below existing lifetime commitment %s",
			types.ErrInvalidConfig,
			msg.Params.BootstrapEmissionCap(),
			committed,
		)
	}
	currentWindow := uint64(ctx.BlockHeight()) / types.BootstrapAdmissionWindowBlocks
	admissionCount := m.GetBootstrapWindowCount(ctx, currentWindow)
	if admissionCount > uint64(msg.Params.BootstrapDailyAdmissionCap) {
		return nil, fmt.Errorf(
			"%w: bootstrap_daily_admission_cap %d is below current window count %d",
			types.ErrInvalidConfig,
			msg.Params.BootstrapDailyAdmissionCap,
			admissionCount,
		)
	}
	m.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.claiming_pot.update_pot_params",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdatePotParamsResponse{}, nil
}

// checkEligibility verifies the claimant meets the pot's eligibility criteria.
func (m msgServer) checkEligibility(ctx context.Context, claimant string, pot *types.ClaimingPot) error {
	if pot.Eligibility == nil {
		return nil // no criteria = everyone eligible
	}

	elig := pot.Eligibility

	// Check whitelist (if non-empty, must be in it)
	if len(elig.Whitelist) > 0 {
		found := false
		for _, addr := range elig.Whitelist {
			if addr == claimant {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: address not in whitelist", types.ErrIneligible)
		}
	}

	// Check staking tier
	if elig.MinStakingTier > 0 {
		tier, err := m.stakingKeeper.GetValidatorTier(ctx, claimant)
		if err != nil {
			return fmt.Errorf("%w: cannot determine tier: %v", types.ErrIneligible, err)
		}
		if tier < elig.MinStakingTier {
			return fmt.Errorf("%w: tier %d < required %d", types.ErrIneligible, tier, elig.MinStakingTier)
		}
	}

	// Check registration age
	if elig.MinRegistrationAge > 0 {
		regBlock, err := m.authKeeper.GetRegistrationBlock(ctx, claimant)
		if err != nil {
			return fmt.Errorf("%w: cannot determine registration age: %v", types.ErrIneligible, err)
		}
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		currentBlock := uint64(sdkCtx.BlockHeight())
		if regBlock == 0 || currentBlock-regBlock < elig.MinRegistrationAge {
			return fmt.Errorf("%w: registration age insufficient", types.ErrIneligible)
		}
	}

	return nil
}
