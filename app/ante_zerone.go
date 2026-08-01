package app

import (
	"fmt"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	zeroneauthkeeper "github.com/zerone-chain/zerone/x/auth/keeper"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
	emergencykeeper "github.com/zerone-chain/zerone/x/emergency/keeper"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
)

const (
	// Emergency executable inspection runs before signature verification and
	// gas charging. Bound every dimension controlled by transaction bytes so a
	// recursive authz/governance wrapper cannot turn the isolation check into
	// unmetered work.
	maxEmergencyExecutableDepth    = 16
	maxEmergencyExecutableMessages = 256
	maxEmergencyExecutableAnyBytes = 1 << 20
)

// getAuthenticatedSignerAddresses returns the signers declared by the
// transaction's messages. Standard Cosmos signature verification has already
// authenticated these addresses by the time the Zerone decorators run.
//
// SignerInfo.public_key is optional once x/auth has stored an account key, so
// deriving addresses from SignatureV2.PubKey would silently skip legitimate
// signers whenever that field is omitted. Every error is returned to callers:
// account policy must fail closed when signer extraction is unavailable.
func getAuthenticatedSignerAddresses(tx sdk.Tx) ([]sdk.AccAddress, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, errors.Wrap(sdkerrors.ErrTxDecode, "transaction must implement SigVerifiableTx")
	}

	signerBytes, err := sigTx.GetSigners()
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrTxDecode, fmt.Sprintf("extract transaction signers: %v", err))
	}
	if len(signerBytes) == 0 {
		return nil, errors.Wrap(sdkerrors.ErrTxDecode, "transaction has no authenticated signers")
	}

	addrs := make([]sdk.AccAddress, len(signerBytes))
	for i, address := range signerBytes {
		if err := sdk.VerifyAddressFormat(address); err != nil {
			return nil, errors.Wrapf(sdkerrors.ErrTxDecode, "transaction signer %d has an invalid address: %v", i, err)
		}
		addrs[i] = sdk.AccAddress(address)
	}
	return addrs, nil
}

// ---------- BootstrapGasFreeDecorator ----------

// BootstrapGasFreeDecorator waives gas and fees for essential PoT transactions
// during the bootstrap period (blocks 1..BootstrapEndBlock).
//
// RETIRED at mainnet genesis: BootstrapEndBlock is 0, so this decorator is a
// no-op for every block height >= 1. Onboarding subsidy is feegrant-based
// (design doc 2026-07-07 §10). The decorator stays wired so a future
// governance-approved upgrade can re-activate a window without re-plumbing
// the ante chain.
type BootstrapGasFreeDecorator struct{}

// NewBootstrapGasFreeDecorator creates a new BootstrapGasFreeDecorator.
func NewBootstrapGasFreeDecorator() BootstrapGasFreeDecorator {
	return BootstrapGasFreeDecorator{}
}

func (bgd BootstrapGasFreeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if ctx.BlockHeight() > BootstrapEndBlock {
		return next(ctx, tx, simulate)
	}

	// Check if ALL messages in the tx are bootstrap-eligible
	msgs := tx.GetMsgs()
	allEligible := len(msgs) > 0
	for _, msg := range msgs {
		if !BootstrapGasFreeTypes[sdk.MsgTypeURL(msg)] {
			allEligible = false
			break
		}
	}

	if !allEligible {
		return next(ctx, tx, simulate)
	}

	// Set gas meter to the block gas limit so bootstrap txs can consume
	// gas freely without hitting out-of-gas errors. We can't use
	// InfiniteGasMeter or math.MaxInt64 because CometBFT's mempool
	// rejects txs with gas_wanted exceeding ConsensusParams.Block.MaxGas.
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(BlockGasLimit))

	return next(ctx, tx, simulate)
}

// ---------- EmergencyHaltDecorator ----------

// EmergencyHaltDecorator enforces an application transaction quarantine. It
// does NOT halt CometBFT consensus: blocks, PreBlock, BeginBlock, and EndBlock
// continue. Emergency coordination messages and a narrowly scoped expedited
// software-upgrade governance lane remain available for recovery.
//
// Placed early in the ante chain (after SetUpContext, before gas/fee processing)
// so quarantined transactions don't consume gas or fees.
type EmergencyHaltDecorator struct {
	ek                     emergencykeeper.Keeper
	recoveryProposalReader EmergencyRecoveryProposalReader
}

// EmergencyRecoveryProposalReader identifies SDK governance proposals whose
// only action is an expedited software upgrade or upgrade cancellation.
// Quarantine must not reopen a general governance execution lane.
type EmergencyRecoveryProposalReader interface {
	IsAuthorizedRecoverySubmission(
		ctx sdk.Context,
		msg *govv1.MsgSubmitProposal,
	) bool
	IsExpeditedRecoveryProposal(ctx sdk.Context, proposalID uint64) bool
}

type emergencyGovernanceReviewReader interface {
	IsGovernanceReviewHeld(ctx sdk.Context) (bool, error)
}

// NewEmergencyHaltDecorator creates a new EmergencyHaltDecorator.
//
// The optional reader keeps focused decorator tests and downstream integrations
// source-compatible. Without a reader, only proposal submission (which can be
// checked from the message itself) is available; votes and deposits fail
// closed because their target proposal cannot be authenticated.
func NewEmergencyHaltDecorator(
	ek emergencykeeper.Keeper,
	readers ...EmergencyRecoveryProposalReader,
) EmergencyHaltDecorator {
	var reader EmergencyRecoveryProposalReader
	if len(readers) > 0 {
		reader = readers[0]
	}
	return EmergencyHaltDecorator{ek: ek, recoveryProposalReader: reader}
}

