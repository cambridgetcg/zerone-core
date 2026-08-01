package app

// ZRN Issuance Doctrine
//
// The application has no per-account allocation constants. A deployment
// ceremony may still add balances, and the live zerone-1 genesis did so:
// 11,333 ZRN of validator collateral/gas and 2,222 ZRN of transferable
// operator float. Those 13,555 ZRN are disclosed in the genesis manifest.
//
// After genesis, all native issuance in the wired application routes through
// x/vesting_rewards.MintWithCap. Source-capable callers include:
//
//   - x/claiming_pot: bootstrap and legacy general-pot claims.
//   - x/substrate_bridge: external-work rewards minted after an
//     attestation survives its challenge rules.
//   - x/knowledge: a governance-configurable probe-bounty rate, default 0.
//   - x/tokens: governance-created emission periods, disabled while the
//     default/published activation latch is 0.
//
// The historical transaction-presence proposer reward, training-fund
// disbursements, and contribution-challenge bonus minting are release-sealed.
// InitChainer separately rejects an authored/imported bank supply above the
// 222,222,222 ZRN hard cap.
//
// This file therefore carries no per-account allocation constants —
// no founder, no AI vault, no validator, no foundation, no research-
// fund, no claiming-pots-total. That source-level property does not erase
// deployment-specific genesis balances.
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
