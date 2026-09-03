// ABCI++ Extension Points for Proof of Truth Consensus
//
// This file implements the ABCI++ methods that integrate PoT's multi-block
// verification rounds into CometBFT's single-block cycle.
//
// Vote Extension Data Flow:
//  1. ExtendVote: Validators attach commitments/reveals to CometBFT votes
//  2. PrepareProposal: Proposer collects extensions, creates injection tx
//  3. ProcessProposal: Validators verify injection tx format
//  4. PreBlocker: All validators process injection tx, store in keeper state
//  5. BeginBlocker: Phase transitions and aggregation via knowledge module
package app

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// VoteExtension is the PoT-specific data validators attach to their votes.
// Serialized as JSON for Phase 2; future phases may use protobuf.
type VoteExtension struct {
	Commitments      []VoteCommitment `json:"commitments,omitempty"`
	Reveals          []VoteReveal     `json:"reveals,omitempty"`
	ValidatorAddress string           `json:"validator_address"`
}

// VoteCommitment represents a validator's commitment to a verification verdict.
type VoteCommitment struct {
	RoundID        string `json:"round_id"`
	CommitmentHash string `json:"commitment_hash"`      // hex-encoded SHA-256
	VRFOutput      string `json:"vrf_output,omitempty"` // hex-encoded VRF output
	VRFProof       string `json:"vrf_proof,omitempty"`  // hex-encoded VRF proof
	Height         uint64 `json:"height"`
}

// VoteReveal represents a validator's revealed verification verdict.
type VoteReveal struct {
	RoundID    string `json:"round_id"`
	Verdict    string `json:"verdict"`    // "accept", "reject", "abstain"
	Confidence uint64 `json:"confidence"` // 0-1000000
	Salt       string `json:"salt"`       // hex-encoded salt
}

// ---- Vote Extension Injection ----

// VoteExtInjectionPrefix is the 4-byte magic header for injected vote extension txs.
// These are pseudo-txs included at position 0 of a block by the proposer.
var VoteExtInjectionPrefix = []byte{0x00, 'V', 'E', 'X'}

// VoteExtInjection contains all commitments and reveals from the previous block's
// vote extensions. The proposer encodes this in PrepareProposal; all validators
// process it in PreBlocker to store data in the knowledge keeper.
type VoteExtInjection struct {
	Commitments []InjectedCommitment `json:"commitments"`
	Reveals     []InjectedReveal     `json:"reveals"`
}

// InjectedCommitment is a commitment extracted from a vote extension.
type InjectedCommitment struct {
	RoundID        string `json:"round_id"`
	Validator      string `json:"validator"`
	CommitmentHash string `json:"commitment_hash"` // hex-encoded
	VRFOutput      string `json:"vrf_output,omitempty"`
	VRFProof       string `json:"vrf_proof,omitempty"`
}

// InjectedReveal is a reveal extracted from a vote extension.
type InjectedReveal struct {
	RoundID    string `json:"round_id"`
	Validator  string `json:"validator"`
	Verdict    string `json:"verdict"`
	Confidence uint64 `json:"confidence"`
	Salt       string `json:"salt"` // hex-encoded
}

// MaxVEXInjectionBytes is the maximum size of a vote extension injection pseudo-tx.
const MaxVEXInjectionBytes = 2 * 1024 * 1024

// BlockGasLimit is defined in gas.go alongside other gas constants.

// IsVoteExtInjectionTx checks if a tx has the vote extension injection prefix.
func IsVoteExtInjectionTx(tx []byte) bool {
	return len(tx) > len(VoteExtInjectionPrefix) &&
		bytes.Equal(tx[:len(VoteExtInjectionPrefix)], VoteExtInjectionPrefix)
}

// EncodeVoteExtInjection serializes a VoteExtInjection with the magic prefix.
func EncodeVoteExtInjection(inj VoteExtInjection) ([]byte, error) {
	data, err := json.Marshal(inj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vote extension injection: %w", err)
	}
	return append(VoteExtInjectionPrefix, data...), nil
}

