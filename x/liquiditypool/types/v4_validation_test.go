package types_test

import (
	"bytes"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

func v4TestAddress(seed byte) string {
	return sdk.AccAddress(bytes.Repeat([]byte{seed}, 20)).String()
}

func v4Max256BitAmount() string {
	max := new(big.Int).Lsh(big.NewInt(1), 256)
	max.Sub(max, big.NewInt(1))
	return max.String()
}

func v4Over256BitAmount() string {
	return new(big.Int).Lsh(big.NewInt(1), 256).String()
}

func v4AssertValidationErrorWithoutPanic(t *testing.T, validate func() error) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("validation panicked: %v", recovered)
		}
	}()
	if err := validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func v4ValidCreatePool() *types.MsgCreatePool {
	return &types.MsgCreatePool{
		Creator:    v4TestAddress(1),
		DenomA:     types.ZRNDenom,
		DenomB:     "uatom",
		AmountA:    "10000000000",
		AmountB:    "1000000",
		SwapFeeBps: 0,
	}
}

func v4ValidSwap() *types.MsgSwap {
	return &types.MsgSwap{
		Sender:        v4TestAddress(2),
		PoolId:        "pool-1",
		TokenInDenom:  types.ZRNDenom,
		TokenInAmount: "1000000",
		MinTokenOut:   "1",
	}
}

func v4ValidAddLiquidity() *types.MsgAddLiquidity {
	return &types.MsgAddLiquidity{
		Sender:      v4TestAddress(3),
		PoolId:      "pool-1",
		AmountA:     "1000000",
		AmountB:     "2000000",
		MinLpTokens: "1",
	}
}

func v4ValidRemoveLiquidity() *types.MsgRemoveLiquidity {
	return &types.MsgRemoveLiquidity{
		Sender:     v4TestAddress(4),
		PoolId:     "pool-1",
		LpTokens:   "1000",
		MinAmountA: "1",
		MinAmountB: "1",
	}
}

