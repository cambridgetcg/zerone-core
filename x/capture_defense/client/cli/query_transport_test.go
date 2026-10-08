package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/capture_defense/client/cli"
	"github.com/zerone-chain/zerone/x/capture_defense/keeper"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

type queryObservation struct {
	method  string
	request any
}

// Exercise the actual generated transport and query keeper over an in-memory
// connection. No node, socket, signing key or external endpoint is involved.
func queryClient(t *testing.T, upstreamError error) (client.Context, context.Context, <-chan queryObservation) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	// Select synchronous pruning before loading IAVL: DB.Close does not stop its async worker.
	cms.SetIAVLSyncPruning(true)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())
	sdkCtx := sdk.NewContext(cms, cmtproto.Header{Time: time.Now()}, false, log.NewNopLogger())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "")
	k.SetGlobalReputation(sdkCtx, &types.GlobalReputation{Validator: "observer", Score: 500000})
	k.SetDomainReputation(sdkCtx, &types.DomainReputation{Validator: "observer", Domain: "physics", Score: 600000, Verifications: 5})
	k.SetCaptureMetrics(sdkCtx, &types.CaptureMetrics{Domain: "physics", HerfindahlIndex: 700000, Flagged: true})
	k.SetCaptureMetrics(sdkCtx, &types.CaptureMetrics{Domain: "chemistry", HerfindahlIndex: 100000})
	k.SetCrossStratumRequirement(sdkCtx, &types.CrossStratumRequirement{
		TargetStratum: "empirical", RequiredStrata: []string{"formal"}, MinValidatorsPerStratum: 2,
	})

	observed := make(chan queryObservation, 1)
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(_ context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		observed <- queryObservation{method: info.FullMethod, request: req}
		if upstreamError != nil {
			return nil, upstreamError
		}
		return handler(sdk.WrapSDKContext(sdkCtx), req)
	}))
	types.RegisterQueryServer(server, keeper.NewQueryServerImpl(k))
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		require.NoError(t, listener.Close())
		err := <-served
		require.True(t, err == nil || errors.Is(err, grpc.ErrServerStopped), "gRPC server: %v", err)
	})
	conn, err := grpc.NewClient("passthrough:///capture-defense-query-test",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return client.Context{}.WithLegacyAmino(codec.NewLegacyAmino()).WithGRPCClient(conn), ctx, observed
}

func TestQueryClientCleanup(t *testing.T) {
	pruners := func() int {
		var stacks bytes.Buffer
		require.NoError(t, pprof.Lookup("goroutine").WriteTo(&stacks, 2))
		return strings.Count(stacks.String(), "github.com/cosmos/iavl.(*nodeDB).startPruning(")
	}
	before := pruners()
	for i := 0; i < 5; i++ {
		t.Run("fixture", func(t *testing.T) {
			queryClient(t, nil)
		})
	}
	// Subtest cleanups have run; check IAVL workers rather than unrelated gRPC goroutines.
	require.Eventually(t, func() bool { return pruners() == before }, 300*time.Millisecond, 10*time.Millisecond,
		"IAVL pruning goroutines before=%d after=%d after five completed fixture cleanups", before, pruners())
	t.Logf("IAVL pruning goroutines before=%d after=%d", before, pruners())
}

