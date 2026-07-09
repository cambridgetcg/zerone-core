package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultParams() *Params {
	return &Params{
		DefaultSwapFeeBps:  3000,          // 0.3%
		MaxPools:           0,             // 0 = unlimited
		MinInitialLiquidity: "10000000000", // 10,000 ZRN in uzrn (uzrn side of the pool)
		TwapWindowBlocks:   1000,          // ~42 minutes at 2.521s blocks
		ProtocolFeeBps:     450_000,       // 45% of swap fees go to protocol (fee_collector)
		MinReserve:         "1",           // minimum reserve after swap
		BillingQuoteDenoms: []string{},    // empty = ZRN price oracle disabled (fail-closed)
	}
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		Pools:            []*Pool{},
		TwapAccumulators: []*TWAPAccumulator{},
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}

func (p *Params) Validate() error {
	if p.DefaultSwapFeeBps > MaxSwapFeeBps {
		return fmt.Errorf("default_swap_fee_bps cannot exceed %d (10%%)", MaxSwapFeeBps)
	}
	// MaxPools 0 = unlimited; no validation needed
	if p.TwapWindowBlocks == 0 {
		return fmt.Errorf("twap_window_blocks must be positive")
	}
	if p.ProtocolFeeBps > 1_000_000 {
		return fmt.Errorf("protocol_fee_bps cannot exceed 1000000")
	}
	// MinInitialLiquidity and MinReserve are bigint strings consumed via
	// SetString(_, 10) in the msg server, which silently keeps a partial-parse
	// prefix on failure — a malformed gov value ("1e10", "10_000") would
	// quietly collapse the floor. Validate them here (the only gate on the
	// UpdateParams path) so a typo is rejected, not silently applied.
	if v, ok := new(big.Int).SetString(p.MinInitialLiquidity, 10); !ok || v.Sign() <= 0 {
		return fmt.Errorf("min_initial_liquidity must be a positive base-10 integer, got %q", p.MinInitialLiquidity)
	}
	if v, ok := new(big.Int).SetString(p.MinReserve, 10); !ok || v.Sign() < 0 {
		return fmt.Errorf("min_reserve must be a non-negative base-10 integer, got %q", p.MinReserve)
	}
	seen := make(map[string]struct{}, len(p.BillingQuoteDenoms))
	for _, denom := range p.BillingQuoteDenoms {
		if err := sdk.ValidateDenom(denom); err != nil {
			return fmt.Errorf("invalid billing quote denom %q: %w", denom, err)
		}
		if denom == ZRNDenom {
			return fmt.Errorf("billing_quote_denoms cannot contain %s itself", ZRNDenom)
		}
		if _, dup := seen[denom]; dup {
			return fmt.Errorf("duplicate billing quote denom %q", denom)
		}
		seen[denom] = struct{}{}
	}
	return nil
}