func TestV4MessageAmountsRequireCanonicalPositiveUint256(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		"0",
		"-0",
		"-1",
		"+1",
		"01",
		" 1",
		"1 ",
		"1_000",
		"1e3",
		"1.0",
		"１",
		"abc",
		v4Over256BitAmount(),
	}

	fields := []struct {
		name     string
		optional bool
		validate func(string) error
	}{
		{
			name: "create amount_a",
			validate: func(value string) error {
				msg := v4ValidCreatePool()
				msg.AmountA = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "create amount_b",
			validate: func(value string) error {
				msg := v4ValidCreatePool()
				msg.AmountB = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "swap token_in_amount",
			validate: func(value string) error {
				msg := v4ValidSwap()
				msg.TokenInAmount = value
				return msg.ValidateBasic()
			},
		},
		{
			name:     "swap min_token_out",
			optional: true,
			validate: func(value string) error {
				msg := v4ValidSwap()
				msg.MinTokenOut = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "add amount_a",
			validate: func(value string) error {
				msg := v4ValidAddLiquidity()
				msg.AmountA = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "add amount_b",
			validate: func(value string) error {
				msg := v4ValidAddLiquidity()
				msg.AmountB = value
				return msg.ValidateBasic()
			},
		},
		{
			name:     "add min_lp_tokens",
			optional: true,
			validate: func(value string) error {
				msg := v4ValidAddLiquidity()
				msg.MinLpTokens = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "remove lp_tokens",
			validate: func(value string) error {
				msg := v4ValidRemoveLiquidity()
				msg.LpTokens = value
				return msg.ValidateBasic()
			},
		},
		{
			name:     "remove min_amount_a",
			optional: true,
			validate: func(value string) error {
				msg := v4ValidRemoveLiquidity()
				msg.MinAmountA = value
				return msg.ValidateBasic()
			},
		},
		{
			name:     "remove min_amount_b",
			optional: true,
			validate: func(value string) error {
				msg := v4ValidRemoveLiquidity()
				msg.MinAmountB = value
				return msg.ValidateBasic()
			},
		},
	}

	for _, field := range fields {
		field := field
		t.Run(field.name, func(t *testing.T) {
			t.Parallel()
			for _, value := range invalid {
				if field.optional && value == "" {
					continue
				}
				value := value
				t.Run(value, func(t *testing.T) {
					t.Parallel()
					v4AssertValidationErrorWithoutPanic(t, func() error {
						return field.validate(value)
					})
				})
			}
		})
	}
}

func TestV4OptionalMinimumsAllowOnlyEmptyOrCanonicalPositiveUint256(t *testing.T) {
	t.Parallel()

	max := v4Max256BitAmount()
	tests := []struct {
		name     string
		validate func(string) error
	}{
		{
			name: "swap min_token_out",
			validate: func(value string) error {
				msg := v4ValidSwap()
				msg.MinTokenOut = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "add min_lp_tokens",
			validate: func(value string) error {
				msg := v4ValidAddLiquidity()
				msg.MinLpTokens = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "remove min_amount_a",
			validate: func(value string) error {
				msg := v4ValidRemoveLiquidity()
				msg.MinAmountA = value
				return msg.ValidateBasic()
			},
		},
		{
			name: "remove min_amount_b",
			validate: func(value string) error {
				msg := v4ValidRemoveLiquidity()
				msg.MinAmountB = value
				return msg.ValidateBasic()
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, value := range []string{"", "1", max} {
				if err := tt.validate(value); err != nil {
					t.Errorf("canonical value %q rejected: %v", value, err)
				}
			}
		})
	}
}

func TestV4RequiredAmountsAcceptUint256Boundary(t *testing.T) {
	t.Parallel()

	max := v4Max256BitAmount()
	tests := []struct {
		name     string
		validate func() error
	}{
		{
			name: "create",
			validate: func() error {
				msg := v4ValidCreatePool()
				msg.AmountA = max
				msg.AmountB = max
				return msg.ValidateBasic()
			},
		},
		{
			name: "swap",
			validate: func() error {
				msg := v4ValidSwap()
				msg.TokenInAmount = max
				return msg.ValidateBasic()
			},
		},
		{
			name: "add",
			validate: func() error {
				msg := v4ValidAddLiquidity()
				msg.AmountA = max
				msg.AmountB = max
				return msg.ValidateBasic()
			},
		},
		{
			name: "remove",
			validate: func() error {
				msg := v4ValidRemoveLiquidity()
				msg.LpTokens = max
				return msg.ValidateBasic()
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("validation panicked at uint256 boundary: %v", recovered)
				}
			}()
			if err := tt.validate(); err != nil {
				t.Fatalf("uint256 boundary rejected: %v", err)
			}
		})
	}
}

func TestV4MessageDenomsUseCosmosValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		validate func() error
	}{
		{
			name: "create denom_a",
			validate: func() error {
				msg := v4ValidCreatePool()
				msg.DenomA = "bad denom"
				return msg.ValidateBasic()
			},
		},
		{
			name: "create denom_b",
			validate: func() error {
				msg := v4ValidCreatePool()
				msg.DenomB = "!"
				return msg.ValidateBasic()
			},
		},
		{
			name: "swap token_in_denom",
			validate: func() error {
				msg := v4ValidSwap()
				msg.TokenInDenom = "not a denom"
				return msg.ValidateBasic()
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v4AssertValidationErrorWithoutPanic(t, tt.validate)
		})
	}
}

func TestV4EconomicParamsUseCanonicalBoundedAmounts(t *testing.T) {
	t.Parallel()

	over256 := v4Over256BitAmount()
	tests := []struct {
		name   string
		values []string
		mutate func(*types.Params, string)
	}{
		{
			name:   "minimum initial liquidity is positive",
			values: []string{"", "0", "-1", "+1", "01", "1e3", " 1", "1 ", over256},
			mutate: func(params *types.Params, value string) {
				params.MinInitialLiquidity = value
			},
		},
		{
			name:   "minimum reserve is nonnegative",
			values: []string{"", "-1", "+1", "01", "1e3", " 1", "1 ", over256},
			mutate: func(params *types.Params, value string) {
				params.MinReserve = value
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, value := range tt.values {
				value := value
				t.Run(value, func(t *testing.T) {
					t.Parallel()
					params := types.DefaultParams()
					tt.mutate(params, value)
					v4AssertValidationErrorWithoutPanic(t, params.Validate)
				})
			}
		})
	}

	for name, mutate := range map[string]func(*types.Params){
		"initial liquidity uint256 maximum": func(params *types.Params) {
			params.MinInitialLiquidity = v4Max256BitAmount()
		},
		"reserve zero": func(params *types.Params) {
			params.MinReserve = "0"
		},
		"reserve uint256 maximum": func(params *types.Params) {
			params.MinReserve = v4Max256BitAmount()
		},
	} {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := types.DefaultParams()
			mutate(params)
			if err := params.Validate(); err != nil {
				t.Fatalf("valid boundary rejected: %v", err)
			}
		})
	}
}

