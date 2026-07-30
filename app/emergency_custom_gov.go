package app

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	zeronegov "github.com/zerone-chain/zerone/x/gov"
	zeronegovkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
)

// emergencyAwareCustomGovAppModule freezes every automatic custom-governance
// transition while application transactions are quarantined. Custom x/gov has
// no emergency recovery authority; the sole executable recovery lane is the
// standard SDK x/gov module guarded in emergency_recovery_gov.go.
type emergencyAwareCustomGovAppModule struct {
	zeronegov.AppModule

	emergency customGovEmergencyReader
	keeper    zeronegovkeeper.Keeper
}

type customGovEmergencyReader interface {
	emergencyQuarantineReader
	GetActiveHaltCeremonyId(context.Context) string
}

func newEmergencyAwareCustomGovAppModule(
	module zeronegov.AppModule,
	keeper zeronegovkeeper.Keeper,
	emergency customGovEmergencyReader,
) emergencyAwareCustomGovAppModule {
	return emergencyAwareCustomGovAppModule{
		AppModule: module,
		keeper:    keeper,
		emergency: emergency,
	}
}

func (am emergencyAwareCustomGovAppModule) BeginBlock(
	goCtx context.Context,
) error {
	if am.emergency == nil {
		return fmt.Errorf("quarantine-aware custom governance is not configured")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if am.emergency.IsHalted(goCtx) {
		incidentID := am.emergency.GetActiveHaltCeremonyId(goCtx)
		if incidentID == "" {
			incidentID = fmt.Sprintf(
				"legacy-or-release-latch-quarantine-at-%d",
				ctx.BlockHeight(),
			)
		}
		hold, changed, err := am.keeper.EnsureEmergencyTransitionHold(
			ctx,
			incidentID,
		)
		if err != nil {
			return err
		}
		if changed {
			event := sdk.NewEvent(
				"zerone.gov.custom_transition_hold_activated",
				sdk.NewAttribute("incident_id", hold.IncidentId),
				sdk.NewAttribute(
					"latest_incident_id",
					hold.LatestIncidentId,
				),
				sdk.NewAttribute(
					"incident_count",
					fmt.Sprintf("%d", hold.IncidentCount),
				),
				sdk.NewAttribute(
					"incident_lineage_sha256",
					fmt.Sprintf("%x", hold.IncidentLineageSha256),
				),
				sdk.NewAttribute(
					"activated_at_block",
					fmt.Sprintf("%d", hold.ActivatedAtBlock),
				),
			)
			if hold.IncidentCount > 1 {
				event = sdk.NewEvent(
					"zerone.gov.custom_transition_hold_extended",
					sdk.NewAttribute("incident_id", hold.IncidentId),
					sdk.NewAttribute(
						"latest_incident_id",
						hold.LatestIncidentId,
					),
					sdk.NewAttribute(
						"incident_count",
						fmt.Sprintf("%d", hold.IncidentCount),
					),
					sdk.NewAttribute(
						"incident_lineage_sha256",
						fmt.Sprintf(
							"%x",
							hold.IncidentLineageSha256,
						),
					),
					sdk.NewAttribute(
						"activated_at_block",
						fmt.Sprintf("%d", hold.ActivatedAtBlock),
					),
				)
			}
			ctx.EventManager().EmitEvent(event)
		}
		emitCustomGovFrozen(ctx, hold, "application_transaction_quarantine")
		return nil
	}

	hold, found, err := am.keeper.GetEmergencyTransitionHold(ctx)
	if err != nil {
		return err
	}
	if found {
		emitCustomGovFrozen(ctx, hold, "post_quarantine_review_hold")
		return nil
	}
	return am.AppModule.BeginBlock(goCtx)
}

func emitCustomGovFrozen(
	ctx sdk.Context,
	hold interface {
		GetIncidentId() string
	},
	reason string,
) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.gov.custom_transitions_frozen",
		sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
		sdk.NewAttribute("reason", reason),
		sdk.NewAttribute("incident_id", hold.GetIncidentId()),
		sdk.NewAttribute(
			"release_mechanism",
			"future_named_software_upgrade_with_no_current_release_api_after_complete_queue_reconciliation",
		),
	))
}
