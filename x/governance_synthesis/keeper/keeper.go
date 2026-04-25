package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/zerone-chain/zerone/x/governance_synthesis/types"
)

// Keeper for x/governance_synthesis. Pure consumer; same shape as
// x/training_provenance and x/trust_score.
type Keeper struct {
	cdc codec.BinaryCodec

	knowledgeKeeper        types.KnowledgeKeeper
	captureChallengeKeeper types.CaptureChallengeKeeper
	alignmentKeeper        types.AlignmentKeeper
}

func NewKeeper(cdc codec.BinaryCodec) Keeper {
	return Keeper{cdc: cdc}
}

func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper)              { k.knowledgeKeeper = kk }
func (k *Keeper) SetCaptureChallengeKeeper(cck types.CaptureChallengeKeeper) {
	k.captureChallengeKeeper = cck
}
func (k *Keeper) SetAlignmentKeeper(ak types.AlignmentKeeper) { k.alignmentKeeper = ak }