// DecodeVoteExtInjection deserializes a VoteExtInjection from prefixed bytes.
func DecodeVoteExtInjection(tx []byte) (VoteExtInjection, error) {
	var inj VoteExtInjection
	if !IsVoteExtInjectionTx(tx) {
		return inj, fmt.Errorf("not a vote extension injection tx")
	}
	err := json.Unmarshal(tx[len(VoteExtInjectionPrefix):], &inj)
	return inj, err
}

// ComputeCommitmentHash generates a commitment hash for a verification verdict.
// Delegates to knowledgetypes.ComputeCommitmentHash (canonical implementation).
// Returns hex-encoded hash for use in JSON transport types.
func ComputeCommitmentHash(roundID, verdict string, confidence uint64, salt string) string {
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(knowledgetypes.ComputeCommitmentHash(roundID, verdict, confidence, saltBytes))
}

// VerifyCommitmentHash checks that a reveal matches its prior commitment.
// Delegates to knowledgetypes.VerifyCommitmentHash (canonical implementation).
func VerifyCommitmentHash(commitmentHash, roundID, verdict string, confidence uint64, salt string) bool {
	hashBytes, err := hex.DecodeString(commitmentHash)
	if err != nil {
		return false
	}
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return false
	}
	return knowledgetypes.VerifyCommitmentHash(hashBytes, roundID, verdict, confidence, saltBytes)
}

// ---- ABCI++ Handlers ----

// PrepareProposalHandler returns a PrepareProposal handler that injects
// vote extension data as a pseudo-tx at position 0.
func (app *ZeroneApp) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return app.prepareProposal
}

func (app *ZeroneApp) prepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (resp *abci.ResponsePrepareProposal, err error) {
	if req == nil {
		return &abci.ResponsePrepareProposal{Txs: nil}, nil
	}

	logger := ctx.Logger().With("module", "abci", "handler", "PrepareProposal")

	defer func() {
		if r := recover(); r != nil {
			logger.Error("PANIC in PrepareProposal — returning empty proposal",
				"height", req.Height, "panic", fmt.Sprintf("%v", r))
			resp = &abci.ResponsePrepareProposal{Txs: nil}
			err = nil
		}
	}()

	maxProposalBytes := effectivePrepareProposalMaxBytes(ctx, req.MaxTxBytes)
	maxProposalGas := effectiveProposalMaxGas(ctx)
	selectedTxs := make([][]byte, 0, MaxProposalTxCount)

	// Process vote extensions from previous block (if available)
	if potVoteExtensionsReleaseEnabled && len(req.LocalLastCommit.Votes) > 0 {
		injection := app.processVoteExtensions(ctx, req.LocalLastCommit.Votes)

		if len(injection.Commitments) > 0 || len(injection.Reveals) > 0 {
			injBytes, err := EncodeVoteExtInjection(injection)
			if err != nil {
				// BaseApp falls back to the unfiltered RequestPrepareProposal
				// transactions when a custom handler returns an error. Keep this
				// failure closed by dropping the optional pseudo-transaction.
				logger.Error("vote extension injection encoding failed — dropping", "err", err)
			} else if len(injBytes) > MaxVEXInjectionBytes {
				logger.Warn("vote extension injection exceeds size limit — dropping",
					"size", len(injBytes), "max", MaxVEXInjectionBytes)
			} else if !proposalFitsByteLimit(appendProposalTx(selectedTxs, injBytes), maxProposalBytes) {
				logger.Warn("vote extension injection exceeds proposal byte limit — dropping",
					"size", proposalProtoSize([][]byte{injBytes}), "max", maxProposalBytes)
			} else {
				selectedTxs = append(selectedTxs, injBytes)
				logger.Info("injecting vote extension data",
					"commitments", len(injection.Commitments),
					"reveals", len(injection.Reveals),
				)
			}
		}
	}

	regularByteBudget := remainingProposalBytes(maxProposalBytes, selectedTxs)
	regularTxs := app.selectRegularProposalTxs(ctx, req, regularByteBudget, maxProposalGas)
	remainingBytes := regularByteBudget
	for _, txBytes := range regularTxs {
		// A VEX pseudo-transaction is application-created only. It must never be
		// admitted from either CometBFT's pool or the SDK application mempool.
		if IsVoteExtInjectionTx(txBytes) {
			logger.Warn("dropping vote extension pseudo-transaction from regular mempool selection")
			break
		}

		txSize := proposalTxProtoSize(txBytes)
		if remainingBytes >= 0 && txSize > remainingBytes {
			// DefaultProposalHandler already accounts for protobuf framing. Stop
			// instead of skipping so a same-signer sequence cannot develop a gap.
			logger.Warn("stopping proposal selection at byte limit",
				"tx_size", txSize, "remaining", remainingBytes, "max", maxProposalBytes)
			break
		}
		selectedTxs = append(selectedTxs, txBytes)
		if remainingBytes >= 0 {
			remainingBytes -= txSize
		}
	}

	logger.Debug("prepared proposal",
		"height", req.Height,
		"txs", len(selectedTxs),
		"bytes", proposalProtoSize(selectedTxs),
		"max_bytes", maxProposalBytes,
		"max_gas", maxProposalGas,
		"vote_extensions", len(req.LocalLastCommit.Votes),
	)

	return &abci.ResponsePrepareProposal{Txs: selectedTxs}, nil
}