func TestV4EconomicParamDenomAllowlistsUseCosmosValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*types.Params)
	}{
		{
			name: "invalid billing quote denom",
			mutate: func(params *types.Params) {
				params.BillingQuoteDenoms = []string{"bad denom"}
			},
		},
		{
			name: "invalid admitted pool denom",
			mutate: func(params *types.Params) {
				params.AllowedPoolDenoms = []string{"!"}
			},
		},
		{
			name: "duplicate admitted pool denom",
			mutate: func(params *types.Params) {
				params.AllowedPoolDenoms = []string{"uatom", "uatom"}
			},
		},
		{
			name: "native denom cannot be its own counter",
			mutate: func(params *types.Params) {
				params.AllowedPoolDenoms = []string{types.ZRNDenom}
			},
		},
		{
			name: "invalid pool creator",
			mutate: func(params *types.Params) {
				params.PoolCreators = []string{"not-an-address"}
			},
		},
		{
			name: "duplicate pool creator",
			mutate: func(params *types.Params) {
				creator := v4TestAddress(9)
				params.PoolCreators = []string{creator, creator}
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := types.DefaultParams()
			tt.mutate(params)
			v4AssertValidationErrorWithoutPanic(t, params.Validate)
		})
	}
}

func v4ValidGenesisState() *types.GenesisState {
	return &types.GenesisState{
		Params: types.DefaultParams(),
		Pools: []*types.Pool{
			{
				PoolId:         "pool-1",
				DenomA:         types.ZRNDenom,
				DenomB:         "uatom",
				ReserveA:       "10000000000",
				ReserveB:       "1000000",
				SwapFeeBps:     3_000,
				LpTokenSupply:  "100000000",
				LpDenom:        types.LPDenom("pool-1"),
				Creator:        v4TestAddress(10),
				CreatedAtBlock: 100,
				Status:         types.PoolStatus_POOL_STATUS_ACTIVE,
			},
		},
		TwapAccumulators: []*types.TWAPAccumulator{
			{
				PoolId:       "pool-1",
				LastBlock:    100,
				StartBlock:   100,
				CumPriceAToB: "0",
				CumPriceBToA: "0",
			},
		},
		NextPoolId: 2,
		TwapObservations: []*types.TWAPObservation{{
			PoolId:       "pool-1",
			BlockHeight:  100,
			CumPriceAToB: "0",
			CumPriceBToA: "0",
		}},
	}
}

func v4CloneGenesis(gs *types.GenesisState) *types.GenesisState {
	return proto.Clone(gs).(*types.GenesisState)
}

func v4AssertGenesisErrorWithoutPanic(t *testing.T, gs *types.GenesisState) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("genesis validation panicked: %v", recovered)
		}
	}()
	if err := gs.Validate(); err == nil {
		t.Fatal("expected invalid genesis to be rejected")
	}
}

func TestV4DefaultAndRepresentativeGenesisValidate(t *testing.T) {
	t.Parallel()

	for name, genesis := range map[string]*types.GenesisState{
		"default":       types.DefaultGenesis(),
		"one open pool": v4ValidGenesisState(),
	} {
		genesis := genesis
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := genesis.Validate(); err != nil {
				t.Fatalf("valid genesis rejected: %v", err)
			}
		})
	}
}

func TestV4GenesisRejectsDuplicateIDsAndOpenPairs(t *testing.T) {
	t.Parallel()

	t.Run("duplicate ID", func(t *testing.T) {
		t.Parallel()
		gs := v4ValidGenesisState()
		duplicate := proto.Clone(gs.Pools[0]).(*types.Pool)
		duplicate.DenomB = "uosmo"
		gs.Pools = append(gs.Pools, duplicate)
		v4AssertGenesisErrorWithoutPanic(t, gs)
	})

	t.Run("reversed duplicate open pair", func(t *testing.T) {
		t.Parallel()
		gs := v4ValidGenesisState()
		gs.Pools = append(gs.Pools, &types.Pool{
			PoolId:         "pool-2",
			DenomA:         "uatom",
			DenomB:         types.ZRNDenom,
			ReserveA:       "1000000",
			ReserveB:       "10000000000",
			SwapFeeBps:     3_000,
			LpTokenSupply:  "100000000",
			LpDenom:        types.LPDenom("pool-2"),
			Creator:        v4TestAddress(11),
			CreatedAtBlock: 101,
			Status:         types.PoolStatus_POOL_STATUS_ACTIVE,
		})
		gs.TwapAccumulators = append(gs.TwapAccumulators, &types.TWAPAccumulator{
			PoolId:       "pool-2",
			LastBlock:    101,
			StartBlock:   101,
			CumPriceAToB: "0",
			CumPriceBToA: "0",
		})
		gs.NextPoolId = 3
		v4AssertGenesisErrorWithoutPanic(t, gs)
	})
}

