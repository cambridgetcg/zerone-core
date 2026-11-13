package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit, second pass: tests added for the review findings on
// the first draft. Each names the rule it holds.

// D2, D6: before this binary blocked it, the seat-deposit address was an
// ordinary address, so a send of one uzrn could create a plain account
// there. The first deposit turns that account into the seat-deposit module
// account, keeping its account number, instead of refusing every seat on the
// chain; whatever it held stays as surplus. A later deposit converts nothing.
func TestSeatDepositConvertsAPlainAccountAtItsAddress(t *testing.T) {
	address := seatModuleAddress()
	f := newSeatFixture(t, nil)
	f.accounts.accounts[address.String()] = authtypes.NewBaseAccount(address, nil, 42, 0)
	round := f.claim(t, "squat-plain")
	first := f.reviewer("seat-squat-plain-1", 100_000_000)
	require.NoError(t, f.commit(t, round, first, "accept"))
	module, ok := f.accounts.accounts[address.String()].(sdk.ModuleAccountI)
	require.True(t, ok, "the plain account is now the module account")
	require.Equal(t, types.SeatDepositModuleName, module.GetName())
	require.Equal(t, address, module.GetAddress())
	require.Equal(t, uint64(42), module.GetAccountNumber(), "the account number is kept")
	require.Empty(t, module.GetPermissions())
	require.Nil(t, module.GetPubKey())
	require.Zero(t, module.GetSequence())
	require.Equal(t, int64(100_000_000), f.bank.custody())
	require.Len(t, f.round(t, round.Id).Commits, 1)

	second := f.reviewer("seat-squat-plain-2", 100_000_000)
	require.NoError(t, f.commit(t, round, second, "accept"))
	require.Same(t, module, f.accounts.accounts[address.String()], "a module account is never rebuilt")
	require.Equal(t, int64(200_000_000), f.bank.custody())
}

// D2, D6: an earlier binary also let anyone create a vesting account at the
// seat-deposit address, with MsgCreateVestingAccount (continuous or delayed),
// MsgCreatePeriodicVestingAccount or MsgCreatePermanentLockedAccount, each for
// one uzrn. No key signs for the address, so the schedule binds nobody. The
// first deposit converts each of the four kinds as it converts a plain
// account, keeping the account number; what it held stays as surplus.
func TestSeatDepositConvertsAVestingAccountAtItsAddress(t *testing.T) {
	address := seatModuleAddress()
	one := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
	continuous, err := vestingtypes.NewContinuousVestingAccount(authtypes.NewBaseAccount(address, nil, 11, 0), one, 1, 2)
	require.NoError(t, err)
	delayed, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccount(address, nil, 12, 0), one, 2)
	require.NoError(t, err)
	periodic, err := vestingtypes.NewPeriodicVestingAccount(authtypes.NewBaseAccount(address, nil, 13, 0), one, 1, vestingtypes.Periods{{Length: 1, Amount: one}})
	require.NoError(t, err)
	locked, err := vestingtypes.NewPermanentLockedAccount(authtypes.NewBaseAccount(address, nil, 14, 0), one)
	require.NoError(t, err)
	for name, squatter := range map[string]sdk.AccountI{
		"continuous": continuous,
		"delayed":    delayed,
		"periodic":   periodic,
		"locked":     locked,
	} {
		t.Run(name, func(t *testing.T) {
			f := newSeatFixture(t, nil)
			f.accounts.accounts[address.String()] = squatter
			round := f.claim(t, "squat-vesting")
			require.NoError(t, f.commit(t, round, f.reviewer("seat-squat-vesting", 100_000_000), "accept"))
			module, ok := f.accounts.accounts[address.String()].(sdk.ModuleAccountI)
			require.True(t, ok, "the vesting account is now the module account")
			require.Equal(t, types.SeatDepositModuleName, module.GetName())
			require.Equal(t, squatter.GetAccountNumber(), module.GetAccountNumber(), "the account number is kept")
			require.Empty(t, module.GetPermissions())
			require.Equal(t, int64(100_000_000), f.bank.custody())
			require.Len(t, f.round(t, round.Id).Commits, 1)
		})
	}
}

// D2: an account at the seat-deposit address that is neither its module
// account nor a plain or vesting account that no key has signed for is
// refused, not converted, and nothing moves or is stored. No key signs for a
// module address, so such an account is not expected; the refusal keeps the
// bank from panicking on it.
func TestSeatDepositRefusesAnAccountItCannotConvert(t *testing.T) {
	address := seatModuleAddress()
	signed := authtypes.NewBaseAccount(address, nil, 7, 1)
	vesting, err := vestingtypes.NewContinuousVestingAccount(authtypes.NewBaseAccount(address, nil, 8, 1), sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1)), 1, 2)
	require.NoError(t, err)
	for name, squatter := range map[string]sdk.AccountI{
		"signed":         signed,
		"signed vesting": vesting,
		"other module":   authtypes.NewModuleAccount(authtypes.NewBaseAccount(address, nil, 9, 0), "not_seat_deposits"),
	} {
		t.Run(name, func(t *testing.T) {
			f := newSeatFixture(t, nil)
			f.accounts.accounts[address.String()] = squatter
			round := f.claim(t, "squat-refused")
			reviewer := f.reviewer("seat-squat-refused", 100_000_000)
			require.ErrorContains(t, f.commit(t, round, reviewer, "accept"), "cannot become its module account")
			require.Same(t, squatter, f.accounts.accounts[address.String()])
			require.Empty(t, f.round(t, round.Id).Commits)
			require.Equal(t, int64(100_000_000), f.bank.balance(reviewer))
			require.Zero(t, f.bank.custody())
		})
	}
}

