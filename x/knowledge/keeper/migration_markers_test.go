package keeper_test

import (
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
)

func TestMigrationMarkerValidationAndIdempotence(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	require.ErrorContains(t, k.WriteMigrationMarker(ctx, "", "migrated"), "key cannot be empty")
	require.ErrorContains(t, k.WriteMigrationMarker(ctx, "h1", ""), "value cannot be empty")
	_, err := k.ReadMigrationMarkerChecked(ctx, "")
	require.ErrorContains(t, err, "key cannot be empty")
	_, _, err = k.ReadMigrationMarkerPresenceChecked(ctx, "")
	require.ErrorContains(t, err, "key cannot be empty")

	value, err := k.ReadMigrationMarkerChecked(ctx, "absent")
	require.NoError(t, err)
	require.Empty(t, value)
	value, found, err := k.ReadMigrationMarkerPresenceChecked(ctx, "absent")
	require.NoError(t, err)
	require.Empty(t, value)
	require.False(t, found)

	require.NoError(t, k.WriteMigrationMarker(ctx, "h1", "migrated"))
	require.NoError(t, k.WriteMigrationMarker(ctx, "h1", "migrated"))
	value, err = k.ReadMigrationMarkerChecked(ctx, "h1")
	require.NoError(t, err)
	require.Equal(t, "migrated", value)
	value, found, err = k.ReadMigrationMarkerPresenceChecked(ctx, "h1")
	require.NoError(t, err)
	require.Equal(t, "migrated", value)
	require.True(t, found)

	err = k.WriteMigrationMarker(ctx, "h1", "forged")
	require.ErrorContains(t, err, "conflicts")
	require.Equal(t, "migrated", k.ReadMigrationMarker(ctx, "h1"),
		"a conflicting writer must not overwrite the first value")
}

func TestMigrationMarkerStoreErrorsPropagate(t *testing.T) {
	getErr := errors.New("get failed")
	getFailureKeeper := knowledgekeeper.NewKeeper(
		markerStoreService{store: markerErrorStore{getErr: getErr}},
		nil,
		"",
		nil,
		nil,
	)

	_, err := getFailureKeeper.ReadMigrationMarkerChecked(context.Background(), "h1")
	require.ErrorIs(t, err, getErr)
	_, _, err = getFailureKeeper.ReadMigrationMarkerPresenceChecked(context.Background(), "h1")
	require.ErrorIs(t, err, getErr)
	require.ErrorIs(t, getFailureKeeper.WriteMigrationMarker(context.Background(), "h1", "migrated"), getErr)
	require.Panics(t, func() {
		getFailureKeeper.ReadMigrationMarker(context.Background(), "h1")
	})

	setErr := errors.New("set failed")
	setFailureKeeper := knowledgekeeper.NewKeeper(
		markerStoreService{store: markerErrorStore{setErr: setErr}},
		nil,
		"",
		nil,
		nil,
	)
	require.ErrorIs(t, setFailureKeeper.WriteMigrationMarker(context.Background(), "h1", "migrated"), setErr)
}

func TestMigrationMarkerPresenceDistinguishesHistoricalEmptyValue(t *testing.T) {
	emptyValueKeeper := knowledgekeeper.NewKeeper(
		markerStoreService{store: markerErrorStore{getValue: []byte{}}},
		nil,
		"",
		nil,
		nil,
	)

	value, found, err := emptyValueKeeper.ReadMigrationMarkerPresenceChecked(context.Background(), "h1")
	require.NoError(t, err)
	require.Empty(t, value)
	require.True(t, found,
		"a present legacy empty value must not be mistaken for an absent marker")
}

type markerStoreService struct {
	store corestore.KVStore
}

func (s markerStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return s.store
}

type markerErrorStore struct {
	getValue []byte
	getErr   error
	setErr   error
}

func (s markerErrorStore) Get([]byte) ([]byte, error) {
	return s.getValue, s.getErr
}

func (s markerErrorStore) Has([]byte) (bool, error) {
	return false, nil
}

func (s markerErrorStore) Set([]byte, []byte) error {
	return s.setErr
}

func (s markerErrorStore) Delete([]byte) error {
	return nil
}

func (s markerErrorStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("iterator unsupported")
}

func (s markerErrorStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("reverse iterator unsupported")
}
