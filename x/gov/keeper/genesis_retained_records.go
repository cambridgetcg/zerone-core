package keeper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// initRetainedRecords restores storage directly: no submission, ballot handler,
// disbursement, approval, phase advancement or first-vote-height recomputation.
func (k Keeper) initRetainedRecords(ctx sdk.Context, gs *types.GenesisState) {
	for _, p := range gs.ResearchSpendProposals {
		k.SetResearchSpendProposal(ctx, p)
	}
	if gs.NextResearchSpendNumber != nil {
		k.SetNextResearchSpendID(ctx, *gs.NextResearchSpendNumber)
	}
	for _, v := range gs.ResearchCommunityVotes {
		k.SetResearchCommunityVote(ctx, v.ProposalId, v.Voter, v.Vote)
	}
	for _, v := range gs.DistinctVoters {
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, v.FirstVotedAtBlock)
		ctx.KVStore(k.storeKey).Set(types.DistinctVoterKey(v.Voter), bz)
	}
	for _, p := range gs.PhaseTransitions {
		k.SetPhaseTransitionMeta(ctx, &types.PhaseTransitionProposal{
			LipID: p.LipId, TargetPhase: p.TargetPhase,
			ConditionsSnapshot: p.ConditionsSnapshot, Stage: p.Stage,
			ActivationBlock: p.ActivationBlock, IsRollback: p.IsRollback,
			CancelReason: p.CancelReason,
		})
	}
}

// exportRetainedRecords is deliberately separate from the live iterators that
// skip malformed rows. Export must fail closed, never silently lose a record.
// FX8 has no activation marker: this is genesis serialization, not a block hook.
func (k Keeper) exportRetainedRecords(ctx sdk.Context, gs *types.GenesisState) error {
	store := ctx.KVStore(k.storeKey)
	if err := walkRetainedRecords(store, types.ResearchSpendKeyPrefix, func(key, value []byte) error {
		var p *types.ResearchSpendProposal
		if err := decodeRetainedRecord(value, &p); err != nil {
			return err
		}
		if p == nil || !bytes.Equal(key, types.ResearchSpendKey(p.ProposalId)) {
			return fmt.Errorf("research proposal key/payload mismatch")
		}
		gs.ResearchSpendProposals = append(gs.ResearchSpendProposals, p)
		return nil
	}); err != nil {
		return err
	}
	if bz := store.Get(types.ResearchSpendCounterKey); bz != nil {
		if len(bz) != 8 {
			return fmt.Errorf("research spend counter must contain exactly 8 bytes")
		}
		next := binary.BigEndian.Uint64(bz)
		gs.NextResearchSpendNumber = &next
	}
	if err := walkRetainedRecords(store, types.ResearchCommunityVotePrefix, func(key, value []byte) error {
		offset := len(types.ResearchCommunityVotePrefix)
		if len(key) <= offset+9 || key[offset+8] != 0 {
			return fmt.Errorf("malformed research community vote key")
		}
		gs.ResearchCommunityVotes = append(gs.ResearchCommunityVotes, &types.GenesisResearchCommunityVote{
			ProposalId: binary.BigEndian.Uint64(key[offset : offset+8]),
			Voter:      string(key[offset+9:]), Vote: string(value),
		})
		return nil
	}); err != nil {
		return err
	}
	if err := walkRetainedRecords(store, types.DistinctVoterKeyPrefix, func(key, value []byte) error {
		if len(key) <= len(types.DistinctVoterKeyPrefix) || len(value) != 8 {
			return fmt.Errorf("malformed distinct voter key/height")
		}
		gs.DistinctVoters = append(gs.DistinctVoters, &types.GenesisDistinctVoter{
			Voter: string(key[len(types.DistinctVoterKeyPrefix):]), FirstVotedAtBlock: binary.BigEndian.Uint64(value),
		})
		return nil
	}); err != nil {
		return err
	}
	if err := walkRetainedRecords(store, types.PhaseTransitionKeyPrefix, func(key, value []byte) error {
		var p *types.PhaseTransitionProposal
		if err := decodeRetainedRecord(value, &p); err != nil {
			return err
		}
		if p == nil || !bytes.Equal(key, types.PhaseTransitionKey(p.LipID)) {
			return fmt.Errorf("phase transition key/payload mismatch")
		}
		gs.PhaseTransitions = append(gs.PhaseTransitions, &types.GenesisPhaseTransition{
			LipId: p.LipID, TargetPhase: p.TargetPhase,
			ConditionsSnapshot: p.ConditionsSnapshot, Stage: p.Stage,
			ActivationBlock: p.ActivationBlock, IsRollback: p.IsRollback,
			CancelReason: p.CancelReason,
		})
		return nil
	}); err != nil {
		return err
	}
	return gs.ValidateRetainedRecords()
}

func decodeRetainedRecord(bz []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(bz))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing retained record JSON: %v", err)
	}
	return nil
}

// walkRetainedRecords preserves byte-key order and checks traversal and close
// errors. The existing SDK normal-EOF accommodation is shared without changing
// it or any live iterator. The callback must not write to the iterated store.
func walkRetainedRecords(store storetypes.KVStore, prefix []byte, visit func([]byte, []byte) error) (err error) {
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer func() {
		err = errors.Join(err, terminalLIPIteratorError(iter), iter.Close())
	}()
	for ; iter.Valid(); iter.Next() {
		if err := visit(iter.Key(), iter.Value()); err != nil {
			return fmt.Errorf("retained record at key %x: %w", iter.Key(), err)
		}
	}
	return nil
}