func (ehd EmergencyHaltDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()
	scan, err := scanEmergencyExecutables(msgs)
	if err != nil {
		return ctx, errors.Wrap(
			sdkerrors.ErrTxDecode,
			fmt.Sprintf("inspect wrapped executable messages for emergency isolation: %v", err),
		)
	}
	if scan.containsWrappedEmergency {
		// Emergency transitions must execute as direct, isolated top-level
		// messages. This rejects authz and SDK-governance wrappers,
		// including recursive combinations of the two.
		return ctx, emergencytypes.ErrUnsafeEmergencyBatch
	}
	if scan.containsSDKGovMutation || scan.containsICAHostReceive {
		if reviewReader, ok :=
			ehd.recoveryProposalReader.(emergencyGovernanceReviewReader); ok {
			reviewHeld, err := reviewReader.IsGovernanceReviewHeld(ctx)
			if err != nil {
				return ctx, errors.Wrap(
					zeronegovtypes.ErrEmergencyTransitionHold,
					fmt.Sprintf(
						"read post-incident governance review hold: %v",
						err,
					),
				)
			}
			if reviewHeld && !ehd.ek.IsHalted(ctx) {
				return ctx, errors.Wrap(
					zeronegovtypes.ErrEmergencyTransitionHold,
					"standard SDK governance admission and ICA-host callbacks are frozen until a named upgrade completes post-incident queue reconciliation",
				)
			}
		}
	}

	containsEmergencyMessage := false
	for _, msg := range msgs {
		if emergencytypes.IsEmergencyMsg(msg) {
			containsEmergencyMessage = true
		}
	}
	if containsEmergencyMessage {
		// Ante runs once before all messages execute. A final halt precommit
		// can activate quarantine midway through a multi-message transaction,
		// so no ordinary tail may share that atomic execution context.
		for _, msg := range msgs {
			if !emergencytypes.IsEmergencyMsg(msg) {
				return ctx, emergencytypes.ErrUnsafeEmergencyBatch
			}
		}
	}

	if simulate {
		return next(ctx, tx, simulate)
	}

	status := ehd.ek.GetEmergencyStatus(ctx)
	restricted := ehd.ek.IsHalted(ctx)
	if !restricted && status == emergencytypes.StatusHaltVoting {
		// Opening a halt vote is not itself authority to quarantine ordinary
		// transactions. Only block ICA-host callbacks here because their
		// embedded SDK-message batch bypasses this ante handler and could
		// finalize the halt before executing an ordinary tail.
		if scan.containsICAHostReceive {
			return ctx, emergencytypes.ErrChainHalted
		}
		return next(ctx, tx, simulate)
	}

	if !restricted {
		return next(ctx, tx, simulate)
	}

	// Application transaction quarantine is active.
	for _, msg := range msgs {
		if emergencytypes.IsEmergencyMsg(msg) {
			continue
		}
		if !ehd.isRecoveryGovernanceMessage(ctx, msg) {
			return ctx, emergencytypes.ErrChainHalted
		}
		// Ante observes one pre-message state snapshot. Two authorized
		// submissions in one transaction could both see the same next SDK
		// proposal ID before the first consumes it, so every recovery
		// governance action must be isolated.
		if len(msgs) != 1 {
			return ctx, emergencytypes.ErrUnsafeEmergencyBatch
		}
	}

	return next(ctx, tx, simulate)
}

// EmergencyAuthenticationDecorator marks a direct, isolated emergency
// transaction only after the standard signature-verification decorator has
// authenticated its declared signers. Message servers require this marker, so
// SDK governance EndBlock execution, IBC callbacks, authz wrappers, and direct
// router calls cannot impersonate Guardian addresses.
type EmergencyAuthenticationDecorator struct{}

// NewEmergencyAuthenticationDecorator creates the post-signature marker.
func NewEmergencyAuthenticationDecorator() EmergencyAuthenticationDecorator {
	return EmergencyAuthenticationDecorator{}
}

func (EmergencyAuthenticationDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return next(ctx, tx, simulate)
	}
	for _, msg := range msgs {
		if !emergencytypes.IsEmergencyMsg(msg) {
			return next(ctx, tx, simulate)
		}
	}
	return next(emergencytypes.WithAuthenticatedEmergencyTx(ctx), tx, simulate)
}

type emergencyExecutableScanResult struct {
	containsEmergency        bool
	containsWrappedEmergency bool
	containsICAHostReceive   bool
	containsSDKGovMutation   bool
}

type emergencyExecutableScanItem struct {
	msg   sdk.Msg
	depth int
}

