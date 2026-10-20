package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// getDomainCountLegacy preserves the GetDomainCount body verbatim from
// 77521a0d (before #124), including all statuses and its original gas usage.
func (a *AlignmentOntologyAdapter) getDomainCountLegacy(ctx context.Context) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	domains := a.k.GetAllDomains(sdkCtx)
	return uint64(len(domains))
}
