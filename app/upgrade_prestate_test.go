package app

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/iavl"
	iavldb "github.com/cosmos/iavl/db"
	"github.com/stretchr/testify/require"
)

func TestActiveSDKGovQueueProposalDeadlinesRequireCanonicalPairEncoding(
	t *testing.T,
) {
	deadline := time.Unix(1_900_000_000, 123).UTC()
	const proposalID uint64 = 17
	pairCodec := collections.PairKeyCodec(sdk.TimeKey, collections.Uint64Key)
	pair := collections.Join(deadline, proposalID)
	suffix := make([]byte, pairCodec.Size(pair))
	written, err := pairCodec.Encode(suffix, pair)
	require.NoError(t, err)
	suffix = suffix[:written]
	value, err := collections.Uint64Value.Encode(proposalID)
	require.NoError(t, err)
	record := upgradePrestateRecord{
		Key: append(
			bytes.Clone(govtypes.ActiveProposalQueuePrefix.Bytes()),
			suffix...,
		),
		Value: value,
	}

	got, err := activeSDKGovQueueProposalDeadlines(
		[]upgradePrestateRecord{record},
	)
	require.NoError(t, err)
	require.Equal(t, deadline, got[proposalID])

	malformed := upgradePrestateRecord{
		Key:   append(bytes.Clone(record.Key), 0x00),
		Value: bytes.Clone(record.Value),
	}
	_, err = activeSDKGovQueueProposalDeadlines(
		[]upgradePrestateRecord{malformed},
	)
	require.ErrorContains(t, err, "read=")

	mismatchedValue, err := collections.Uint64Value.Encode(proposalID + 1)
	require.NoError(t, err)
	mismatched := upgradePrestateRecord{
		Key:   bytes.Clone(record.Key),
		Value: mismatchedValue,
	}
	_, err = activeSDKGovQueueProposalDeadlines(
		[]upgradePrestateRecord{mismatched},
	)
	require.ErrorContains(t, err, "does not match")
}

func TestVerifyIAVLExportReconstructsCommittedRootAndSelectsRecords(
	t *testing.T,
) {
	tree := iavl.NewMutableTree(
		iavldb.NewMemDB(),
		0,
		true,
		iavl.NewNopLogger(),
		iavl.AsyncPruningOption(false),
	)
	for _, record := range []upgradePrestateRecord{
		{Key: []byte{0x01, 'a'}, Value: []byte("first")},
		{Key: []byte{0x02, 'b'}, Value: []byte("second")},
		{Key: []byte{0x02, 'c'}, Value: []byte{}},
		{Key: []byte{0x03, 'd'}, Value: []byte("fourth")},
	} {
		_, err := tree.Set(record.Key, record.Value)
		require.NoError(t, err)
	}
	root, _, err := tree.SaveVersion()
	require.NoError(t, err)
	require.Len(t, root, 32)
	t.Cleanup(func() {
		require.NoError(t, tree.Close())
	})

	nodes := exportIAVLNodesForTest(t, tree)
	selected, reconstructed, err := verifyIAVLExport(
		nodes,
		func(key, _ []byte) (bool, error) {
			return bytes.HasPrefix(key, []byte{0x02}), nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, root, reconstructed)
	require.Equal(t, []upgradePrestateRecord{
		{Key: []byte{0x02, 'b'}, Value: []byte("second")},
		{Key: []byte{0x02, 'c'}, Value: []byte{}},
	}, selected)

	truncated := nodes[:len(nodes)-1]
	_, truncatedRoot, err := verifyIAVLExport(
		truncated,
		func(_, _ []byte) (bool, error) { return false, nil },
	)
	if err == nil {
		require.NotEqual(
			t,
			root,
			truncatedRoot,
			"a silently truncated export must never reconstruct the committed root",
		)
	}

	tampered := cloneExportNodes(nodes)
	for _, node := range tampered {
		if node.Height == 0 {
			node.Value = append(bytes.Clone(node.Value), '!')
			break
		}
	}
	_, tamperedRoot, err := verifyIAVLExport(
		tampered,
		func(_, _ []byte) (bool, error) { return false, nil },
	)
	require.NoError(t, err)
	require.NotEqual(t, root, tamperedRoot)
}

func TestVerifyIAVLExportRejectsMalformedStructure(t *testing.T) {
	_, _, err := verifyIAVLExport(
		nil,
		func(_, _ []byte) (bool, error) { return false, nil },
	)
	require.ErrorContains(t, err, "incomplete")
	_, _, err = verifyIAVLExport(nil, nil)
	require.ErrorContains(t, err, "selector")

	tree := iavl.NewMutableTree(
		iavldb.NewMemDB(),
		0,
		true,
		iavl.NewNopLogger(),
		iavl.AsyncPruningOption(false),
	)
	for _, key := range [][]byte{[]byte("a"), []byte("b")} {
		_, setErr := tree.Set(key, key)
		require.NoError(t, setErr)
	}
	_, _, err = tree.SaveVersion()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tree.Close())
	})
	nodes := exportIAVLNodesForTest(t, tree)
	require.GreaterOrEqual(t, len(nodes), 3)
	nodes[len(nodes)-1].Key = []byte("wrong-separator")
	_, _, err = verifyIAVLExport(
		nodes,
		func(_, _ []byte) (bool, error) { return false, nil },
	)
	require.ErrorContains(t, err, "separator key")
}