func scanEmergencyExecutables(
	msgs []sdk.Msg,
) (emergencyExecutableScanResult, error) {
	result := emergencyExecutableScanResult{}
	stack := make([]emergencyExecutableScanItem, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		stack = append(stack, emergencyExecutableScanItem{msg: msgs[i]})
	}

	nodes := 0
	anyBytes := 0
	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if item.msg == nil {
			return result, fmt.Errorf("executable message is nil")
		}
		if item.depth > maxEmergencyExecutableDepth {
			return result, fmt.Errorf(
				"executable wrapper depth %d exceeds limit %d",
				item.depth,
				maxEmergencyExecutableDepth,
			)
		}
		nodes++
		if nodes > maxEmergencyExecutableMessages {
			return result, fmt.Errorf(
				"executable message count exceeds limit %d",
				maxEmergencyExecutableMessages,
			)
		}

		if emergencytypes.IsEmergencyMsg(item.msg) {
			result.containsEmergency = true
			if item.depth > 0 {
				result.containsWrappedEmergency = true
			}
		}
		typeURL := sdk.MsgTypeURL(item.msg)
		if strings.HasPrefix(typeURL, "/cosmos.gov.v1.Msg") ||
			strings.HasPrefix(typeURL, "/cosmos.gov.v1beta1.Msg") {
			result.containsSDKGovMutation = true
		}
		if recv, ok := item.msg.(*channeltypes.MsgRecvPacket); ok &&
			recv.Packet.DestinationPort == icatypes.HostPortID {
			result.containsICAHostReceive = true
		}

		var messages []*codectypes.Any
		switch typed := item.msg.(type) {
		case *authz.MsgExec:
			messages = typed.Msgs
		case *govv1.MsgSubmitProposal:
			messages = typed.Messages
		default:
			continue
		}
		if item.depth == maxEmergencyExecutableDepth && len(messages) > 0 {
			return result, fmt.Errorf(
				"executable wrapper depth exceeds limit %d",
				maxEmergencyExecutableDepth,
			)
		}
		for i := len(messages) - 1; i >= 0; i-- {
			message := messages[i]
			if message == nil {
				return result, fmt.Errorf("executable wrapper contains a nil Any")
			}
			encodedSize := len(message.TypeUrl) + len(message.Value)
			if encodedSize > maxEmergencyExecutableAnyBytes-anyBytes {
				return result, fmt.Errorf(
					"executable Any bytes exceed limit %d",
					maxEmergencyExecutableAnyBytes,
				)
			}
			anyBytes += encodedSize
			if emergencytypes.IsEmergencyTypeURL(message.TypeUrl) {
				result.containsEmergency = true
				result.containsWrappedEmergency = true
			}
			if message.TypeUrl == sdk.MsgTypeURL(&channeltypes.MsgRecvPacket{}) {
				nested, ok := message.GetCachedValue().(*channeltypes.MsgRecvPacket)
				if !ok {
					return result, fmt.Errorf(
						"ICA candidate %s is not unpacked",
						message.TypeUrl,
					)
				}
				if nested.Packet.DestinationPort == icatypes.HostPortID {
					result.containsICAHostReceive = true
				}
			}
			nested, ok := message.GetCachedValue().(sdk.Msg)
			if !ok {
				return result, fmt.Errorf(
					"executable wrapper message %s is not unpacked",
					message.TypeUrl,
				)
			}
			stack = append(stack, emergencyExecutableScanItem{
				msg:   nested,
				depth: item.depth + 1,
			})
		}
	}
	return result, nil
}

func containsWrappedEmergencyMessage(msg sdk.Msg) (bool, error) {
	result, err := scanEmergencyExecutables([]sdk.Msg{msg})
	if err != nil {
		return false, err
	}
	return result.containsWrappedEmergency, nil
}

func containsEmergencyExecutableMessage(msgs []sdk.Msg) (bool, error) {
	result, err := scanEmergencyExecutables(msgs)
	if err != nil {
		return false, err
	}
	return result.containsEmergency, nil
}

func containsICAHostReceive(msgs []sdk.Msg) bool {
	result, err := scanEmergencyExecutables(msgs)
	return err != nil || result.containsICAHostReceive
}

func (ehd EmergencyHaltDecorator) isRecoveryGovernanceMessage(ctx sdk.Context, msg sdk.Msg) bool {
	switch typed := msg.(type) {
	case *govv1.MsgSubmitProposal:
		return ehd.recoveryProposalReader != nil &&
			ehd.recoveryProposalReader.IsAuthorizedRecoverySubmission(
				ctx,
				typed,
			)
	case *govv1.MsgVote:
		return ehd.isExpeditedRecoveryProposal(ctx, typed.ProposalId)
	case *govv1.MsgVoteWeighted:
		return ehd.isExpeditedRecoveryProposal(ctx, typed.ProposalId)
	case *govv1.MsgDeposit:
		return ehd.isExpeditedRecoveryProposal(ctx, typed.ProposalId)
	default:
		return false
	}
}

func (ehd EmergencyHaltDecorator) isExpeditedRecoveryProposal(ctx sdk.Context, proposalID uint64) bool {
	return ehd.recoveryProposalReader != nil &&
		ehd.recoveryProposalReader.IsExpeditedRecoveryProposal(ctx, proposalID)
}

// ---------- ZRNGasDecorator ----------

// ZRNGasDecorator validates that transactions provide sufficient gas based on
// the ZRN gas cost table (translated from core/billing/gas.ts).
type ZRNGasDecorator struct{}

// NewZRNGasDecorator creates a new ZRNGasDecorator.
func NewZRNGasDecorator() ZRNGasDecorator {
	return ZRNGasDecorator{}
}

