package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// roundsExecution carries entry-point marker snapshots by value. It is never
// stored on the wired Keeper or in context. Its methods shadow the standalone
// entry points, so loops, cached contexts and challenge paths cannot re-read a
// marker. Unchanged orchestration bodies retain their historical call spelling.
// With every flag false each changed leaf keeps its predecessor behaviour: the
// calibration and zero-floor leaves dispatch to their frozen legacy bodies, and
// metabolism keeps its label-only doctrine exemption.
type roundsExecution struct {
	Keeper
	calibrationSmoothing bool
	zeroConfidenceFloor  bool
	doctrineIdentity     bool
}

func (k Keeper) selectRoundsExecution(ctx context.Context) (roundsExecution, error) {
	calibration, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return roundsExecution{}, err
	}
	floor, err := k.ZeroConfidenceFloorEnabled(ctx)
	if err != nil {
		return roundsExecution{}, err
	}
	// Selected here, at entry (BeginBlocker, and the standalone CompleteRound
	// and AdvanceRoundPhases), so an unreadable or malformed marker halts the
	// block before any write. BeginBlocker logs a ProcessMetabolism error and
	// carries on, so a fault read only at the fitness epoch would skip
	// metabolism quietly instead. The two standalone entries never reach
	// metabolism, but they read the marker too and so also return its fault,
	// as they do for the other two markers.
	doctrine, err := k.DoctrineIdentityEnabled(ctx)
	if err != nil {
		return roundsExecution{}, err
	}
	return roundsExecution{Keeper: k, calibrationSmoothing: calibration, zeroConfidenceFloor: floor, doctrineIdentity: doctrine}, nil
}

// CompleteRound is also a standalone entry point. Block processing instead
// calls roundsExecution.CompleteRound with the already-selected booleans.
func (k Keeper) CompleteRound(ctx context.Context, round *types.VerificationRound, result *VerificationResult) error {
	execution, err := k.selectRoundsExecution(ctx)
	if err != nil {
		return err
	}
	return execution.CompleteRound(ctx, round, result)
}

func (k Keeper) AdvanceRoundPhases(ctx context.Context) error {
	execution, err := k.selectRoundsExecution(ctx)
	if err != nil {
		return err
	}
	return execution.AdvanceRoundPhases(ctx)
}

func (k Keeper) AdvanceConfidence(ctx context.Context) error {
	enabled, err := k.ZeroConfidenceFloorEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, zeroConfidenceFloor: enabled}).AdvanceConfidence(ctx)
}

func (k roundsExecution) AdvanceConfidence(ctx context.Context) error {
	if !k.zeroConfidenceFloor {
		return k.advanceConfidenceLegacy(ctx)
	}
	return k.advanceConfidenceWithZeroFloor(ctx)
}

// ProcessMetabolism is also a standalone entry point. It reads the
// doctrine-identity marker once, before any fact is read, and returns a fault
// rather than choose. Both BeginBlocker bodies instead call
// roundsExecution.ProcessMetabolism with the marker selected at entry.
func (k Keeper) ProcessMetabolism(ctx context.Context, epoch uint64) error {
	enabled, err := k.DoctrineIdentityEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, doctrineIdentity: enabled}).ProcessMetabolism(ctx, epoch)
}

func (k roundsExecution) ProcessMetabolism(ctx context.Context, epoch uint64) error {
	return k.processMetabolism(ctx, epoch, k.doctrineIdentity)
}

func (k Keeper) RecordSubmissionOutcome(ctx context.Context, submitter, methodID string, verdict types.Verdict) error {
	enabled, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, calibrationSmoothing: enabled}).RecordSubmissionOutcome(ctx, submitter, methodID, verdict)
}

func (k Keeper) RecordCorroborationForSubmitter(ctx context.Context, submitter, methodID string) error {
	enabled, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, calibrationSmoothing: enabled}).RecordCorroborationForSubmitter(ctx, submitter, methodID)
}

func (k Keeper) RecordDisprovalForSubmitter(ctx context.Context, submitter, methodID string) error {
	enabled, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, calibrationSmoothing: enabled}).RecordDisprovalForSubmitter(ctx, submitter, methodID)
}

func (k Keeper) RecordChallengeOutcome(ctx context.Context, challenger string, succeeded bool) error {
	enabled, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, calibrationSmoothing: enabled}).RecordChallengeOutcome(ctx, challenger, succeeded)
}