// ProcessProposalHandler returns a ProcessProposal handler that validates
// vote extension injection txs and regular transactions.
func (app *ZeroneApp) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return app.processProposal
}

func (app *ZeroneApp) processProposal(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
	if req == nil {
		return rejectProposal(), nil
	}

	logger := ctx.Logger().With("module", "abci", "handler", "ProcessProposal")

	defer func() {
		if r := recover(); r != nil {
			logger.Error("PANIC in ProcessProposal — rejecting proposal",
				"height", req.Height, "panic", fmt.Sprintf("%v", r))
			resp = &abci.ResponseProcessProposal{
				Status: abci.ResponseProcessProposal_REJECT,
			}
			err = nil
		}
	}()

	// Every ordinary transaction must declare at least MinGasLimit and aggregate
	// gas is source-capped. Reject impossible cardinality before a hostile
	// proposal can induce millions of scans or a duplicate framing allocation
	// with zero-length protobuf fields.
	if len(req.Txs) > MaxProposalTxCount {
		logger.Warn("proposal exceeds immutable transaction-count limit",
			"txs", len(req.Txs), "max", MaxProposalTxCount)
		return rejectProposal(), nil
	}

	if err := validateVoteExtInjectionLayout(req.Txs); err != nil {
		logger.Warn("invalid vote extension injection layout", "err", err)
		return rejectProposal(), nil
	}

	maxProposalBytes := effectiveProcessProposalMaxBytes(ctx)
	proposalBytes := proposalProtoSize(req.Txs)
	if !proposalFitsByteLimit(req.Txs, maxProposalBytes) {
		logger.Warn("proposal exceeds consensus byte limit",
			"size", proposalBytes, "max", maxProposalBytes)
		return rejectProposal(), nil
	}

	maxProposalGas := effectiveProposalMaxGas(ctx)
	var totalGas uint64
	for i, txBytes := range req.Txs {
		if IsVoteExtInjectionTx(txBytes) {
			if !potVoteExtensionsReleaseEnabled {
				logger.Warn("vote extension injection rejected: release safety latch is closed",
					"index", i)
				return rejectProposal(), nil
			}
			if len(txBytes) > MaxVEXInjectionBytes {
				logger.Warn("vote extension injection tx exceeds size limit",
					"index", i, "size", len(txBytes), "max", MaxVEXInjectionBytes)
				return rejectProposal(), nil
			}
			if _, err := DecodeVoteExtInjection(txBytes); err != nil {
				logger.Warn("invalid vote extension injection tx", "index", i, "err", err)
				return rejectProposal(), nil
			}
			continue
		}

		tx, err := app.TxDecode(txBytes)
		if err != nil {
			logger.Warn("invalid tx in proposal", "index", i, "err", err)
			return rejectProposal(), nil
		}

		gasTx, ok := tx.(baseapp.GasTx)
		if !ok {
			logger.Warn("proposal tx does not expose a declared gas limit", "index", i)
			return rejectProposal(), nil
		}
		if gasTx.GetGas() < MinGasLimit {
			logger.Warn("proposal tx declares less than the consensus minimum gas",
				"index", i, "tx_gas", gasTx.GetGas(), "minimum", MinGasLimit)
			return rejectProposal(), nil
		}
		if gasWouldExceedLimit(totalGas, gasTx.GetGas(), maxProposalGas) {
			logger.Warn("proposal exceeds aggregate gas limit",
				"index", i,
				"gas_before", totalGas,
				"tx_gas", gasTx.GetGas(),
				"max", maxProposalGas,
			)
			return rejectProposal(), nil
		}
		totalGas += gasTx.GetGas()
	}

	// Decode-only or ValidateBasic-only checks do not authenticate signatures,
	// fees, sequences, timeouts, emergency quarantine, or Zerone capability
	// policy. Run the same BaseApp Ante path used during transaction execution.
	// Calls are sequential so account-sequence transitions inside a proposal are
	// checked against one another.
	for i, txBytes := range req.Txs {
		if IsVoteExtInjectionTx(txBytes) {
			continue
		}
		if _, err := app.ProcessProposalVerifyTx(txBytes); err != nil {
			logger.Warn("proposal tx failed ante verification", "index", i, "err", err)
			return rejectProposal(), nil
		}
	}

	logger.Debug("accepted proposal",
		"height", req.Height,
		"txs", len(req.Txs),
	)

	return &abci.ResponseProcessProposal{
		Status: abci.ResponseProcessProposal_ACCEPT,
	}, nil
}