func TestQueryCommandsReachExistingService(t *testing.T) {
	tests := []struct {
		args     []string
		method   string
		request  proto.Message
		response proto.Message
		want     proto.Message
	}{
		{[]string{"params"}, types.Query_Params_FullMethodName, &types.QueryParamsRequest{}, &types.QueryParamsResponse{}, &types.QueryParamsResponse{Params: types.DefaultParams()}},
		{[]string{"reputation", "observer"}, types.Query_Reputation_FullMethodName, &types.QueryReputationRequest{Validator: "observer"}, &types.QueryReputationResponse{}, &types.QueryReputationResponse{Global: &types.GlobalReputation{Validator: "observer", Score: 500000}, EffectiveScore: 500000}},
		{[]string{"reputation", "observer", "physics"}, types.Query_Reputation_FullMethodName, &types.QueryReputationRequest{Validator: "observer", Domain: "physics"}, &types.QueryReputationResponse{}, &types.QueryReputationResponse{Global: &types.GlobalReputation{Validator: "observer", Score: 500000}, Domain: &types.DomainReputation{Validator: "observer", Domain: "physics", Score: 600000, Verifications: 5}, EffectiveScore: 600000}},
		{[]string{"metrics", "physics"}, types.Query_CaptureMetrics_FullMethodName, &types.QueryCaptureMetricsRequest{Domain: "physics"}, &types.QueryCaptureMetricsResponse{}, &types.QueryCaptureMetricsResponse{Metrics: &types.CaptureMetrics{Domain: "physics", HerfindahlIndex: 700000, Flagged: true}}},
		{[]string{"flagged-domains"}, types.Query_FlaggedDomains_FullMethodName, &types.QueryFlaggedDomainsRequest{}, &types.QueryFlaggedDomainsResponse{}, &types.QueryFlaggedDomainsResponse{Metrics: []*types.CaptureMetrics{{Domain: "physics", HerfindahlIndex: 700000, Flagged: true}}}},
		{[]string{"cross-stratum-requirements"}, types.Query_CrossStratumRequirements_FullMethodName, &types.QueryCrossStratumRequirementsRequest{}, &types.QueryCrossStratumRequirementsResponse{}, &types.QueryCrossStratumRequirementsResponse{Requirements: []*types.CrossStratumRequirement{{TargetStratum: "empirical", RequiredStrata: []string{"formal"}, MinValidatorsPerStratum: 2}}}},
	}
	for _, tc := range tests {
		t.Run(tc.method+"/"+tc.args[len(tc.args)-1], func(t *testing.T) {
			clientCtx, ctx, observed := queryClient(t, nil)
			var output bytes.Buffer
			root := cli.NewQueryCmd()
			root.SetContext(ctx)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			require.NoError(t, client.SetCmdClientContext(root, clientCtx.WithOutput(&output).WithOutputFormat("json")))
			root.SetArgs(tc.args)
			require.NoError(t, root.Execute())
			var got queryObservation
			select {
			case got = <-observed:
			case <-ctx.Done():
				t.Fatal("query command did not reach the service")
			}
			require.Equal(t, tc.method, got.method)
			require.IsType(t, tc.request, got.request)
			require.True(t, proto.Equal(tc.request, got.request.(proto.Message)), "unexpected query request: %v", got.request)
			require.NoError(t, clientCtx.LegacyAmino.UnmarshalJSON(output.Bytes(), tc.response))
			require.True(t, proto.Equal(tc.want, tc.response), "want %s; got %s", tc.want, tc.response)
		})
	}
}

func TestQueryCommandsPreserveErrors(t *testing.T) {
	for _, args := range [][]string{
		{"params"}, {"reputation", "observer"}, {"metrics", "physics"},
		{"flagged-domains"}, {"cross-stratum-requirements"},
	} {
		t.Run(args[0], func(t *testing.T) {
			clientCtx, ctx, observed := queryClient(t, status.Error(codes.Unavailable, "test service unavailable"))
			var output bytes.Buffer
			root := cli.NewQueryCmd()
			root.SetContext(ctx)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			require.NoError(t, client.SetCmdClientContext(root, clientCtx.WithOutput(&output).WithOutputFormat("json")))
			root.SetArgs(args)
			err := root.Execute()
			require.ErrorContains(t, err, "test service unavailable")
			require.Equal(t, codes.Unavailable, status.Code(err))
			require.Empty(t, output.String(), "errors must not print a successful response")
			require.Len(t, observed, 1)
		})
	}
}

func TestCrossStratumRequirementsRejectsArgumentsBeforeQuery(t *testing.T) {
	clientCtx, ctx, observed := queryClient(t, nil)
	root := cli.NewQueryCmd()
	root.SetContext(ctx)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	require.NoError(t, client.SetCmdClientContext(root, clientCtx))
	root.SetArgs([]string{"cross-stratum-requirements", "extra"})
	require.Error(t, root.Execute())
	require.Empty(t, observed)
}