// D6 at import: a plain, never-signed account at the seat-deposit address does
// not block an import with the marker on; the first deposit converts it. An
// account that cannot be converted is refused before any write.
func TestSeatDepositImportAcceptsAConvertibleAccount(t *testing.T) {
	address := seatModuleAddress()
	gs := types.DefaultGenesis()
	gs.SeatDepositEnabled = true
	plain := newKnowledgeStoreFixture(t, "seat-import-plain")
	plain.k.SetAccountKeeper(&seatAccounts{accounts: map[string]sdk.AccountI{
		address.String(): authtypes.NewBaseAccountWithAddress(address),
	}})
	require.NoError(t, plain.k.InitGenesis(plain.ctx, gs))

	signed := newKnowledgeStoreFixture(t, "seat-import-signed")
	signed.k.SetAccountKeeper(&seatAccounts{accounts: map[string]sdk.AccountI{
		address.String(): authtypes.NewBaseAccount(address, nil, 3, 5),
	}})
	require.ErrorContains(t, signed.k.InitGenesis(signed.ctx, gs), "cannot become its module account")
	it := signed.ctx.KVStore(signed.key).Iterator(nil, nil)
	defer it.Close()
	require.False(t, it.Valid(), "a refused import writes nothing")
}

// Section 7 and D6 at activation: EnableSeatDeposit, the function a carrier
// looping over the v3 registry would call, refuses to turn the deposit on
// without each predecessor marker (the reviewer-balance parameter, review
// neutrality, record integrity), with an account at the seat-deposit address
// that the first deposit cannot convert, or with less in custody than the
// stored rounds hold locked. Each refusal writes no marker. Once the marker is
// on, Enable checks nothing and stays idempotent.
func TestSeatDepositEnableRequiresItsPredecessorsAndCustody(t *testing.T) {
	off := func(gs *types.GenesisState) { gs.SeatDepositEnabled = false }
	requireRefused := func(t *testing.T, f *seatFixture, want string) {
		t.Helper()
		require.ErrorContains(t, f.k.EnableSeatDeposit(f.ctx), want)
		require.Nil(t, f.ctx.KVStore(f.key).Get(types.SeatDepositEnabledKey), "a refused enable writes no marker")
	}
	for name, key := range map[string][]byte{
		"reviewer_balance_param_enabled": types.ReviewerBalanceParamEnabledKey,
		"review_neutrality_enabled":      []byte(keeper.ReviewNeutralityEnabledStoreKey),
		"record_integrity_enabled":       []byte(keeper.RecordIntegrityEnabledStoreKey),
	} {
		t.Run("without "+name, func(t *testing.T) {
			f := newSeatFixture(t, off)
			f.ctx.KVStore(f.key).Delete(key)
			requireRefused(t, f, "seat_deposit_enabled requires "+name)
		})
	}
	t.Run("an account it cannot convert", func(t *testing.T) {
		f := newSeatFixture(t, off)
		address := seatModuleAddress()
		f.accounts.accounts[address.String()] = authtypes.NewBaseAccount(address, nil, 5, 1)
		requireRefused(t, f, "cannot become its module account")
	})
	t.Run("a custody shortfall", func(t *testing.T) {
		f := newSeatFixture(t, nil)
		round := f.claim(t, "enable-custody")
		require.NoError(t, f.commit(t, round, f.reviewer("seat-enable-custody", 100_000_000), "accept"))
		f.ctx.KVStore(f.key).Delete(types.SeatDepositEnabledKey)
		f.bank.moduleBalances[types.SeatDepositModuleName] = sdk.NewCoins()
		requireRefused(t, f, "less than the 100000000 uzrn of locked, unreturned deposits")
	})
	t.Run("everything in place", func(t *testing.T) {
		f := newSeatFixture(t, off)
		address := seatModuleAddress()
		f.accounts.accounts[address.String()] = authtypes.NewBaseAccount(address, nil, 6, 0)
		require.NoError(t, f.k.EnableSeatDeposit(f.ctx))
		require.Equal(t, []byte{1}, f.ctx.KVStore(f.key).Get(types.SeatDepositEnabledKey))
		_, plain := f.accounts.accounts[address.String()].(*authtypes.BaseAccount)
		require.True(t, plain, "enable converts nothing; the first deposit does")
		f.ctx.KVStore(f.key).Delete(types.ReviewerBalanceParamEnabledKey)
		require.NoError(t, f.k.EnableSeatDeposit(f.ctx), "with the marker on, enable checks nothing")
	})
}