// selectRegularProposalTxs delegates production selection to the SDK's
// application mempool handler. Tests and operators may explicitly configure a
// NoOpMempool; in that mode CometBFT supplies FIFO candidates and we still run
// full Ante verification instead of inheriting the SDK no-op verification path.
func (app *ZeroneApp) selectRegularProposalTxs(
	ctx sdk.Context,
	req *abci.RequestPrepareProposal,
	maxBytes int64,
	maxGas uint64,
) [][]byte {
	if maxBytes == 0 {
		return nil
	}

	regularReq := *req
	regularReq.MaxTxBytes = maxBytes
	regularReq.Txs = withoutVoteExtInjectionTxs(req.Txs)

	if isNoOpMempool(app.Mempool()) {
		return app.selectFIFORequestTxs(ctx, regularReq.Txs, maxBytes, maxGas)
	}

	return app.selectApplicationMempoolTxs(ctx, &regularReq, maxBytes, maxGas)
}

// selectApplicationMempoolTxs is a bounded adaptation of the SDK v0.53
// SenderNonce proposal handler. The upstream handler verifies a candidate
// before asking its selector whether capacity is exhausted, so a full stale
// pool can overrun the two-second proposal window. This version counts every
// inspected entry, pre-checks exact framing and declared gas, preserves all
// signer sequences, and removes ante-invalid entries only after SelectBy drops
// the SenderNonce mutex.
func (app *ZeroneApp) selectApplicationMempoolTxs(
	ctx sdk.Context,
	req *abci.RequestPrepareProposal,
	maxBytes int64,
	maxGas uint64,
) [][]byte {
	pool := app.Mempool()
	signerAdapter := newMessageSignerExtractionAdapter()
	selectedSignerSequences := make(map[string]uint64)
	blockedSigners := make(map[string]struct{})
	selected := make([][]byte, 0)
	invalid := make([]sdk.Tx, 0)
	var totalBytes int64
	var totalGas uint64
	inspected := 0
	inspectionLimitReached := false

	sdkmempool.SelectBy(ctx, pool, req.Txs, func(memTx sdk.Tx) bool {
		if inspected >= MaxProposalCandidateInspections {
			inspectionLimitReached = true
			return false
		}
		if maxGas-totalGas < MinGasLimit {
			return false
		}
		inspected++

		unorderedTx, unordered := memTx.(sdk.TxWithUnordered)
		isUnordered := unordered && unorderedTx.GetUnordered()
		signerSequences := make(map[string]uint64)
		signerAddresses := make(map[string]sdk.AccAddress)
		if !isUnordered {
			signers, err := signerAdapter.GetSigners(memTx)
			if err != nil {
				invalid = append(invalid, memTx)
				return true
			}
			for _, signer := range signers {
				key := signer.Signer.String()
				if _, blocked := blockedSigners[key]; blocked {
					return true
				}
				signerSequences[key] = signer.Sequence
				signerAddresses[key] = signer.Signer
			}
		}
		blockCandidateSigners := func() {
			for signer := range signerSequences {
				blockedSigners[signer] = struct{}{}
			}
		}
		if !isUnordered {
			invalidSequence := false
			deferredSequence := false
			for key, sequence := range signerSequences {
				if prior, seen := selectedSignerSequences[key]; seen {
					if prior == ^uint64(0) || sequence != prior+1 {
						deferredSequence = true
					}
					continue
				}

				account := app.AccountKeeper.GetAccount(ctx, signerAddresses[key])
				if account == nil || sequence < account.GetSequence() {
					invalidSequence = true
				}
				if account != nil && sequence > account.GetSequence() {
					// A predecessor may have left only the local application
					// pool (for example after proposer-side eviction) while it
					// is still in flight through CometBFT. Preserve this
					// transaction so it can become valid after that nonce is
					// committed or reintroduced.
					deferredSequence = true
				}
			}
			if invalidSequence {
				invalid = append(invalid, memTx)
				blockCandidateSigners()
				return true
			}
			if deferredSequence {
				// PriorityNonce orders only by a transaction's first signer. A
				// higher-priority multi-signer transaction can therefore appear
				// before the lower-priority transaction that advances one of its
				// secondary signers. Preserve the future transaction without
				// blocking those signers so that predecessor can still be selected
				// later in this pass; the future transaction can follow after that
				// predecessor commits.
				return true
			}
		}

		gasTx, ok := memTx.(baseapp.GasTx)
		if !ok || gasTx.GetGas() < MinGasLimit {
			invalid = append(invalid, memTx)
			blockCandidateSigners()
			return true
		}
		if gasWouldExceedLimit(totalGas, gasTx.GetGas(), maxGas) {
			blockCandidateSigners()
			return true
		}

		encoded, err := app.TxEncode(memTx)
		if err != nil {
			invalid = append(invalid, memTx)
			blockCandidateSigners()
			return true
		}
		txSize := proposalTxProtoSize(encoded)
		if maxBytes >= 0 && (totalBytes > maxBytes || txSize > maxBytes-totalBytes) {
			blockCandidateSigners()
			return true
		}

		verified, err := app.PrepareProposalVerifyTx(memTx)
		if err != nil {
			invalid = append(invalid, memTx)
			blockCandidateSigners()
			return true
		}
		if !bytes.Equal(encoded, verified) {
			invalid = append(invalid, memTx)
			blockCandidateSigners()
			return true
		}

		selected = append(selected, verified)
		totalBytes += txSize
		totalGas += gasTx.GetGas()
		for signer, sequence := range signerSequences {
			selectedSignerSequences[signer] = sequence
		}
		return true
	})
	if inspectionLimitReached {
		ctx.Logger().Warn(
			"application-mempool proposal selection reached candidate inspection limit",
			"inspected", inspected,
			"limit", MaxProposalCandidateInspections,
			"selected", len(selected),
		)
	}

	for _, tx := range invalid {
		if err := pool.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
			ctx.Logger().Warn("failed to evict invalid application-mempool transaction", "err", err)
		}
	}
	return selected
}

