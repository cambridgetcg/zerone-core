package app

// DET-2 wiring: the Unicode table refusal must run when the application is
// constructed, before any store is opened. The guard's own logic is tested in
// x/knowledge/types; these tests fail if newZeroneApp stops calling it.

import (
	"errors"
	"reflect"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	zeroneknowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestUnicodeGuardIsTheKnowledgeTypesCheck(t *testing.T) {
	require.Equal(t,
		reflect.ValueOf(zeroneknowledgetypes.ValidateUnicodeTables).Pointer(),
		reflect.ValueOf(validateUnicodeTables).Pointer(),
		"newZeroneApp must run x/knowledge/types.ValidateUnicodeTables")
}

func TestConstructorsRefuseOtherUnicodeTablesBeforeOpeningStores(t *testing.T) {
	refusal := errors.New("unicode tables differ (test)")
	validateUnicodeTables = func() error { return refusal }
	t.Cleanup(func() { validateUnicodeTables = zeroneknowledgetypes.ValidateUnicodeTables })

	for _, tc := range []struct {
		name      string
		construct func(db dbm.DB, home string) *ZeroneApp
	}{
		{"NewZeroneApp", func(db dbm.DB, home string) *ZeroneApp {
			return NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(home))
		}},
		{"NewActivationPreflightApp", func(db dbm.DB, home string) *ZeroneApp {
			return NewActivationPreflightApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(home))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				tc.construct(db, t.TempDir())
			}()
			require.NotNil(t, recovered, "constructor returned instead of refusing")
			err, ok := recovered.(error)
			require.True(t, ok, "panic value %v is not an error", recovered)
			require.ErrorIs(t, err, refusal)
			require.Contains(t, err.Error(), "refusing to start")

			it, iterErr := db.Iterator(nil, nil)
			require.NoError(t, iterErr)
			defer it.Close()
			require.False(t, it.Valid(), "the refused constructor wrote to the database")
		})
	}
}
