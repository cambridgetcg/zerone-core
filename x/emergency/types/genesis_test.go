package types

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNormalizeLegacyParams(t *testing.T) {
	legacy := DefaultParams()
	legacy.HaltQuorum = 0
	legacy.RevertQuorum = 0
	legacy.ResumeQuorum = 0
	legacy.RevertPrevoteBlocks = 0
	legacy.RevertPrecommitBlocks = 0
	legacy.RevertTimeoutBlocks = 0
	legacy.ResumePrevoteBlocks = 0
	legacy.ResumePrecommitBlocks = 0
	legacy.ResumeTimeoutBlocks = 0
	legacy.MaxProposalsPerEpoch = 0
	legacy.MaxProposalsPerGuardianPerEpoch = 0
	legacy.MinGuardianStake = ""
	legacy.MinDistinctVoters = 0
	legacy.MaxRevertDepth = 0

	normalized := NormalizeLegacyParams(&legacy)
	if err := normalized.Validate(); err != nil {
		t.Fatalf("normalized legacy params remain invalid: %v", err)
	}
	if normalized.HaltQuorum == 0 || normalized.ResumeTimeoutBlocks == 0 ||
		normalized.MinDistinctVoters == 0 {
		t.Fatalf("required safety fields were not restored: %+v", normalized)
	}
}

func TestGenesisCouncilRequiresMessageCompatibleAddress(t *testing.T) {
	params := DefaultParams()
	params.GenesisCouncil = []string{"zrn1not-a-valid-address"}
	params.CouncilExpiryBlock = 100
	if err := params.Validate(); err == nil {
		t.Fatal("expected invalid genesis council address rejection")
	}

	params.GenesisCouncil = []string{
		sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String(),
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("message-compatible council address rejected: %v", err)
	}
}

func TestParamsRejectImpossibleCeremonyTimings(t *testing.T) {
	params := DefaultParams()
	params.HaltTimeoutBlocks = params.HaltPrevoteBlocks + params.HaltPrecommitBlocks - 1
	if err := params.Validate(); err == nil {
		t.Fatal("expected timeout shorter than ceremony phases to be rejected")
	}

	params = DefaultParams()
	params.ResumePrevoteBlocks = ^uint64(0)
	params.ResumePrecommitBlocks = 1
	params.ResumeTimeoutBlocks = ^uint64(0)
	if err := params.Validate(); err == nil {
		t.Fatal("expected overflowing phase duration to be rejected")
	}
}

func TestNormalizeDoesNotRepairPreviouslyRejectedParams(t *testing.T) {
	legacy := DefaultParams()
	legacy.HaltQuorum = ^uint64(0)
	legacy.HaltPrevoteBlocks = 0
	legacy.HaltTimeoutBlocks = 0
	legacy.EpochBlocks = 0
	legacy.MaxHaltDurationBlocks = 0

	normalized := NormalizeLegacyParams(&legacy)
	if err := normalized.Validate(); err == nil {
		t.Fatal("parameters rejected by the prior validator must remain invalid")
	}
	if normalized.HaltQuorum != legacy.HaltQuorum ||
		normalized.HaltPrevoteBlocks != legacy.HaltPrevoteBlocks ||
		normalized.HaltTimeoutBlocks != legacy.HaltTimeoutBlocks ||
		normalized.EpochBlocks != legacy.EpochBlocks ||
		normalized.MaxHaltDurationBlocks != legacy.MaxHaltDurationBlocks {
		t.Fatal("normalization changed fields the prior validator rejected")
	}
}

func TestNormalizeDisablesUnsignableLegacyCouncil(t *testing.T) {
	legacy := DefaultParams()
	legacy.GenesisCouncil = []string{"invalid"}
	legacy.CouncilExpiryBlock = 100
	legacy.CouncilVirtualStake = "not-a-number"

	normalized := NormalizeLegacyParams(&legacy)
	if err := normalized.Validate(); err != nil {
		t.Fatalf("legacy council normalization left params invalid: %v", err)
	}
	if len(normalized.GenesisCouncil) != 0 {
		t.Fatal("invalid legacy council must be disabled")
	}
}

func TestGenesisQuarantineReleaseBlockRequiresNormalFiniteState(t *testing.T) {
	genesis := DefaultGenesis()
	genesis.Status = string(StatusHalted)
	genesis.QuarantineReleaseBlock = 100
	if err := genesis.Validate(); err == nil {
		t.Fatal("non-normal genesis accepted a quarantine release block")
	}

	genesis = DefaultGenesis()
	genesis.QuarantineReleaseBlock = MaxSDKBlockHeight - PostResumeCancellationGraceBlocks + 1
	if err := genesis.Validate(); err == nil {
		t.Fatal("release block without a representable grace window was accepted")
	}

	genesis.QuarantineReleaseBlock = 100
	if err := genesis.Validate(); err != nil {
		t.Fatalf("finite normal-state release latch was rejected: %v", err)
	}
}
