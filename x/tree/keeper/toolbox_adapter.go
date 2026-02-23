package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/tree/types"
)

// ServiceExists returns true if a service with the given ID exists in state.
func (k Keeper) ServiceExists(ctx sdk.Context, serviceID string) bool {
	_, found := k.GetService(ctx, serviceID)
	return found
}

// IsServiceContributor returns true if the given address is a contributor on
// the service's parent project.
func (k Keeper) IsServiceContributor(ctx sdk.Context, serviceID, address string) bool {
	service, found := k.GetService(ctx, serviceID)
	if !found {
		return false
	}
	if service.ProjectId == "" {
		return true
	}
	project, projFound := k.GetProject(ctx, service.ProjectId)
	if !projFound {
		return true
	}
	for _, contrib := range project.Contributors {
		if contrib.Did == address {
			return true
		}
	}
	return false
}

// CallService executes a service call at the keeper level.
func (k Keeper) CallService(ctx sdk.Context, caller, serviceID string, inputData []byte) ([]byte, error) {
	service, found := k.GetService(ctx, serviceID)
	if !found {
		return nil, types.ErrServiceNotFound.Wrapf("service %s not found", serviceID)
	}
	if service.Status != string(types.ServiceActive) {
		return nil, types.ErrServiceNotActive.Wrapf("service %s has status %s", serviceID, service.Status)
	}

	prev, _ := strconv.ParseUint(service.TotalCalls, 10, 64)
	service.TotalCalls = fmt.Sprintf("%d", prev+1)
	service.LastCalledBlock = uint64(ctx.BlockHeight())
	k.SetService(ctx, service)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.tree.service_called_via_adapter",
		sdk.NewAttribute("service_id", serviceID),
		sdk.NewAttribute("caller", caller),
		sdk.NewAttribute("via", "toolbox_adapter"),
	))

	return nil, nil
}