// selectFIFORequestTxs is the strict fallback for an explicitly configured
// NoOpMempool. It preserves CometBFT FIFO order while dropping malformed or
// stale candidates. Capacity is checked before Ante so an unselected
// transaction cannot advance proposal-local account sequence state.
func (app *ZeroneApp) selectFIFORequestTxs(
	ctx sdk.Context,
	candidates [][]byte,
	maxBytes int64,
	maxGas uint64,
) [][]byte {
	selected := make([][]byte, 0, len(candidates))
	var totalBytes int64
	var totalGas uint64

	for index, txBytes := range candidates {
		if index >= MaxProposalCandidateInspections {
			ctx.Logger().Warn(
				"CometBFT FIFO proposal selection reached candidate inspection limit",
				"inspected", index,
				"limit", MaxProposalCandidateInspections,
				"selected", len(selected),
			)
			break
		}
		if IsVoteExtInjectionTx(txBytes) {
			continue
		}

		tx, err := app.TxDecode(txBytes)
		if err != nil {
			ctx.Logger().Debug("dropping undecodable CometBFT mempool candidate", "index", index, "err", err)
			continue
		}
		gasTx, ok := tx.(baseapp.GasTx)
		if !ok || gasTx.GetGas() < MinGasLimit || gasWouldExceedLimit(totalGas, gasTx.GetGas(), maxGas) {
			continue
		}

		canonicalBytes, err := app.TxEncode(tx)
		if err != nil {
			ctx.Logger().Debug("dropping unencodable CometBFT mempool candidate", "index", index, "err", err)
			continue
		}
		txSize := proposalTxProtoSize(canonicalBytes)
		if maxBytes >= 0 && (totalBytes > maxBytes || txSize > maxBytes-totalBytes) {
			continue
		}

		verifiedBytes, err := app.PrepareProposalVerifyTx(tx)
		if err != nil {
			ctx.Logger().Debug("dropping CometBFT mempool candidate that failed ante verification", "index", index, "err", err)
			continue
		}
		if !bytes.Equal(canonicalBytes, verifiedBytes) {
			// A verifier that rewrites bytes after the capacity decision could
			// invalidate signatures or create a sequence gap. Fail this candidate
			// closed; BaseApp's encoder is expected to be stable.
			break
		}

		selected = append(selected, verifiedBytes)
		totalBytes += txSize
		totalGas += gasTx.GetGas()
	}

	return selected
}