func (zgd ZRNGasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "tx must implement FeeTx")
	}

	gasLimit := feeTx.GetGas()

	// Enforce block gas limit
	if gasLimit > BlockGasLimit {
		return ctx, errors.Wrapf(sdkerrors.ErrOutOfGas,
			"tx gas limit %d exceeds block gas limit %d", gasLimit, BlockGasLimit)
	}

	// Enforce per-tx gas limit
	if gasLimit > TxGasLimit {
		return ctx, errors.Wrapf(sdkerrors.ErrOutOfGas,
			"tx gas limit %d exceeds per-tx limit %d", gasLimit, TxGasLimit)
	}

	// Calculate minimum required gas from message types.
	// ANTE P1-2 FIX: Saturating addition to prevent uint64 overflow.
	msgs := tx.GetMsgs()
	var totalRequiredGas uint64
	for _, msg := range msgs {
		msgType := sdk.MsgTypeURL(msg)
		requiredGas := lookupMsgGas(msgType)
		totalRequiredGas += requiredGas
		if totalRequiredGas < requiredGas { // overflow detected
			return ctx, errors.Wrap(sdkerrors.ErrOutOfGas, "gas requirement overflow")
		}
	}

	if totalRequiredGas < MinGasLimit {
		totalRequiredGas = MinGasLimit
	}

	if gasLimit < totalRequiredGas {
		return ctx, errors.Wrapf(sdkerrors.ErrOutOfGas,
			"tx gas limit %d below minimum required %d for %d messages",
			gasLimit, totalRequiredGas, len(msgs))
	}

	// Enforce minimum gas price at CONSENSUS level (CheckTx AND DeliverTx),
	// for ALL transactions including zero-fee ones. The previous
	// `if !feeCoins.IsZero()` guard let zero-fee txs skip fee enforcement
	// entirely, leaving node-local minimum-gas-prices as the only spam gate
	// (design doc 2026-07-07 §10: "zero-fee consensus bypass closed
	// PRE-GENESIS"). Subsidized onboarding pays via x/feegrant: a feegranted
	// tx still declares a NON-zero fee (the granter pays it in
	// DeductFeeDecorator), so it passes this check unchanged.
	//
	// Sole exemption: height 0, i.e. InitChain genesis delivery. Gentxs are
	// zero-fee ceremony artifacts executed before the chain accepts user
	// transactions; CometBFT never delivers user txs at height 0, so this
	// is not reachable as a bypass on a live chain.
	if ctx.BlockHeight() > 0 {
		feeCoins := feeTx.GetFee()
		minFee := sdk.NewCoin(BondDenom, math.NewIntFromUint64(gasLimit*MinGasPrice))
		if feeCoins.AmountOf(BondDenom).LT(minFee.Amount) {
			return ctx, errors.Wrapf(sdkerrors.ErrInsufficientFee,
				"fee %s below minimum %s (gas %d * min price %d)",
				feeCoins, minFee, gasLimit, MinGasPrice)
		}
	}

	return next(ctx, tx, simulate)
}

// lookupMsgGas maps a Cosmos SDK message type URL to its ZRN gas cost.
// Type URLs are like "/zerone.knowledge.v1.MsgSubmitClaim".
func lookupMsgGas(msgTypeURL string) uint64 {
	if cost, ok := msgTypeURLToGas[msgTypeURL]; ok {
		return cost
	}
	// Unknown message types get minimum gas
	return MinGasLimit
}

