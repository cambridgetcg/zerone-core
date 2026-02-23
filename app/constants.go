package app

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

	// ─── Testnet genesis constants ──────────────────────────────────────────

	// TestnetChainID is the chain ID for the first public testnet.
	TestnetChainID = "zerone-testnet-1"

	// MicroDenomMultiplier converts 1 ZRN to uzrn (1 ZRN = 1,000,000 uzrn).
	MicroDenomMultiplier = 1_000_000

	// TotalSupplyZRN is the total ZRN supply at genesis (222,222,222,222).
	TotalSupplyZRN = 222_222_222_222

	// TotalSupplyUZRN is the total supply in micro-denomination.
	TotalSupplyUZRN = TotalSupplyZRN * MicroDenomMultiplier // 222,222,222,222,000,000

	// ─── Token allocation (in ZRN) ──────────────────────────────────────────

	// ResearchFundZRN is the research fund allocation (20%).
	ResearchFundZRN = 44_444_444_444

	// FounderZRN is the founder allocation (10%).
	FounderZRN = 22_222_222_222

	// AIAgentZRN is the AI agent allocation (10%).
	AIAgentZRN = 22_222_222_222

	// ValidatorZRN is the per-validator allocation (10% each, 4 validators = 40%).
	ValidatorZRN = 22_222_222_222

	// ValidatorCount is the number of genesis validators.
	ValidatorCount = 4

	// ClaimingPotsZRN is the claiming pots allocation (20% + 2 ZRN rounding remainder).
	ClaimingPotsZRN = 44_444_444_446
)