func TestActivationCommitInfoBindsLoadedSubstoreRoot(t *testing.T) {
	const version = int64(42)
	emergencyHash := bytes.Repeat([]byte{1}, 32)
	govHash := bytes.Repeat([]byte{2}, 32)
	customGovHash := bytes.Repeat([]byte{3}, 32)
	info := &storetypes.CommitInfo{
		Version: version,
		StoreInfos: []storetypes.StoreInfo{
			{
				Name: "emergency",
				CommitId: storetypes.CommitID{
					Version: version,
					Hash:    emergencyHash,
				},
			},
			{
				Name: "gov",
				CommitId: storetypes.CommitID{
					Version: version,
					Hash:    govHash,
				},
			},
			{
				Name: "zerone_gov",
				CommitId: storetypes.CommitID{
					Version: version,
					Hash:    customGovHash,
				},
			},
		},
	}
	rootID := storetypes.CommitID{Version: version, Hash: info.Hash()}
	commitIDs, err := validateActivationRootCommitInfo(
		rootID,
		info,
		[]string{"emergency", "gov", "zerone_gov"},
	)
	require.NoError(t, err)
	require.Equal(t, emergencyHash, commitIDs["emergency"].Hash)

	tamperedLoaded := storetypes.CommitID{
		Version: version,
		Hash:    bytes.Repeat([]byte{9}, 32),
	}
	err = validateActivationSubstoreCommitID(
		"emergency",
		tamperedLoaded,
		commitIDs["emergency"],
	)
	require.ErrorContains(t, err, "not the subroot bound by root CommitInfo")

	badRootID := rootID
	badRootID.Hash = bytes.Repeat([]byte{8}, 32)
	_, err = validateActivationRootCommitInfo(
		badRootID,
		info,
		[]string{"emergency"},
	)
	require.ErrorContains(t, err, "does not match root commit id")
}

func exportIAVLNodesForTest(
	t *testing.T,
	tree *iavl.MutableTree,
) []*iavl.ExportNode {
	t.Helper()
	exporter, err := tree.ImmutableTree.Export()
	require.NoError(t, err)
	defer exporter.Close()
	var nodes []*iavl.ExportNode
	for {
		node, err := exporter.Next()
		if errors.Is(err, iavl.ErrorExportDone) {
			return nodes
		}
		require.NoError(t, err)
		require.NotNil(t, node)
		nodes = append(nodes, &iavl.ExportNode{
			Key:     bytes.Clone(node.Key),
			Value:   bytes.Clone(node.Value),
			Version: node.Version,
			Height:  node.Height,
		})
	}
}

func cloneExportNodes(nodes []*iavl.ExportNode) []*iavl.ExportNode {
	cloned := make([]*iavl.ExportNode, len(nodes))
	for i, node := range nodes {
		cloned[i] = &iavl.ExportNode{
			Key:     bytes.Clone(node.Key),
			Value:   bytes.Clone(node.Value),
			Version: node.Version,
			Height:  node.Height,
		}
	}
	return cloned
}