// msgTypeURLToGas maps protobuf message type URLs to gas costs.
// Built from the TransactionGasCosts table and proto service definitions.
var msgTypeURLToGas = map[string]uint64{
	// Standard Cosmos messages
	"/cosmos.bank.v1beta1.MsgSend":               TransactionGasCosts["transfer"],
	"/cosmos.bank.v1beta1.MsgMultiSend":          TransactionGasCosts["transfer"] * 2,
	"/cosmos.staking.v1beta1.MsgDelegate":        TransactionGasCosts["delegate"],
	"/cosmos.staking.v1beta1.MsgUndelegate":      TransactionGasCosts["undelegate"],
	"/cosmos.staking.v1beta1.MsgBeginRedelegate": TransactionGasCosts["redelegate"],
	"/cosmos.gov.v1.MsgSubmitProposal":           TransactionGasCosts["governance_propose"],
	"/cosmos.gov.v1.MsgVote":                     TransactionGasCosts["governance_vote"],

	// IBC
	"/ibc.applications.transfer.v1.MsgTransfer": TransactionGasCosts["transfer"],

	// Knowledge module
	"/zerone.knowledge.v1.MsgSubmitClaim":             TransactionGasCosts["claim_submit"],
	"/zerone.knowledge.v1.MsgSubmitCommitment":        TransactionGasCosts["verification_commit"],
	"/zerone.knowledge.v1.MsgSubmitReveal":            TransactionGasCosts["verification_reveal"],
	"/zerone.knowledge.v1.MsgChallengeFact":           TransactionGasCosts["challenge_fact"],
	"/zerone.knowledge.v1.MsgAddFact":                 TransactionGasCosts["add_fact"],
	"/zerone.knowledge.v1.MsgSubmitContradiction":     TransactionGasCosts["submit_contradiction"],
	"/zerone.knowledge.v1.MsgPatronizeFact":           TransactionGasCosts["patronize_fact"],
	"/zerone.knowledge.v1.MsgProposeDomain":           TransactionGasCosts["propose_domain"],
	"/zerone.knowledge.v1.MsgEndorseDomainProposal":   TransactionGasCosts["endorse_domain"],
	"/zerone.knowledge.v1.MsgChallengeDomainProposal": TransactionGasCosts["challenge_domain"],
	"/zerone.knowledge.v1.MsgRegisterStratum":         TransactionGasCosts["register_stratum"],

	// Knowledge (extended — hand-written types)
	"/zerone.knowledge.v1.MsgChallengeProvisionalFact": TransactionGasCosts["challenge_provisional_fact"],
	"/zerone.knowledge.v1.MsgUpdateExtendedParams":     TransactionGasCosts["update_extended_params"],

	// Auth module (agent identity)
	"/zerone.auth.v1.MsgRegisterAccount": TransactionGasCosts["register_account"],
	"/zerone.auth.v1.MsgRotateKey":       TransactionGasCosts["rotate_key"],

	// Staking module (extended)
	"/zerone.staking.v1.MsgRegisterValidator":    TransactionGasCosts["register_validator"],
	"/zerone.staking.v1.MsgUpdateValidatorStake": TransactionGasCosts["update_validator_stake"],
	"/zerone.staking.v1.MsgDelegate":             TransactionGasCosts["zerone_delegate"],
	"/zerone.staking.v1.MsgUndelegate":           TransactionGasCosts["zerone_undelegate"],
	"/zerone.staking.v1.MsgRedelegate":           TransactionGasCosts["zerone_redelegate"],

	// Vesting rewards
	"/zerone.vesting_rewards.v1.MsgCreateVesting":     TransactionGasCosts["create_vesting"],
	"/zerone.vesting_rewards.v1.MsgClaimVesting":      TransactionGasCosts["claim_vesting"],
	"/zerone.vesting_rewards.v1.MsgPauseVesting":      TransactionGasCosts["pause_vesting"],
	"/zerone.vesting_rewards.v1.MsgResumeVesting":     TransactionGasCosts["resume_vesting"],
	"/zerone.vesting_rewards.v1.MsgAccelerateVesting": TransactionGasCosts["accelerate_vesting"],
	"/zerone.vesting_rewards.v1.MsgFalsifyVesting":    TransactionGasCosts["falsify_vesting"],
	"/zerone.vesting_rewards.v1.MsgCompleteVesting":   TransactionGasCosts["complete_vesting"],

	// Governance (extended)
	"/zerone.gov.v1.MsgSubmitLIP":           TransactionGasCosts["submit_lip"],
	"/zerone.gov.v1.MsgCastVote":            TransactionGasCosts["cast_vote"],
	"/zerone.gov.v1.MsgLockVote":            TransactionGasCosts["lock_vote"],
	"/zerone.gov.v1.MsgUnlockVote":          TransactionGasCosts["unlock_vote"],
	"/zerone.gov.v1.MsgCommitReview":        TransactionGasCosts["commit_review"],
	"/zerone.gov.v1.MsgRevealReview":        TransactionGasCosts["reveal_review"],
	"/zerone.gov.v1.MsgSubmitDisbursement":  TransactionGasCosts["submit_disbursement"],
	"/zerone.gov.v1.MsgExecuteDisbursement": TransactionGasCosts["execute_disbursement"],

	// Emergency
	"/zerone.emergency.v1.MsgProposeHalt":   TransactionGasCosts["propose_halt"],
	"/zerone.emergency.v1.MsgVoteHalt":      TransactionGasCosts["vote_halt"],
	"/zerone.emergency.v1.MsgProposeRevert": TransactionGasCosts["propose_revert"],
	"/zerone.emergency.v1.MsgVoteRevert":    TransactionGasCosts["vote_revert"],
	"/zerone.emergency.v1.MsgProposeResume": TransactionGasCosts["propose_resume"],
	"/zerone.emergency.v1.MsgVoteResume":    TransactionGasCosts["vote_resume"],

	// Claiming pots
	"/zerone.claiming_pot.v1.MsgCreatePot":    TransactionGasCosts["create_pot"],
	"/zerone.claiming_pot.v1.MsgFundPot":      TransactionGasCosts["fund_pot"],
	"/zerone.claiming_pot.v1.MsgClaimFromPot": TransactionGasCosts["claim_from_pot"],
	"/zerone.claiming_pot.v1.MsgClosePot":     TransactionGasCosts["close_pot"],

	// Capture defense
	"/zerone.capture_defense.v1.MsgRequestQualification": TransactionGasCosts["request_capture_qualification"],
	"/zerone.capture_defense.v1.MsgEndorseQualification": TransactionGasCosts["endorse_capture_qualification"],

	// Capture challenge
	"/zerone.capture_challenge.v1.MsgSubmitChallenge":  TransactionGasCosts["submit_capture_challenge"],
	"/zerone.capture_challenge.v1.MsgAddEvidence":      TransactionGasCosts["add_challenge_evidence"],
	"/zerone.capture_challenge.v1.MsgResolveChallenge": TransactionGasCosts["resolve_capture_challenge"],

	// Agent Home
	"/zerone.home.v1.MsgCreateHome":        TransactionGasCosts["create_home"],
	"/zerone.home.v1.MsgUpdateHome":        TransactionGasCosts["update_home"],
	"/zerone.home.v1.MsgUpdateMemoryCID":   TransactionGasCosts["update_memory_cid"],
	"/zerone.home.v1.MsgStartSession":      TransactionGasCosts["home_start_session"],
	"/zerone.home.v1.MsgEndSession":        TransactionGasCosts["home_end_session"],
	"/zerone.home.v1.MsgRegisterKey":       TransactionGasCosts["home_register_key"],
	"/zerone.home.v1.MsgRevokeKey":         TransactionGasCosts["home_revoke_key"],
	"/zerone.home.v1.MsgConfigureGuardian": TransactionGasCosts["configure_guardian"],
	"/zerone.home.v1.MsgAcknowledgeAlert":  TransactionGasCosts["acknowledge_alert"],
	"/zerone.home.v1.MsgSetSpendingLimit":  TransactionGasCosts["set_spending_limit"],

	// Qualification
	"/zerone.qualification.v1.MsgRequestQualification":  TransactionGasCosts["request_qualification"],
	"/zerone.qualification.v1.MsgEndorseValidator":      TransactionGasCosts["endorse_validator"],
	"/zerone.qualification.v1.MsgRenewQualification":    TransactionGasCosts["renew_qualification"],
	"/zerone.qualification.v1.MsgWithdrawQualification": TransactionGasCosts["withdraw_qualification"],

	// Auth account freeze controls
	"/zerone.auth.v1.MsgFreezeAccount":   TransactionGasCosts["freeze_account"],
	"/zerone.auth.v1.MsgUnfreezeAccount": TransactionGasCosts["unfreeze_account"],

	// Governance extras
	"/zerone.gov.v1.MsgAttachUpgradePlan":   TransactionGasCosts["attach_upgrade_plan"],
	"/zerone.gov.v1.MsgStakeLIP":            TransactionGasCosts["governance_stake_lip"],
	"/zerone.gov.v1.MsgAmendLIP":            TransactionGasCosts["amend_lip"],
	"/zerone.gov.v1.MsgAdvanceLIPStage":     TransactionGasCosts["advance_lip_stage"],
	"/zerone.gov.v1.MsgWithdrawLIP":         TransactionGasCosts["withdraw_lip"],
	"/zerone.gov.v1.MsgSwitchVote":          TransactionGasCosts["switch_vote"],
	"/zerone.gov.v1.MsgFinalizeReview":      TransactionGasCosts["finalize_review"],
	"/zerone.gov.v1.MsgStakeDisbursement":   TransactionGasCosts["stake_disbursement"],
	"/zerone.gov.v1.MsgAdvanceDisbursement": TransactionGasCosts["advance_disbursement"],
	"/zerone.gov.v1.MsgCancelDisbursement":  TransactionGasCosts["cancel_disbursement"],
	"/zerone.gov.v1.MsgRegisterOperator":    TransactionGasCosts["register_operator"],
	"/zerone.gov.v1.MsgAddAgent":            TransactionGasCosts["add_agent"],
	"/zerone.gov.v1.MsgRemoveAgent":         TransactionGasCosts["remove_agent"],
	"/zerone.gov.v1.MsgSlashOperator":       TransactionGasCosts["slash_operator"],
	"/zerone.gov.v1.MsgCreateDeployment":    TransactionGasCosts["create_deployment"],
	"/zerone.gov.v1.MsgAdvanceDeployment":   TransactionGasCosts["advance_deployment"],
	"/zerone.gov.v1.MsgApproveDeployment":   TransactionGasCosts["approve_deployment"],
	"/zerone.gov.v1.MsgRollbackDeployment":  TransactionGasCosts["rollback_deployment"],

	// Ontology
	"/zerone.ontology.v1.MsgProposeDomain":             TransactionGasCosts["propose_domain"],
	"/zerone.ontology.v1.MsgVoteDomainProposal":        TransactionGasCosts["vote_domain_proposal"],
	"/zerone.ontology.v1.MsgUpdateDomain":              TransactionGasCosts["update_domain"],
	"/zerone.ontology.v1.MsgRegisterLogicZone":         TransactionGasCosts["register_logic_zone"],
	"/zerone.ontology.v1.MsgAcknowledgeIncompleteness": TransactionGasCosts["acknowledge_incompleteness"],

	// Alignment
	"/zerone.alignment.v1.MsgActivate": TransactionGasCosts["activate_alignment"],

	// Liquidity pool
	"/zerone.liquiditypool.v1.MsgCreatePool":      TransactionGasCosts["create_pool"],
	"/zerone.liquiditypool.v1.MsgSwap":            TransactionGasCosts["lp_swap"],
	"/zerone.liquiditypool.v1.MsgAddLiquidity":    TransactionGasCosts["lp_add_liquidity"],
	"/zerone.liquiditypool.v1.MsgRemoveLiquidity": TransactionGasCosts["lp_remove_liquidity"],

	// IBC rate limiting
	"/zerone.ibcratelimit.v1.MsgAddRateLimit":    TransactionGasCosts["add_rate_limit"],
	"/zerone.ibcratelimit.v1.MsgRemoveRateLimit": TransactionGasCosts["remove_rate_limit"],

	// Governance research spending
	"/zerone.gov.v1.MsgSubmitResearchSpend": TransactionGasCosts["submit_research_spend"],
	"/zerone.gov.v1.MsgVoteResearchSpend":   TransactionGasCosts["vote_research_spend"],
	"/zerone.gov.v1.MsgSetResearchVoters":   TransactionGasCosts["set_research_voters"],
}