func TestV4GenesisRejectsEconomicAndLifecycleInconsistency(t *testing.T) {
	t.Parallel()

	over256 := v4Over256BitAmount()
	tests := []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{
			name: "zero reserve on open pool",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].ReserveA = "0"
			},
		},
		{
			name: "zero LP supply on open pool",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].LpTokenSupply = "0"
			},
		},
		{
			name: "noncanonical reserve",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].ReserveB = "01"
			},
		},
		{
			name: "negative reserve",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].ReserveA = "-1"
			},
		},
		{
			name: "over-256-bit supply",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].LpTokenSupply = over256
			},
		},
		{
			name: "open pool has closed height",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].ClosedAtBlock = 200
			},
		},
		{
			name: "closed pool retains reserves and supply",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].Status = types.PoolStatus_POOL_STATUS_CLOSED
				gs.Pools[0].ClosedAtBlock = 200
			},
		},
		{
			name: "closed pool predates creation",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].Status = types.PoolStatus_POOL_STATUS_CLOSED
				gs.Pools[0].ReserveA = "0"
				gs.Pools[0].ReserveB = "0"
				gs.Pools[0].LpTokenSupply = "0"
				gs.Pools[0].ClosedAtBlock = 99
				gs.TwapAccumulators = nil
			},
		},
		{
			name: "unspecified status",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].Status = types.PoolStatus_POOL_STATUS_UNSPECIFIED
			},
		},
		{
			name: "transient lock persisted",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].Locked = true
			},
		},
		{
			name: "LP denom does not derive from pool ID",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].LpDenom = types.LPDenom("pool-2")
			},
		},
		{
			name: "invalid economic denom",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].DenomB = "bad denom"
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gs := v4ValidGenesisState()
			tt.mutate(gs)
			v4AssertGenesisErrorWithoutPanic(t, gs)
		})
	}
}

func TestV4GenesisRejectsInconsistentTWAPState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{
			name: "open pool missing accumulator",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators = nil
			},
		},
		{
			name: "duplicate accumulator",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators = append(
					gs.TwapAccumulators,
					proto.Clone(gs.TwapAccumulators[0]).(*types.TWAPAccumulator),
				)
			},
		},
		{
			name: "orphan accumulator",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].PoolId = "pool-2"
			},
		},
		{
			name: "accumulator time runs backwards",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].StartBlock = 101
			},
		},
		{
			name: "accumulator predates pool creation",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].StartBlock = 99
			},
		},
		{
			name: "malformed accumulator cumulative",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].CumPriceAToB = "01"
			},
		},
		{
			name: "closed pool retains accumulator",
			mutate: func(gs *types.GenesisState) {
				gs.Pools[0].Status = types.PoolStatus_POOL_STATUS_CLOSED
				gs.Pools[0].ReserveA = "0"
				gs.Pools[0].ReserveB = "0"
				gs.Pools[0].LpTokenSupply = "0"
				gs.Pools[0].ClosedAtBlock = 200
			},
		},
		{
			name: "observation without accumulator",
			mutate: func(gs *types.GenesisState) {
				gs.TwapObservations = []*types.TWAPObservation{{
					PoolId:       "pool-2",
					BlockHeight:  100,
					CumPriceAToB: "0",
					CumPriceBToA: "0",
				}}
			},
		},
		{
			name: "observation outside accumulator range",
			mutate: func(gs *types.GenesisState) {
				gs.TwapObservations = []*types.TWAPObservation{{
					PoolId:       "pool-1",
					BlockHeight:  101,
					CumPriceAToB: "0",
					CumPriceBToA: "0",
				}}
			},
		},
		{
			name: "observation exceeds accumulator",
			mutate: func(gs *types.GenesisState) {
				gs.TwapObservations = []*types.TWAPObservation{{
					PoolId:       "pool-1",
					BlockHeight:  100,
					CumPriceAToB: "1",
					CumPriceBToA: "0",
				}}
			},
		},
		{
			name: "duplicate observation checkpoint",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].LastBlock = 110
				gs.TwapAccumulators[0].CumPriceAToB = "100"
				gs.TwapAccumulators[0].CumPriceBToA = "200"
				observation := &types.TWAPObservation{
					PoolId:       "pool-1",
					BlockHeight:  105,
					CumPriceAToB: "50",
					CumPriceBToA: "100",
				}
				gs.TwapObservations = []*types.TWAPObservation{
					observation,
					proto.Clone(observation).(*types.TWAPObservation),
				}
			},
		},
		{
			name: "observation cumulatives decrease",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].LastBlock = 110
				gs.TwapAccumulators[0].CumPriceAToB = "100"
				gs.TwapAccumulators[0].CumPriceBToA = "200"
				gs.TwapObservations = []*types.TWAPObservation{
					{
						PoolId:       "pool-1",
						BlockHeight:  105,
						CumPriceAToB: "90",
						CumPriceBToA: "190",
					},
					{
						PoolId:       "pool-1",
						BlockHeight:  110,
						CumPriceAToB: "80",
						CumPriceBToA: "200",
					},
				}
			},
		},
		{
			name: "observations omit accumulator checkpoint",
			mutate: func(gs *types.GenesisState) {
				gs.TwapAccumulators[0].LastBlock = 110
				gs.TwapAccumulators[0].CumPriceAToB = "100"
				gs.TwapAccumulators[0].CumPriceBToA = "200"
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gs := v4ValidGenesisState()
			tt.mutate(gs)
			v4AssertGenesisErrorWithoutPanic(t, gs)
		})
	}
}

