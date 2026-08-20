package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// ValidateSHA256Hex follows x/knowledge's established bare lowercase 64-hex
// convention while keeping sponsorship's message validation self-contained.
func ValidateSHA256Hex(field, value string) error {
	if len(value) != knowledgetypes.SHA256HexLength {
		return fmt.Errorf("%s must be %d lowercase SHA-256 hex characters", field, knowledgetypes.SHA256HexLength)
	}
	if value != strings.ToLower(value) {
		return fmt.Errorf("%s must use lowercase SHA-256 hex", field)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s must be valid lowercase SHA-256 hex: %w", field, err)
	}
	return nil
}

func (c *WorkContract) Validate() error {
	if c == nil {
		return fmt.Errorf("work_contract is required")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"work_spec_hash", c.WorkSpecHash},
		{"acceptance_hash", c.AcceptanceHash},
		{"input_root", c.InputRoot},
		{"environment_root", c.EnvironmentRoot},
	} {
		if err := ValidateSHA256Hex(field.name, field.value); err != nil {
			return err
		}
	}
	worker, err := sdk.AccAddressFromBech32(c.WorkerAddress)
	if err != nil {
		return fmt.Errorf("worker_address must be a valid account address: %w", err)
	}
	if worker.String() != c.WorkerAddress {
		return fmt.Errorf("worker_address must use canonical lowercase bech32 encoding")
	}
	return nil
}

// MatchesComputationalCommitment checks the four digest constraints.
// Artifact/evidence/receipt roots are result-specific and intentionally not
// supplied by the sponsor or FulfillBounty caller. The handler separately
// requires Fact.Submitter to equal WorkerAddress.
func (c *WorkContract) MatchesComputationalCommitment(work *knowledgetypes.ComputationalCommitment) bool {
	return c != nil && work != nil &&
		c.WorkSpecHash == work.WorkSpecHash &&
		c.AcceptanceHash == work.AcceptanceHash &&
		c.InputRoot == work.InputRoot &&
		c.EnvironmentRoot == work.EnvironmentRoot
}

// ComputeSettlementNullifier is sponsorship-global. It deliberately excludes
// bounty ID, fact ID, caller, height, evidence, and receipt so wrapping the
// same artifact in fresh evidence/receipt records cannot make it payable
// again under the same immutable work contract and canonical worker
// assignment. WorkContract.Validate enforces the lowercase Bech32 encoding
// hashed here. Other modules need their own shared nullifier registry before claiming
// cross-module replay safety.
func ComputeSettlementNullifier(workSpecHash, acceptanceHash, inputRoot, environmentRoot, artifactRoot, workerAddress string) string {
	h := sha256.New()
	h.Write([]byte("ZRN.sponsorship.settlement.v2\x00"))
	writePart := func(value string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		h.Write(size[:])
		h.Write([]byte(value))
	}
	writePart(workSpecHash)
	writePart(acceptanceHash)
	writePart(inputRoot)
	writePart(environmentRoot)
	writePart(artifactRoot)
	writePart(workerAddress)
	return hex.EncodeToString(h.Sum(nil))
}

func ParsePositiveAmount(value string) (*big.Int, error) {
	amount, err := parseCanonicalAmount(value, false)
	if err != nil {
		return nil, err
	}
	return amount, nil
}

func ParseNonNegativeAmount(value string) (*big.Int, error) {
	return parseCanonicalAmount(value, true)
}

// NormalizeLegacyPositiveAmount accepts exactly the historical v1 positive
// base-10 forms understood by big.Int (including leading zeroes and a leading
// plus) and returns their single v2 canonical representation. The sdk.Int
// bound prevents import from reinterpreting an amount v2 could not hold.
func NormalizeLegacyPositiveAmount(value string) (string, error) {
	return normalizeLegacyAmount(value, false)
}

// NormalizeLegacyNonNegativeAmount canonicalizes v1-authored
// escrow_remaining values. Runtime-created v1 orders were already canonical,
// but v1 genesis validation admitted equivalent forms such as "+2" and
// "002". Only nil-WorkContract records may use this compatibility path.
func NormalizeLegacyNonNegativeAmount(value string) (string, error) {
	return normalizeLegacyAmount(value, true)
}

func normalizeLegacyAmount(value string, allowZero bool) (string, error) {
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok || amount.Sign() < 0 || (!allowZero && amount.Sign() == 0) || amount.BitLen() > sdkmath.MaxBitLen {
		kind := "positive"
		if allowZero {
			kind = "non-negative"
		}
		return "", fmt.Errorf("%q is not a %s base-10 integer within %d bits", value, kind, sdkmath.MaxBitLen)
	}
	return amount.String(), nil
}

func parseCanonicalAmount(value string, allowZero bool) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("amount cannot be empty")
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok || amount.String() != value {
		return nil, fmt.Errorf("%q is not a canonical base-10 integer", value)
	}
	if amount.Sign() < 0 || (!allowZero && amount.Sign() == 0) {
		if allowZero {
			return nil, fmt.Errorf("amount cannot be negative")
		}
		return nil, fmt.Errorf("amount must be positive")
	}
	if amount.BitLen() > sdkmath.MaxBitLen {
		return nil, fmt.Errorf("amount exceeds %d-bit sdk.Int limit", sdkmath.MaxBitLen)
	}
	return amount, nil
}

// ExpectedEscrowRemaining derives an order's exact outstanding liability.
func ExpectedEscrowRemaining(order *BountyOrder) (*big.Int, error) {
	if order == nil {
		return nil, fmt.Errorf("bounty order is nil")
	}
	if order.FulfilledCount > order.TargetCount {
		return nil, fmt.Errorf("fulfilled_count %d exceeds target_count %d", order.FulfilledCount, order.TargetCount)
	}
	if order.TargetCount == 0 {
		return nil, fmt.Errorf("target_count must be positive")
	}
	price, err := ParsePositiveAmount(order.PricePerArtifact)
	if err != nil {
		return nil, fmt.Errorf("invalid price_per_artifact: %w", err)
	}
	switch order.Status {
	case BountyStatus_BOUNTY_STATUS_ACTIVE, BountyStatus_BOUNTY_STATUS_EXPIRED:
		// Open/refundable liability is the unpaid slot count.
	case BountyStatus_BOUNTY_STATUS_FULFILLED:
		if order.FulfilledCount != order.TargetCount {
			return nil, fmt.Errorf("fulfilled order count %d != target_count %d", order.FulfilledCount, order.TargetCount)
		}
		return new(big.Int), nil
	case BountyStatus_BOUNTY_STATUS_CANCELED:
		return new(big.Int), nil
	default:
		return nil, fmt.Errorf("invalid bounty status %s", order.Status)
	}
	remainingSlots := new(big.Int).SetUint64(uint64(order.TargetCount - order.FulfilledCount))
	expected := new(big.Int).Mul(price, remainingSlots)
	if expected.BitLen() > sdkmath.MaxBitLen {
		return nil, fmt.Errorf("escrow liability exceeds %d-bit sdk.Int limit", sdkmath.MaxBitLen)
	}
	return expected, nil
}
