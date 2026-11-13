package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// KnowledgeAuthAdapter wraps the zerone auth Keeper to satisfy
// knowledgetypes.ZeroneAuthKeeper.
type KnowledgeAuthAdapter struct {
	k Keeper
}

// NewKnowledgeAuthAdapter returns an adapter for the knowledge module.
func NewKnowledgeAuthAdapter(k Keeper) *KnowledgeAuthAdapter {
	return &KnowledgeAuthAdapter{k: k}
}

var _ knowledgetypes.ZeroneAuthKeeper = (*KnowledgeAuthAdapter)(nil)

// GetAccountType returns the account type for a bech32 address.
func (a *KnowledgeAuthAdapter) GetAccountType(goCtx context.Context, address string) (string, bool) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	account, found := a.k.GetAccount(ctx, address)
	if !found {
		return "", false
	}
	return account.AccountType, true
}

// CheckReviewer applies, inside the knowledge handler, the admission the ante
// applies to a MsgSubmitCommitment signer: registered with zerone_auth,
// CanSubmitClaims set, and not frozen (ZeroneCapabilityDecorator and
// ZeroneAccountDecorator in app/ante_zerone.go). A round with seat terms
// calls it (reviewer-entry step B, D3), because simulation, the
// interchain-accounts host and any future route that skips the ante would
// otherwise store a seat for an account the ante would refuse.
func (a *KnowledgeAuthAdapter) CheckReviewer(goCtx context.Context, address string) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	account, found := a.k.GetAccount(ctx, address)
	if !found {
		return knowledgetypes.ErrReviewerNotAdmitted.Wrapf("%s is not registered with zerone_auth", address)
	}
	if account.Flags == nil || !account.Flags.CanSubmitClaims {
		return knowledgetypes.ErrReviewerNotAdmitted.Wrapf("account type %q cannot review (CanSubmitClaims=false)", account.AccountType)
	}
	if account.Flags.Frozen {
		return knowledgetypes.ErrReviewerNotAdmitted.Wrapf("%s is frozen", address)
	}
	return nil
}