func TestV4GenesisCounterCannotReusePoolOrLPIdentity(t *testing.T) {
	t.Parallel()

	for name, nextPoolID := range map[string]uint64{
		"equal to maximum ID": 1,
	} {
		nextPoolID := nextPoolID
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gs := v4ValidGenesisState()
			gs.NextPoolId = nextPoolID
			v4AssertGenesisErrorWithoutPanic(t, gs)
		})
	}

	t.Run("legacy zero counter is an accepted derivation sentinel", func(t *testing.T) {
		t.Parallel()
		gs := v4ValidGenesisState()
		gs.NextPoolId = 0
		if err := gs.Validate(); err != nil {
			t.Fatalf("legacy next_pool_id sentinel rejected: %v", err)
		}
	})

	t.Run("pool ID exceeds lifetime record cap", func(t *testing.T) {
		t.Parallel()
		gs := v4ValidGenesisState()
		gs.Pools[0].PoolId = "pool-10001"
		gs.Pools[0].LpDenom = types.LPDenom(gs.Pools[0].PoolId)
		gs.NextPoolId = 10_002
		v4AssertGenesisErrorWithoutPanic(t, gs)
	})

	t.Run("next ID exceeds lifetime record boundary", func(t *testing.T) {
		t.Parallel()
		gs := v4ValidGenesisState()
		gs.NextPoolId = types.MaxPoolRecordsCap + 2
		v4AssertGenesisErrorWithoutPanic(t, gs)
	})
}

func TestV4GenesisAllowsClosedTombstoneAndGovernedReplacementPair(t *testing.T) {
	t.Parallel()

	gs := v4ValidGenesisState()
	closed := gs.Pools[0]
	closed.Status = types.PoolStatus_POOL_STATUS_CLOSED
	closed.ReserveA = "0"
	closed.ReserveB = "0"
	closed.LpTokenSupply = "0"
	closed.ClosedAtBlock = 200

	replacement := &types.Pool{
		PoolId:         "pool-2",
		DenomA:         "uatom",
		DenomB:         types.ZRNDenom,
		ReserveA:       "2000000",
		ReserveB:       "20000000000",
		SwapFeeBps:     3_000,
		LpTokenSupply:  "200000000",
		LpDenom:        types.LPDenom("pool-2"),
		Creator:        v4TestAddress(11),
		CreatedAtBlock: 201,
		Status:         types.PoolStatus_POOL_STATUS_ACTIVE,
	}
	gs.Pools = append(gs.Pools, replacement)
	gs.TwapAccumulators = []*types.TWAPAccumulator{{
		PoolId:       replacement.PoolId,
		LastBlock:    replacement.CreatedAtBlock,
		StartBlock:   replacement.CreatedAtBlock,
		CumPriceAToB: "0",
		CumPriceBToA: "0",
	}}
	gs.TwapObservations = []*types.TWAPObservation{{
		PoolId:       replacement.PoolId,
		BlockHeight:  replacement.CreatedAtBlock,
		CumPriceAToB: "0",
		CumPriceBToA: "0",
	}}
	gs.NextPoolId = 3

	if err := gs.Validate(); err != nil {
		t.Fatalf("valid tombstone plus replacement rejected: %v", err)
	}
	if closed.LpDenom == replacement.LpDenom {
		t.Fatalf("replacement reused tombstone LP denom %q", closed.LpDenom)
	}
}