// ---------- Fee Router Decorator ----------

// FeeRouterDecorator observes fees whose actual routing is performed by
// vesting_rewards.RouteFees using the current on-chain RevenueSplit params.
type FeeRouterDecorator struct {
	bankKeeper bankkeeper.Keeper
}

// NewFeeRouterDecorator creates a new FeeRouterDecorator.
func NewFeeRouterDecorator(bk bankkeeper.Keeper) FeeRouterDecorator {
	return FeeRouterDecorator{bankKeeper: bk}
}

func (frd FeeRouterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Fee routing is handled in x/vesting_rewards BeginBlocker via
	// keeper.RouteFees(). This decorator does not hold that keeper and must not
	// report a hard-coded split that can drift from on-chain params.
	//
	// Log only the observed fee and the authoritative source of routing data.

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	fee := feeTx.GetFee()
	if fee.IsZero() {
		return next(ctx, tx, simulate)
	}

	ctx.Logger().Debug("ZRN fee routing",
		"total_fee", fee.String(),
		"split_source", "x/vesting_rewards revenue_split params",
	)

	return next(ctx, tx, simulate)
}

// GetMinimumFee calculates the minimum fee for a transaction based on message types.
func GetMinimumFee(msgs []sdk.Msg) sdk.Coins {
	var totalGas uint64
	for _, msg := range msgs {
		totalGas += lookupMsgGas(sdk.MsgTypeURL(msg))
	}
	if totalGas < MinGasLimit {
		totalGas = MinGasLimit
	}

	minFee := math.NewIntFromUint64(totalGas * MinGasPrice)
	return sdk.NewCoins(sdk.NewCoin(BondDenom, minFee))
}

// ---------- ZeroneDIDDecorator ----------

// ZeroneDIDDecorator validates DID references in transactions.
// If a tx memo contains "did:zrn:", it validates the DID resolves to the sender
// and emits indexing events.
//
// Runs AFTER signature verification (post-auth) to prevent unauthenticated
// state reads. DID validation is an additional constraint, not a prerequisite.
//
// ANTE P0-1 FIX: Uses tx-level signer extraction for DID-to-signer matching.
type ZeroneDIDDecorator struct {
	zak zeroneauthkeeper.Keeper
}

// NewZeroneDIDDecorator creates a new ZeroneDIDDecorator.
func NewZeroneDIDDecorator(zak zeroneauthkeeper.Keeper) ZeroneDIDDecorator {
	return ZeroneDIDDecorator{zak: zak}
}