func isNoOpMempool(pool sdkmempool.Mempool) bool {
	if pool == nil {
		return true
	}
	switch pool.(type) {
	case sdkmempool.NoOpMempool, *sdkmempool.NoOpMempool:
		return true
	default:
		return false
	}
}

func effectiveProposalMaxGas(ctx sdk.Context) uint64 {
	maxGas := BlockGasLimit
	// Comet uses -1 for unlimited. Zero is a real zero-gas consensus limit and
	// must not silently expand to the application's source-level ceiling.
	if block := ctx.ConsensusParams().Block; block != nil && block.MaxGas >= 0 {
		consensusMaxGas := uint64(block.MaxGas)
		if consensusMaxGas < maxGas {
			maxGas = consensusMaxGas
		}
	}
	return maxGas
}

func effectivePrepareProposalMaxBytes(ctx sdk.Context, requestMax int64) int64 {
	maxBytes := BlockMaxBytesLimit
	if requestMax >= 0 && requestMax < maxBytes {
		maxBytes = requestMax
	}
	if block := ctx.ConsensusParams().Block; block != nil && block.MaxBytes >= 0 &&
		block.MaxBytes < maxBytes {
		maxBytes = block.MaxBytes
	}
	return maxBytes
}

func effectiveProcessProposalMaxBytes(ctx sdk.Context) int64 {
	if block := ctx.ConsensusParams().Block; block != nil && block.MaxBytes >= 0 && block.MaxBytes < BlockMaxBytesLimit {
		return block.MaxBytes
	}
	return BlockMaxBytesLimit
}

func remainingProposalBytes(maxBytes int64, selected [][]byte) int64 {
	if maxBytes < 0 {
		return -1
	}
	used := proposalProtoSize(selected)
	if used >= maxBytes {
		return 0
	}
	return maxBytes - used
}

func proposalProtoSize(txs [][]byte) int64 {
	cometTxs := make(cmttypes.Txs, len(txs))
	for i, tx := range txs {
		cometTxs[i] = cmttypes.Tx(tx)
	}
	return cmttypes.ComputeProtoSizeForTxs(cometTxs)
}

func proposalTxProtoSize(tx []byte) int64 {
	return cmttypes.ComputeProtoSizeForTxs(cmttypes.Txs{cmttypes.Tx(tx)})
}

func proposalFitsByteLimit(txs [][]byte, maxBytes int64) bool {
	return maxBytes < 0 || proposalProtoSize(txs) <= maxBytes
}

func appendProposalTx(txs [][]byte, tx []byte) [][]byte {
	candidate := make([][]byte, len(txs)+1)
	copy(candidate, txs)
	candidate[len(txs)] = tx
	return candidate
}

func gasWouldExceedLimit(total, next, limit uint64) bool {
	return total > limit || next > limit-total
}

