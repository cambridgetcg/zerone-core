package app

// ZRN Issuance Doctrine
//
// The chain has no per-account allocation constants. ZRN enters
// circulation through two participation-gated emission pathways:
//
//   - x/vesting_rewards: PoT block rewards minted to validators
//     verifying truth (decay curve, floor, validator scaling).
//   - x/claiming_pot: bootstrap claims minted on demand to
//     whitelisted agents (0.222 ZRN each).
//
// Both pathways gate through MintWithCap against the hard cap
// of 222,222,222 ZRN (see x/vesting_rewards/types/keys.go:MaxSupplyUzrn).
// Neither grants anyone a privileged starting balance.
//
// This file therefore carries no per-account allocation constants —
// no founder, no AI vault, no validator, no foundation, no research-
// fund, no claiming-pots-total. Issuance follows participation; the
// doctrine refuses any other model.
//
// Full doctrine: docs/tokenomics/GENESIS.md.

const (
	// AppName is the application name.
	AppName = "zeroned"

	// AccountAddressPrefix is the bech32 prefix for Zerone addresses.
	AccountAddressPrefix = "zrn"

	// BondDenom is the staking denomination.
	BondDenom = "uzrn"

	// DisplayDenom is the human-readable denomination.
	DisplayDenom = "zrn"

	// DefaultBlockTime is the target block time in milliseconds.
	DefaultBlockTime = 2521

	// MicroDenomMultiplier converts 1 ZRN to uzrn (1 ZRN = 1,000,000 uzrn).
	MicroDenomMultiplier = 1_000_000

	// TestnetChainID is the chain ID for the first public testnet.
	TestnetChainID = "zerone-testnet-1"
)
