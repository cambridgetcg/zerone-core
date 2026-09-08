package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	emergency "github.com/zerone-chain/zerone/x/emergency"
	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

// Real keeper/message-server transitions on independent in-memory stores. The
// one-guardian electorate is fixture-local; this is not a disk restart proof.
func TestPostResumeGraceLaterHaltContinuation(t *testing.T) {
	for _, outcome := range []string{"pending", "rejected", "timeout", "finalized"} {
		t.Run(outcome, func(t *testing.T) {
			k, staking, ctx := setupKeeper(t)
			guardian := testCouncilAddr(1)
			staking.addGuardian(guardian, "100000000000")
			params := k.GetParams(ctx)
			params.HaltPrevoteBlocks = 2
			params.HaltPrecommitBlocks = 2
			params.HaltTimeoutBlocks = 5
			params.MaxHaltDurationBlocks = 2
			k.SetParams(ctx, params)
			require.Zero(t, params.CooldownBlocks)
			srv := keeper.NewMsgServerImpl(k)
			halt, err := srv.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: guardian, Reason: "first incident"})
			require.NoError(t, err)
			for i := 0; i < 2; i++ {
				_, err = srv.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardian, ProposalId: halt.ProposalId, Approve: true})
				require.NoError(t, err)
			}
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
			resume, err := srv.ProposeResume(ctx, testResumeMsg(guardian))
			require.NoError(t, err)
			for i := 0; i < 2; i++ {
				_, err = srv.VoteResume(ctx, &types.MsgVoteResume{Voter: guardian, ProposalId: resume.ProposalId, Approve: true})
				require.NoError(t, err)
			}
			release := uint64(ctx.BlockHeight())
			require.Equal(t, release, k.GetQuarantineReleaseBlock(ctx))
			require.True(t, k.IsHalted(ctx), "resume block itself stays quarantined")
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
			require.NoError(t, emergency.NewAppModule(nil, k).BeginBlock(ctx))
			nextHalt, err := srv.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: guardian, Reason: "later incident during grace"})
			require.NoError(t, err)
			require.Equal(t, types.StatusHaltVoting, k.GetEmergencyStatus(ctx))
			require.Equal(t, release, k.GetQuarantineReleaseBlock(ctx), "proposal alone cannot erase cancellation grace")
			require.False(t, k.IsHalted(ctx), "a vote is not quarantine authority")

			exported := k.ExportGenesis(ctx)
			require.NoError(t, exported.Validate(), "a legitimate later halt vote must be importable")
			// Reject imported combinations not reachable from this transition.
			for _, mutation := range []func(*types.GenesisState){
				func(g *types.GenesisState) { g.Ceremonies = nil },
				func(g *types.GenesisState) { g.ActiveHaltCeremonyId = halt.ProposalId; g.HaltStartBlock = release - 1 },
				func(g *types.GenesisState) { g.LastHaltEscalationBlock = release + 1 },
				func(g *types.GenesisState) { g.QuarantineReleaseBlock = uint64(ctx.BlockHeight()) },
			} {
				invalid := proto.Clone(exported).(*types.GenesisState)
				mutation(invalid)
				require.Error(t, invalid.Validate())
			}
			imported, importedStaking, importedCtx := setupKeeper(t)
			importedCtx = importedCtx.WithBlockHeight(ctx.BlockHeight())
			imported.InitGenesis(importedCtx, exported)
			require.True(t, proto.Equal(exported, imported.ExportGenesis(importedCtx)))
			require.Equal(t, release, imported.GetQuarantineReleaseBlock(importedCtx))
			active, found := imported.GetActiveCeremony(importedCtx)
			require.True(t, found)
			require.Equal(t, nextHalt.ProposalId, active.Id)
			importedSrv := keeper.NewMsgServerImpl(imported)

			switch outcome {
			case "pending":
				return // The active imported electorate survives without a fresh staking snapshot.
			case "rejected":
				_, err = importedSrv.VoteHalt(importedCtx, &types.MsgVoteHalt{Voter: guardian, ProposalId: active.Id, Approve: false})
				require.NoError(t, err)
			case "timeout":
				importedCtx = importedCtx.WithBlockHeight(int64(active.TimeoutDeadline + 1))
				require.NoError(t, emergency.NewAppModule(nil, imported).BeginBlock(importedCtx))
			case "finalized":
				for i := 0; i < 2; i++ {
					_, err = importedSrv.VoteHalt(importedCtx, &types.MsgVoteHalt{Voter: guardian, ProposalId: active.Id, Approve: true})
					require.NoError(t, err)
				}
				require.Equal(t, types.StatusHalted, imported.GetEmergencyStatus(importedCtx))
				require.Zero(t, imported.GetQuarantineReleaseBlock(importedCtx), "finalized quarantine supersedes old grace, not a mere proposal")
				importedCtx = importedCtx.WithBlockHeight(importedCtx.BlockHeight() + 3)
				require.NoError(t, emergency.NewAppModule(nil, imported).BeginBlock(importedCtx))
				require.NotZero(t, imported.GetLastHaltEscalationBlock(importedCtx))
				require.True(t, imported.IsHalted(importedCtx), "escalation cannot reopen admission")
				require.True(t, imported.IsHalted(importedCtx.WithBlockHeight(int64(release+types.PostResumeCancellationGraceBlocks+1))), "expiry of the old grace cannot reopen the new quarantine")
				require.NoError(t, imported.ExportGenesis(importedCtx).Validate())
				// A later affirmative resume, unlike escalation, installs a fresh
				// full grace window for the new incident.
				importedStaking.addGuardian(guardian, "100000000000")
				newResume, err := importedSrv.ProposeResume(importedCtx, testResumeMsg(guardian))
				require.NoError(t, err)
				require.NoError(t, imported.ExportGenesis(importedCtx).Validate())
				for i := 0; i < 2; i++ {
					_, err = importedSrv.VoteResume(importedCtx, &types.MsgVoteResume{Voter: guardian, ProposalId: newResume.ProposalId, Approve: true})
					require.NoError(t, err)
				}
				require.Greater(t, uint64(importedCtx.BlockHeight()), release)
				require.Equal(t, uint64(importedCtx.BlockHeight()), imported.GetQuarantineReleaseBlock(importedCtx))
				require.True(t, imported.IsHalted(importedCtx), "new resume block is still quarantined")
				require.NoError(t, imported.ExportGenesis(importedCtx).Validate())
				return
			}
			require.Equal(t, types.StatusNormal, imported.GetEmergencyStatus(importedCtx))
			require.Equal(t, release, imported.GetQuarantineReleaseBlock(importedCtx), "failed vote must preserve original grace")
			require.NoError(t, imported.ExportGenesis(importedCtx).Validate())
			for height := importedCtx.BlockHeight(); height <= int64(release+types.PostResumeCancellationGraceBlocks); height++ {
				importedCtx = importedCtx.WithBlockHeight(height)
				require.NoError(t, emergency.NewAppModule(nil, imported).BeginBlock(importedCtx))
				require.Equal(t, release, imported.GetQuarantineReleaseBlock(importedCtx))
			}
		})
	}
}