func withoutVoteExtInjectionTxs(txs [][]byte) [][]byte {
	capacity := len(txs)
	if capacity > MaxProposalCandidateInspections {
		capacity = MaxProposalCandidateInspections
	}
	regular := make([][]byte, 0, capacity)
	for index, tx := range txs {
		if index >= MaxProposalCandidateInspections {
			break
		}
		if !IsVoteExtInjectionTx(tx) {
			regular = append(regular, tx)
		}
	}
	return regular
}

func validateVoteExtInjectionLayout(txs [][]byte) error {
	found := false
	for index, tx := range txs {
		if !IsVoteExtInjectionTx(tx) {
			continue
		}
		if found {
			return fmt.Errorf("multiple vote extension injections are forbidden")
		}
		if index != 0 {
			return fmt.Errorf("vote extension injection must be transaction 0, got index %d", index)
		}
		found = true
	}
	return nil
}

func rejectProposal() *abci.ResponseProcessProposal {
	return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
}

// PotPreBlocker processes vote extension injection data before BeginBlock.
// Called by baseapp before module BeginBlockers run.
//
// It MUST run the module PreBlockers first: in SDK v0.50 the whole of
// x/upgrade's plan machinery — ApplyUpgrade, the halt-at-height panic,
// upgrade-info.json for cosmovisor — lives ONLY in the module's PreBlock.
// Setting a custom app PreBlocker replaces the SDK default, so skipping
// the module pass silently disables every scheduled upgrade: governance
// stores the plan and the chain sails past the height.
func (app *ZeroneApp) PotPreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	res, err := app.ModuleManager.PreBlock(ctx)
	if err != nil {
		return nil, err
	}

	// Check first tx for VEX injection
	if potVoteExtensionsReleaseEnabled && len(req.Txs) > 0 && IsVoteExtInjectionTx(req.Txs[0]) {
		app.ProcessVoteExtInjection(ctx, req.Txs[0])
	}

	return res, nil
}

// ---- Internal Methods ----

// processVoteExtensions collects commitments and reveals from vote extensions.
// Returns a VoteExtInjection with deterministic ordering for all validators.
func (app *ZeroneApp) processVoteExtensions(
	ctx sdk.Context,
	votes []abci.ExtendedVoteInfo,
) VoteExtInjection {
	var inj VoteExtInjection

	for _, vote := range votes {
		if len(vote.VoteExtension) == 0 {
			continue
		}

		var ext VoteExtension
		if err := json.Unmarshal(vote.VoteExtension, &ext); err != nil {
			continue
		}

		if ext.ValidatorAddress == "" {
			continue
		}

		for _, c := range ext.Commitments {
			if c.RoundID == "" || c.CommitmentHash == "" {
				continue
			}
			inj.Commitments = append(inj.Commitments, InjectedCommitment{
				RoundID:        c.RoundID,
				Validator:      ext.ValidatorAddress,
				CommitmentHash: c.CommitmentHash,
				VRFOutput:      c.VRFOutput,
				VRFProof:       c.VRFProof,
			})
		}

		for _, r := range ext.Reveals {
			if r.RoundID == "" || r.Salt == "" {
				continue
			}
			inj.Reveals = append(inj.Reveals, InjectedReveal{
				RoundID:    r.RoundID,
				Validator:  ext.ValidatorAddress,
				Verdict:    r.Verdict,
				Confidence: r.Confidence,
				Salt:       r.Salt,
			})
		}
	}

	// Sort for deterministic ordering (all validators must process the same data)
	sort.Slice(inj.Commitments, func(i, j int) bool {
		if inj.Commitments[i].RoundID != inj.Commitments[j].RoundID {
			return inj.Commitments[i].RoundID < inj.Commitments[j].RoundID
		}
		return inj.Commitments[i].Validator < inj.Commitments[j].Validator
	})
	sort.Slice(inj.Reveals, func(i, j int) bool {
		if inj.Reveals[i].RoundID != inj.Reveals[j].RoundID {
			return inj.Reveals[i].RoundID < inj.Reveals[j].RoundID
		}
		return inj.Reveals[i].Validator < inj.Reveals[j].Validator
	})

	return inj
}