func (zdd ZeroneDIDDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	// Check if tx has a memo with DID reference
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return next(ctx, tx, simulate)
	}

	memo := memoTx.GetMemo()
	if len(memo) < 8 || memo[:8] != "did:zrn:" {
		return next(ctx, tx, simulate)
	}

	// Extract DID from memo (format: "did:zrn:{32-64hex}")
	did := memo
	if len(did) > 72 { // "did:zrn:" + 64 hex = 72
		did = memo[:72]
	}

	if err := zeroneauthtypes.ValidateDID(did); err != nil {
		return ctx, zeroneauthtypes.ErrDIDResolutionFailed
	}

	// Verify DID resolves to a known address
	address, found := zdd.zak.GetAddressForDID(ctx, did)
	if !found {
		return ctx, zeroneauthtypes.ErrDIDResolutionFailed
	}

	// Validate that the DID resolves to one of the tx signers.
	signers, err := getAuthenticatedSignerAddresses(tx)
	if err != nil {
		return ctx, err
	}
	senderMatch := false
	for _, signer := range signers {
		if signer.String() == address {
			senderMatch = true
			break
		}
	}

	if !senderMatch {
		return ctx, zeroneauthtypes.ErrDIDResolutionFailed
	}

	// Emit DID event for indexing
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"did_reference",
			sdk.NewAttribute("did", did),
			sdk.NewAttribute("address", address),
		),
	)

	return next(ctx, tx, simulate)
}

// ---------- ZeroneFeeGranterDecorator ----------

// ZeroneFeeGranterDecorator applies the account freeze invariant to the
// account paying through x/feegrant. A fee granter is not necessarily a
// transaction signer, so the signer-only ZeroneAccountDecorator cannot enforce
// this boundary. It must run before the SDK DeductFeeDecorator consumes an
// allowance or transfers the granter's fee.
type ZeroneFeeGranterDecorator struct {
	zak zeroneauthkeeper.Keeper
}

// NewZeroneFeeGranterDecorator creates a fee-granter freeze guard.
func NewZeroneFeeGranterDecorator(zak zeroneauthkeeper.Keeper) ZeroneFeeGranterDecorator {
	return ZeroneFeeGranterDecorator{zak: zak}
}

func (zfg ZeroneFeeGranterDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}
	granterBytes := feeTx.FeeGranter()
	if len(granterBytes) == 0 {
		return next(ctx, tx, simulate)
	}
	if len(granterBytes) != 20 {
		return ctx, errors.Wrapf(
			sdkerrors.ErrInvalidAddress,
			"fee granter address must be 20 bytes, got %d",
			len(granterBytes),
		)
	}
	if err := sdk.VerifyAddressFormat(granterBytes); err != nil {
		return ctx, errors.Wrapf(
			sdkerrors.ErrInvalidAddress,
			"invalid fee granter address: %v",
			err,
		)
	}

	granter := sdk.AccAddress(granterBytes).String()
	account, found := zfg.zak.GetAccount(ctx, granter)
	if found && account.Flags != nil && account.Flags.Frozen {
		return ctx, zeroneauthtypes.ErrAccountFrozen
	}
	return next(ctx, tx, simulate)
}

// ---------- ZeroneAccountDecorator ----------

// ZeroneAccountDecorator enforces Zerone-specific account constraints:
// 1. Frozen accounts cannot send transactions
// 2. Updates LastActiveBlock for registered Zerone accounts
//
// Runs AFTER signature verification (signer is already authenticated).
//
// ANTE P0-1 FIX: Uses tx-level signer extraction (getSignerAddresses) instead of
// per-message type assertion. SDK v0.50 proto-generated types (MsgSend, MsgDelegate)
// don't implement GetSigners() []sdk.AccAddress, so the old approach silently skipped
// frozen account checks for all standard Cosmos SDK messages.
type ZeroneAccountDecorator struct {
	zak zeroneauthkeeper.Keeper
}

// NewZeroneAccountDecorator creates a new ZeroneAccountDecorator.
func NewZeroneAccountDecorator(zak zeroneauthkeeper.Keeper) ZeroneAccountDecorator {
	return ZeroneAccountDecorator{zak: zak}
}

func (zad ZeroneAccountDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	currentHeight := uint64(ctx.BlockHeight())

	// Extract signers from tx signature data (works for ALL message types).
	signers, err := getAuthenticatedSignerAddresses(tx)
	if err != nil {
		return ctx, err
	}
	for _, signer := range signers {
		address := signer.String()

		account, found := zad.zak.GetAccount(ctx, address)
		if !found {
			// Not a registered Zerone account — standard Cosmos account, skip
			continue
		}

		// Check frozen status
		if account.Flags != nil && account.Flags.Frozen {
			return ctx, zeroneauthtypes.ErrAccountFrozen
		}

		// Update last active block.
		if account.LastActiveBlock < currentHeight {
			account.LastActiveBlock = currentHeight
			zad.zak.SetAccount(ctx, account)
		}
	}

	return next(ctx, tx, simulate)
}

// ---------- ZeroneCapabilityDecorator ----------

// ZeroneCapabilityDecorator enforces account-level capabilities for every
// signer of a tx. The session-key path (delegated ephemeral keys with
// default-deny capabilities) was removed in the 2026-07 slim cut: delegated
// agent authority is an agenttool-platform concern; the chain keeps the
// account-flag gates (claims/challenges) on the identity anchor.
//
// Runs AFTER signature verification and ZeroneAccountDecorator.
//
// Signers come from SigVerifiableTx.GetSigners after standard signature
// verification. This covers SDK-generated messages and remains correct when a
// transaction omits an already-stored public key from SignerInfo.
type ZeroneCapabilityDecorator struct {
	zak zeroneauthkeeper.Keeper
}

// NewZeroneCapabilityDecorator creates a new ZeroneCapabilityDecorator.
func NewZeroneCapabilityDecorator(zak zeroneauthkeeper.Keeper) ZeroneCapabilityDecorator {
	return ZeroneCapabilityDecorator{zak: zak}
}

