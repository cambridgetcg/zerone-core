package app

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	challengetypes "github.com/zerone-chain/zerone/x/capture_challenge/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	trustkeeper "github.com/zerone-chain/zerone/x/trust_score/keeper"
	trusttypes "github.com/zerone-chain/zerone/x/trust_score/types"
)

// This fixture exercises the application-installed readers with synthetic records.
// It does not adjudicate a challenge, establish a person's trustworthiness or run
// a node/transport. Query output, read gas and producer-store bytes are separate checks.
func TestTrustScoreComposition(t *testing.T) {
	application := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID("trust-observer"), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, application.Close()) })
	ctx := application.NewUncachedContext(false, cmtproto.Header{ChainID: "trust-observer", Height: 77})
	query := trustkeeper.NewQueryServerImpl(application.TrustScoreKeeper)
	const address = "synthetic-observed-address"

	// Length-delimited raw entries from the three producer stores, not an AppHash.
	storeDigest := func() [32]byte {
		h := sha256.New()
		for _, name := range []string{knowledgetypes.StoreKey, qualificationtypes.StoreKey, challengetypes.StoreKey} {
			kv := ctx.KVStore(application.keys[name])
			iterator := kv.Iterator(nil, nil)
			for ; iterator.Valid(); iterator.Next() {
				for _, value := range [][]byte{[]byte(name), iterator.Key(), iterator.Value()} {
					var size [8]byte
					binary.BigEndian.PutUint64(size[:], uint64(len(value)))
					_, _ = h.Write(size[:])
					_, _ = h.Write(value)
				}
			}
			require.NoError(t, iterator.Close())
		}
		var digest [32]byte
		copy(digest[:], h.Sum(nil))
		return digest
	}
	// Captured from the pre-extraction keeper at df848ce79c0b9ff02279a3ca2b3e46b216ae4574.
	snapshots := map[string]struct {
		gas        uint64
		wireSHA256 string
	}{
		"empty":               {1141, "9a90076d117ddf06272f76c932f0a8e648531ae5bab4bc53ed123bae71208353"},
		"weighted":            {14384, "d453b59864c4d358ca38ccfcb4cd2868107eaca7dce6d69de0258ff772c652a9"},
		"penalized":           {15110, "c8f9c13d69a3292980376deb532911847a72131c913d2086e904f53636e85802"},
		"expired":             {15110, "5b0fa35d16225496cd1aa98dc622f2bfdc7fb8494c9c13c947a7568a9151e18b"},
		"strike":              {16805, "d06154b6b033700a8bdaad1f749a7e4b0f98d464883089e451f837d0510c9af1"},
		"strike-after-expiry": {16805, "23abfa1f4739cea54cfb6ec60151f0fd1fbab679e15ff0a365b1d69d7a76151a"},
	}
	check := func(name string, height int64, composite uint64, penalties, strikes uint32, band string, domains int) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			before := storeDigest()
			observed := ctx.WithBlockHeight(height).WithGasMeter(storetypes.NewInfiniteGasMeter())
			response, err := query.TrustScore(observed, &trusttypes.QueryTrustScoreRequest{Address: address})
			require.NoError(t, err)
			readGas := observed.GasMeter().GasConsumed()
			score := response.Score
			require.Equal(t, address, score.Address)
			require.Equal(t, uint64(height), score.ComputedAtBlock)
			require.Equal(t, composite, score.CompositeBps)
			require.Equal(t, penalties, score.ActivePenalties)
			require.Equal(t, strikes, score.CartelStrikes)
			require.Equal(t, band, score.Band)
			require.Len(t, score.PerDomain, domains)
			for i := 1; i < len(score.PerDomain); i++ {
				require.Less(t, score.PerDomain[i-1].Domain, score.PerDomain[i].Domain)
			}
			wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(response)
			require.NoError(t, err)
			want, exists := snapshots[name]
			require.True(t, exists, "missing baseline fixture")
			require.Equal(t, want.gas, readGas, "read gas changed")
			require.Equal(t, want.wireSHA256, fmt.Sprintf("%x", sha256.Sum256(wire)), "query bytes changed")
			require.Equal(t, before, storeDigest(), "query changed a producer store")
			repeated, err := query.TrustScore(ctx.WithBlockHeight(height), &trusttypes.QueryTrustScoreRequest{Address: address})
			require.NoError(t, err)
			require.True(t, proto.Equal(response, repeated))
		})
	}
	check("empty", 77, 500_000, 0, 0, "C", 0)
	calibration := &knowledgetypes.AgentCalibration{Address: address, CalibrationScoreBps: 900_000}
	require.NoError(t, application.KnowledgeKeeper.SetAgentCalibration(ctx, calibration))
	active := qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE
	for _, q := range []*qualificationtypes.DomainQualification{
		{Validator: address, Domain: "z-primary", Status: active, Weight: 3, Metrics: &qualificationtypes.QualificationMetrics{AccuracyBps: 800_001, TotalVerifications: 9}},
		{Validator: address, Domain: "a-secondary", Status: active, Weight: 1, Metrics: &qualificationtypes.QualificationMetrics{AccuracyBps: 400_000, TotalVerifications: 5}},
		{Validator: address, Domain: "inactive", Weight: 100, Metrics: &qualificationtypes.QualificationMetrics{AccuracyBps: 1_000_000}},
		{Validator: address, Domain: "no-metrics", Status: active, Weight: 0},
		{Validator: "different-address", Domain: "excluded", Status: active, Weight: 100, Metrics: &qualificationtypes.QualificationMetrics{AccuracyBps: 1_000_000}},
	} {
		application.QualificationKeeper.SetQualification(ctx, q)
	}
	check("weighted", 77, 800_000, 0, 0, "A", 4)
	require.NoError(t, application.QualificationKeeper.ReduceQualificationWeight(ctx, address, "z-primary", 500_000, 80))
	check("penalized", 77, 700_000, 1, 0, "B", 4)
	check("expired", 80, 800_000, 0, 0, "A", 4)

	resolved := challengetypes.ChallengeStatus_CHALLENGE_STATUS_RESOLVED
	upheld := challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD
	for i, c := range []*challengetypes.CaptureChallenge{
		{Status: resolved, Resolution: &challengetypes.CaptureResolution{Outcome: upheld}, AccusedValidators: []string{address, address}},
		{Status: resolved, Resolution: &challengetypes.CaptureResolution{Outcome: upheld}, AccusedValidators: []string{"different-address"}},
		{Status: resolved, Resolution: &challengetypes.CaptureResolution{Outcome: challengetypes.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED}, AccusedValidators: []string{address}},
		{Resolution: &challengetypes.CaptureResolution{Outcome: upheld}, AccusedValidators: []string{address}},
		{Status: resolved, AccusedValidators: []string{address}},
	} {
		c.Id = fmt.Sprintf("synthetic-challenge-%d", i)
		application.CaptureChallengeKeeper.SetChallenge(ctx, c)
	}
	check("strike", 77, 450_000, 1, 1, "F", 4)
	check("strike-after-expiry", 80, 550_000, 0, 1, "F", 4)
	for _, request := range []*trusttypes.QueryTrustScoreRequest{nil, {}} {
		_, err := query.TrustScore(ctx, request)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}