// ProcessVoteExtInjection processes a vote extension injection tx by storing
// commitments and reveals in the knowledge keeper state.
// Called from PreBlocker before BeginBlock phase transitions.
func (app *ZeroneApp) ProcessVoteExtInjection(ctx sdk.Context, data []byte) {
	logger := ctx.Logger().With("module", "abci", "handler", "PreBlocker")
	if !potVoteExtensionsReleaseEnabled {
		logger.Warn("vote extension injection ignored: release safety latch is closed")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error("PANIC in ProcessVoteExtInjection — skipping injection",
				"height", ctx.BlockHeight(), "panic", fmt.Sprintf("%v", r))
		}
	}()

	inj, err := DecodeVoteExtInjection(data)
	if err != nil {
		logger.Error("failed to decode vote extension injection", "err", err)
		return
	}

	height := uint64(ctx.BlockHeight())
	storedCommitments := 0
	storedReveals := 0

	// Store commitments in keeper state
	for _, c := range inj.Commitments {
		// Re-verify VRF proof before storing each commitment.
		// A malicious proposer could construct a fake injection with invalid proofs.
		if c.VRFOutput == "" || c.VRFProof == "" {
			logger.Warn("injected commitment missing VRF proof — discarding",
				"round_id", c.RoundID,
				"validator", c.Validator,
			)
			continue
		}

		vrfOutput, err := hex.DecodeString(c.VRFOutput)
		if err != nil {
			logger.Warn("invalid VRF output hex in commitment", "round_id", c.RoundID)
			continue
		}
		vrfProof, err := hex.DecodeString(c.VRFProof)
		if err != nil {
			logger.Warn("invalid VRF proof hex in commitment", "round_id", c.RoundID)
			continue
		}

		selected, err := app.KnowledgeKeeper.VerifyValidatorVRFSelection(
			ctx, c.RoundID, c.Validator, vrfOutput, vrfProof,
		)
		if err != nil || !selected {
			logger.Warn("VRF selection verification failed for injected commitment",
				"round_id", c.RoundID,
				"validator", c.Validator,
				"err", err,
			)
			continue
		}

		commitHash, err := hex.DecodeString(c.CommitmentHash)
		if err != nil {
			logger.Warn("invalid commitment hash hex", "round_id", c.RoundID)
			continue
		}

		commitment := &knowledgetypes.CommitEntry{
			Verifier:         c.Validator,
			CommitHash:       commitHash,
			CommittedAtBlock: height,
		}

		if err := app.KnowledgeKeeper.StoreCommitmentInRound(ctx, c.RoundID, commitment); err != nil {
			if errors.Is(err, knowledgetypes.ErrEquivocation) {
				logger.Error("EQUIVOCATION in vote extension commitment",
					"round_id", c.RoundID,
					"validator", c.Validator,
					"error", err.Error(),
				)
			} else {
				logger.Debug("skipped commitment from vote extension",
					"round_id", c.RoundID,
					"validator", c.Validator,
					"reason", err.Error(),
				)
			}
			continue
		}
		storedCommitments++
	}

	// Store reveals in keeper state
	for _, r := range inj.Reveals {
		saltBytes, err := hex.DecodeString(r.Salt)
		if err != nil {
			logger.Warn("invalid salt hex in reveal", "round_id", r.RoundID)
			continue
		}

		reveal := &knowledgetypes.RevealEntry{
			Verifier:        r.Validator,
			Vote:            r.Verdict,
			Salt:            saltBytes,
			RevealedAtBlock: height,
		}

		if err := app.KnowledgeKeeper.StoreRevealInRound(ctx, r.RoundID, reveal, r.Confidence); err != nil {
			if errors.Is(err, knowledgetypes.ErrEquivocation) {
				logger.Error("EQUIVOCATION in vote extension reveal",
					"round_id", r.RoundID,
					"validator", r.Validator,
					"error", err.Error(),
				)
			} else {
				logger.Debug("skipped reveal from vote extension",
					"round_id", r.RoundID,
					"validator", r.Validator,
					"reason", err.Error(),
				)
			}
			continue
		}
		storedReveals++
	}

	if storedCommitments > 0 || storedReveals > 0 {
		logger.Info("processed vote extension injection",
			"height", height,
			"commitments_stored", storedCommitments,
			"commitments_total", len(inj.Commitments),
			"reveals_stored", storedReveals,
			"reveals_total", len(inj.Reveals),
		)
	}
}