func (zcd ZeroneCapabilityDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()

	signers, err := getAuthenticatedSignerAddresses(tx)
	if err != nil {
		return ctx, err
	}
	for _, signer := range signers {
		address := signer.String()

		// Enforce account-level capabilities (default-allow for unrecognized types).
		for _, msg := range msgs {
			if err := zcd.checkAccountCapability(ctx, address, msg); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

// checkAccountCapability enforces account-level capabilities for primary key signers.
// Unlike session keys (default-deny), primary keys use default-allow for unrecognized types.
func (zcd ZeroneCapabilityDecorator) checkAccountCapability(ctx sdk.Context, address string, msg sdk.Msg) error {
	msgType := sdk.MsgTypeURL(msg)

	// Auth management messages are always allowed (registration, key rotation, etc.)
	if isAuthManagementMsg(msgType) {
		return nil
	}

	account, found := zcd.zak.GetAccount(ctx, address)
	if !found {
		// Unregistered account: block Zerone-specific ops, allow everything else
		if isZeroneSpecificMsg(msgType) {
			return errors.Wrapf(zeroneauthtypes.ErrAccountCapabilityDenied,
				"%s is not registered with zerone_auth — one-shot fix: zeroned tx zerone_auth onboard agent --from <your-key>", address)
		}
		return nil
	}

	return zcd.checkRegisteredAccountCapability(account, msgType)
}

// checkRegisteredAccountCapability enforces capabilities for registered accounts.
// Flag-based capabilities (CanSubmitClaims, CanChallenge) are checked from AccountFlags.
// Type-based restrictions (staking, voting) are derived from account_type.
func (zcd ZeroneCapabilityDecorator) checkRegisteredAccountCapability(account *zeroneauthtypes.Account, msgType string) error {
	flags := account.Flags
	accountType := account.AccountType

	switch {
	case isClaimSubmissionMsg(msgType):
		if flags == nil || !flags.CanSubmitClaims {
			return errors.Wrapf(zeroneauthtypes.ErrAccountCapabilityDenied,
				"account type %q cannot submit claims or witness (CanSubmitClaims=false; 'agent' and 'human' accounts can)", accountType)
		}
		return nil
	case isChallengeMsg(msgType):
		if flags == nil || !flags.CanChallenge {
			return errors.Wrapf(zeroneauthtypes.ErrAccountCapabilityDenied,
				"account type %q cannot challenge facts (CanChallenge=false)", accountType)
		}
		return nil
	case isStakeMsg(msgType):
		if accountType == "contract" {
			return zeroneauthtypes.ErrAccountCapabilityDenied
		}
		return nil
	case isVoteMsg(msgType):
		if accountType == "contract" {
			return zeroneauthtypes.ErrAccountCapabilityDenied
		}
		return nil
	case isTransferMsg(msgType):
		// Allowed for all registered account types
		return nil
	default:
		// Default-allow for primary keys (preserves authz/fee grants/etc.)
		return nil
	}
}

func isTransferMsg(msgType string) bool {
	return msgType == "/cosmos.bank.v1beta1.MsgSend" ||
		msgType == "/cosmos.bank.v1beta1.MsgMultiSend" ||
		msgType == "/ibc.applications.transfer.v1.MsgTransfer"
}

func isStakeMsg(msgType string) bool {
	return msgType == "/cosmos.staking.v1beta1.MsgDelegate" ||
		msgType == "/cosmos.staking.v1beta1.MsgUndelegate" ||
		msgType == "/cosmos.staking.v1beta1.MsgBeginRedelegate" ||
		msgType == "/zerone.staking.v1.MsgRegisterValidator"
}

func isClaimSubmissionMsg(msgType string) bool {
	return msgType == "/zerone.knowledge.v1.MsgSubmitClaim" ||
		msgType == "/zerone.knowledge.v1.MsgSubmitCommitment" ||
		msgType == "/zerone.knowledge.v1.MsgSubmitReveal"
}

func isChallengeMsg(msgType string) bool {
	return msgType == "/zerone.knowledge.v1.MsgChallengeFact"
}

func isClaimMsg(msgType string) bool {
	return isClaimSubmissionMsg(msgType) || isChallengeMsg(msgType)
}

// isAuthManagementMsg checks if a message is an auth management operation.
// These are always allowed — accounts must be able to register and manage keys.
func isAuthManagementMsg(msgType string) bool {
	return msgType == "/zerone.auth.v1.MsgRegisterAccount" ||
		msgType == "/zerone.auth.v1.MsgRotateKey" ||
		msgType == "/zerone.auth.v1.MsgFreezeAccount" ||
		msgType == "/zerone.auth.v1.MsgUnfreezeAccount"
}

// isZeroneSpecificMsg checks if a message is a Zerone-specific operation
// that requires registration. Used for unregistered account gating.
func isZeroneSpecificMsg(msgType string) bool {
	return isClaimSubmissionMsg(msgType) ||
		isChallengeMsg(msgType)
}

func isVoteMsg(msgType string) bool {
	return msgType == "/cosmos.gov.v1.MsgVote" ||
		msgType == "/zerone.gov.v1.MsgCastVote" ||
		msgType == "/zerone.emergency.v1.MsgVoteHalt" ||
		msgType == "/zerone.emergency.v1.MsgVoteRevert" ||
		msgType == "/zerone.emergency.v1.MsgVoteResume"
}

// ---------- Helpers ----------

func init() {
	// Validate that all gas costs are within limits at startup
	for txType, gas := range TransactionGasCosts {
		if gas > TxGasLimit {
			panic(fmt.Sprintf("gas cost for %s (%d) exceeds tx limit (%d)", txType, gas, TxGasLimit))
		}
	}
}
