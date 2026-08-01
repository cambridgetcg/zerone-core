package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Status returns the current emergency status.
func (q queryServer) Status(goCtx context.Context, _ *types.QueryStatusRequest) (*types.QueryStatusResponse, error) {
	status := q.GetEmergencyStatus(goCtx)
	params := q.GetParams(goCtx)
	readiness := q.GetGuardianReadiness(goCtx)
	startedAt := q.GetHaltStartBlock(goCtx)
	restricted := q.IsHalted(goCtx)
	releaseBlock := q.GetQuarantineReleaseBlock(goCtx)
	reopensAt := uint64(0)
	if releaseBlock != 0 {
		reopensAt = releaseBlock + 1
	}

	deadlineExceeded := false
	if restricted && startedAt > 0 {
		currentHeight := sdk.UnwrapSDKContext(goCtx).BlockHeight()
		if currentHeight >= 0 {
			current := uint64(currentHeight)
			deadlineExceeded = current >= startedAt &&
				current-startedAt >= params.MaxHaltDurationBlocks
		}
	}

	return &types.QueryStatusResponse{
		Status:                      string(status),
		IsHalted:                    restricted,
		ActiveHaltCeremonyId:        q.GetActiveHaltCeremonyId(goCtx),
		RestrictionScope:            "application_transactions",
		ConsensusContinues:          true,
		EligibleGuardians:           readiness.EligibleGuardians,
		EffectiveGuardianStake:      readiness.EffectiveStake.String(),
		MinimumDistinctVoters:       params.MinDistinctVoters,
		HaltCeremonyReady:           readiness.Ready,
		ReadinessReason:             readiness.Reason,
		AutomaticResumeEnabled:      false,
		ArbitraryStateRevertEnabled: false,
		QuarantineDeadlineExceeded:  deadlineExceeded,
		QuarantineStartedAtBlock:    startedAt,
		QuarantineReleaseBlock:      releaseBlock,
		AdmissionReopensAtBlock:     reopensAt,
	}, nil
}

// ActiveCeremony returns the currently active ceremony, if any.
func (q queryServer) ActiveCeremony(goCtx context.Context, _ *types.QueryActiveCeremonyRequest) (*types.QueryActiveCeremonyResponse, error) {
	ceremony, found := q.GetActiveCeremony(goCtx)
	if !found {
		return &types.QueryActiveCeremonyResponse{Found: false}, nil
	}
	return &types.QueryActiveCeremonyResponse{
		Ceremony: ceremony,
		Found:    true,
	}, nil
}

// CompletedCeremonies returns completed (finalized/failed) ceremonies.
func (q queryServer) CompletedCeremonies(goCtx context.Context, req *types.QueryCompletedCeremoniesRequest) (*types.QueryCompletedCeremoniesResponse, error) {
	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	var completed []*types.EmergencyCeremony
	var total uint64

	q.IterateCeremonies(goCtx, func(c *types.EmergencyCeremony) bool {
		if c.Phase == string(types.PhaseFinalized) || c.Phase == string(types.PhaseFailed) {
			total++
			if total > uint64(req.Offset) && uint32(len(completed)) < limit {
				completed = append(completed, c)
			}
		}
		return false
	})

	return &types.QueryCompletedCeremoniesResponse{
		Ceremonies: completed,
		Total:      total,
	}, nil
}

// AuditLog returns the emergency audit log.
func (q queryServer) AuditLog(goCtx context.Context, req *types.QueryAuditLogRequest) (*types.QueryAuditLogResponse, error) {
	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	entries := q.GetAuditLog(goCtx)
	total := uint64(len(entries))

	start := uint32(0)
	if req.Offset < uint32(len(entries)) {
		start = req.Offset
	} else {
		start = uint32(len(entries))
	}
	end := start + limit
	if end > uint32(len(entries)) {
		end = uint32(len(entries))
	}

	return &types.QueryAuditLogResponse{
		Entries: entries[start:end],
		Total:   total,
	}, nil
}

// Params returns the emergency module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params := q.GetParams(goCtx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// RecoveryAuthorization returns the current incident-bound SDK governance
// recovery capability, including terminal outcome when consumed.
func (q queryServer) RecoveryAuthorization(
	goCtx context.Context,
	_ *types.QueryRecoveryAuthorizationRequest,
) (*types.QueryRecoveryAuthorizationResponse, error) {
	authorization, found, err := q.GetRecoveryAuthorization(goCtx)
	if err != nil {
		return nil, err
	}
	return &types.QueryRecoveryAuthorizationResponse{
		Found:         found,
		Authorization: authorization,
	}, nil
}
