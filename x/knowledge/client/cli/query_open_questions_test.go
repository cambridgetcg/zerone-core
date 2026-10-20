package cli

import (
	"bytes"
	"context"
	"math"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

type openQuestionsQueryServer struct {
	types.UnimplementedQueryServer
	captured chan *types.QueryOpenQuestionsRequest
	response *types.QueryOpenQuestionsResponse
	err      error
}

func (s *openQuestionsQueryServer) OpenQuestions(_ context.Context, req *types.QueryOpenQuestionsRequest) (*types.QueryOpenQuestionsResponse, error) {
	s.captured <- req
	return s.response, s.err
}

func TestOpenQuestionsQueryFlags(t *testing.T) {
	cmd := NewQueryOpenQuestionsCmd()
	for _, name := range []string{"domain", flags.FlagLimit, flags.FlagHeight, flags.FlagNode, flags.FlagOutput} {
		require.NotNilf(t, cmd.Flags().Lookup(name), "missing --%s", name)
	}
	limit, err := cmd.Flags().GetUint32(flags.FlagLimit)
	require.NoError(t, err)
	require.Equal(t, uint32(100), limit)
	require.Equal(t, "100", cmd.Flags().Lookup(flags.FlagLimit).DefValue)
	domain, err := cmd.Flags().GetString("domain")
	require.NoError(t, err)
	require.Empty(t, domain)
	// The existing RPC has no PageRequest. Do not advertise controls it cannot honor.
	for _, name := range []string{flags.FlagPage, flags.FlagPageKey, flags.FlagOffset, flags.FlagCountTotal, flags.FlagReverse} {
		require.Nilf(t, cmd.Flags().Lookup(name), "unsupported --%s", name)
	}
	require.Contains(t, cmd.Long, "scan_limit_reached")
	require.Contains(t, cmd.Long, "not page/offset or cursor pagination")
}

func TestOpenQuestionsQueryForwardsRequestAndPrintsResponse(t *testing.T) {
	response := &types.QueryOpenQuestionsResponse{
		Questions: []*types.OpenQuestion{{
			FactId: "question-7", Statement: "An unsettled proposition", FalsificationPredicate: "A counterexample",
			Domain: "mathematics", Proposer: "synthetic-proposer", PostedAtBlock: 11,
			SurvivedProbes: 3, Energy: 17, UnderChallenge: true,
		}},
		Total: 9, SnapshotBlockHeight: 42, FactsExamined: 5000, ScanLimitReached: true,
	}
	tests := []struct {
		name     string
		args     []string
		domain   string
		limit    uint32
		response *types.QueryOpenQuestionsResponse
	}{
		{name: "defaults", limit: 100, response: response},
		{name: "filtered", args: []string{"--domain=mathematics", "--limit=7"}, domain: "mathematics", limit: 7, response: response},
		{name: "server default", args: []string{"--limit=0"}, response: response},
		{name: "uint32 maximum forwarded for server clamp", args: []string{"--limit=4294967295"}, limit: math.MaxUint32, response: response},
		{name: "empty result", limit: 100, response: &types.QueryOpenQuestionsResponse{SnapshotBlockHeight: 42}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := &openQuestionsQueryServer{captured: make(chan *types.QueryOpenQuestionsRequest, 1), response: test.response}
			var output bytes.Buffer
			clientCtx := knowledgeQueryClient(t, server).
				WithCodec(codec.NewProtoCodec(codectypes.NewInterfaceRegistry())).
				WithOutput(&output)
			cmd := GetQueryCmd()
			cmd.SetContext(context.Background())
			require.NoError(t, client.SetCmdClientContext(cmd, clientCtx))
			args := append([]string{"open-questions", "--output=json"}, test.args...)
			cmd.SetArgs(args)
			require.NoError(t, cmd.Execute())

			select {
			case request := <-server.captured:
				require.Equal(t, test.domain, request.Domain)
				require.Equal(t, test.limit, request.Limit)
			default:
				t.Fatal("OpenQuestions RPC was not called")
			}
			var got types.QueryOpenQuestionsResponse
			require.NoError(t, protojson.Unmarshal(output.Bytes(), &got))
			require.True(t, proto.Equal(test.response, &got), "response fields must survive CLI output: %s", output.String())
		})
	}
}

func TestOpenQuestionsQueryPropagatesRPCError(t *testing.T) {
	server := &openQuestionsQueryServer{
		captured: make(chan *types.QueryOpenQuestionsRequest, 1),
		err:      status.Error(codes.Unavailable, "synthetic query failure"),
	}
	var output bytes.Buffer
	cmd := NewQueryOpenQuestionsCmd()
	cmd.SetContext(context.Background())
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	require.NoError(t, client.SetCmdClientContext(cmd, knowledgeQueryClient(t, server).WithOutput(&output)))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.ErrorContains(t, err, "failed to query open questions")
	require.ErrorContains(t, err, "synthetic query failure")
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Empty(t, output.String())
	require.Len(t, server.captured, 1)
}

func TestOpenQuestionsQueryRejectsInvalidArgumentsBeforeRPC(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "positional argument", args: []string{"mathematics"}},
		{name: "negative limit", args: []string{"--limit=-1"}},
		{name: "overflowing limit", args: []string{"--limit=4294967296"}},
		{name: "noninteger limit", args: []string{"--limit=many"}},
		{name: "page", args: []string{"--page=2"}},
		{name: "cursor", args: []string{"--page-key=YWJj"}},
		{name: "offset", args: []string{"--offset=1"}},
		{name: "count total", args: []string{"--count-total"}},
		{name: "reverse", args: []string{"--reverse"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := &openQuestionsQueryServer{
				captured: make(chan *types.QueryOpenQuestionsRequest, 1),
				err:      status.Error(codes.Aborted, "unexpected RPC"),
			}
			cmd := NewQueryOpenQuestionsCmd()
			cmd.SetContext(context.Background())
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			require.NoError(t, client.SetCmdClientContext(cmd, knowledgeQueryClient(t, server)))
			cmd.SetArgs(test.args)
			require.Error(t, cmd.Execute())
			require.Empty(t, server.captured, "invalid arguments must be rejected before querying")
		})
	}
}
