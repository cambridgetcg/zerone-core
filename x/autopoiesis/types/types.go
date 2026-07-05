package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SSI category strings.
const (
	SSICritical = "critical"
	SSIStressed = "stressed"
	SSIHealthy  = "healthy"
	SSIThriving = "thriving"
)

// Valid multiplier paths.
var ValidPaths = map[string]bool{
	"rewards.block":      true,
	"slashing.severity":  true,
	"fees.base":          true,
}

// ComputeSSI calculates the System Stability Index from cross-module signals.
// Weights: 40% staking participation, 40% verification rate, 20% emergency.
// Returns a value in [0, 1,000,000] where higher = healthier.
func ComputeSSI(stakingParticipation, verificationRate uint64, isHalted bool) uint64 {
	// Cap inputs at BPSScale
	if stakingParticipation > BPSScale {
		stakingParticipation = BPSScale
	}
	if verificationRate > BPSScale {
		verificationRate = BPSScale
	}

	var emergencyScore uint64
	if isHalted {
		emergencyScore = 0
	} else {
		emergencyScore = BPSScale
	}

	// Weighted average: 40% + 40% + 20%
	ssi := (stakingParticipation*40 + verificationRate*40 + emergencyScore*20) / 100
	if ssi > BPSScale {
		ssi = BPSScale
	}
	return ssi
}

// ClassifySSI returns the SSI category based on thresholds.
func ClassifySSI(ssi uint64, params *Params) string {
	switch {
	case ssi < params.SsiCriticalThreshold:
		return SSICritical
	case ssi < params.SsiStressedThreshold:
		return SSIStressed
	case ssi < params.SsiHealthyThreshold:
		return SSIHealthy
	default:
		return SSIThriving
	}
}

// ComputeTarget returns the target multiplier for a path given the SSI score.
//
// - rewards.block: higher SSI → higher rewards (encourage continued participation)
// - slashing.severity: lower SSI → higher slashing (enforce discipline under stress)
// - fees.base: lower SSI → higher fees (reduce spam under stress)
func ComputeTarget(ssi uint64, path string) uint64 {
	switch path {
	case "rewards.block":
		// Scale linearly: SSI=0 → 0.5x, SSI=1M → 1.5x.
		// NOTE: this SSI-driven block multiplier is currently DORMANT (nothing reads
		// GetMultiplier("rewards.block")). It is fed by the accept-rate via SSI, so if
		// it is ever wired to emission it would re-introduce the accept-rate coupling
		// the survival-gate removes — freeze it via MsgFreezeMultiplier / genesis then.
		return 500_000 + ssi
	case "slashing.severity":
		// Inverse: SSI=0 → 2.0x, SSI=1M → 0.5x
		return 2_000_000 - (ssi * 3 / 2)
	case "fees.base":
		// Inverse: SSI=0 → 2.0x, SSI=1M → 0.5x
		return 2_000_000 - (ssi * 3 / 2)
	default:
		return BPSScale // 1.0x fallback
	}
}

// RegisterCodec registers the autopoiesis module's message types.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "autopoiesis/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgActivateAutopoiesis{}, "autopoiesis/MsgActivateAutopoiesis", nil)
	cdc.RegisterConcrete(&MsgOverrideMultiplier{}, "autopoiesis/MsgOverrideMultiplier", nil)
	cdc.RegisterConcrete(&MsgFreezeMultiplier{}, "autopoiesis/MsgFreezeMultiplier", nil)
}

// RegisterInterfaces registers the autopoiesis module's interface types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgActivateAutopoiesis{},
		&MsgOverrideMultiplier{},
		&MsgFreezeMultiplier{},
	)
}
