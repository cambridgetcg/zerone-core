package cli

import (
	"context"
	"net"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

type capturedKnowledgeQuery struct {
	method  string
	request any
}

type capturingKnowledgeQueryServer struct {
	types.UnimplementedQueryServer
	captured chan capturedKnowledgeQuery
}

func (s *capturingKnowledgeQueryServer) capture(method string, request any) error {
	s.captured <- capturedKnowledgeQuery{method: method, request: request}
	return status.Error(codes.Aborted, "request captured")
}

func (s *capturingKnowledgeQueryServer) Facts(_ context.Context, req *types.QueryFactsRequest) (*types.QueryFactsResponse, error) {
	return nil, s.capture("Facts", req)
}

func (s *capturingKnowledgeQueryServer) FactsByDomain(_ context.Context, req *types.QueryFactsByDomainRequest) (*types.QueryFactsByDomainResponse, error) {
	return nil, s.capture("FactsByDomain", req)
}

func (s *capturingKnowledgeQueryServer) FactsBySubmitter(_ context.Context, req *types.QueryFactsBySubmitterRequest) (*types.QueryFactsBySubmitterResponse, error) {
	return nil, s.capture("FactsBySubmitter", req)
}

func (s *capturingKnowledgeQueryServer) PendingClaims(_ context.Context, req *types.QueryPendingClaimsRequest) (*types.QueryPendingClaimsResponse, error) {
	return nil, s.capture("PendingClaims", req)
}

func (s *capturingKnowledgeQueryServer) Domains(_ context.Context, req *types.QueryDomainsRequest) (*types.QueryDomainsResponse, error) {
	return nil, s.capture("Domains", req)
}

func queryCaptureClient(t *testing.T) (client.Context, <-chan capturedKnowledgeQuery) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	captured := make(chan capturedKnowledgeQuery, 1)
	server := grpc.NewServer()
	types.RegisterQueryServer(server, &capturingKnowledgeQueryServer{captured: captured})
	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"passthrough:///knowledge-query-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
		server.Stop()
		require.NoError(t, listener.Close())
	})

	return client.Context{}.WithGRPCClient(conn), captured
}

func TestPaginatedQueryCommandsExposeBoundedDefaults(t *testing.T) {
	tests := []struct {
		name string
		new  func() *cobra.Command
	}{
		{name: "facts", new: NewQueryFactsCmd},
		{name: "facts by domain", new: NewQueryFactsByDomainCmd},
		{name: "facts by submitter", new: NewQueryFactsBySubmitterCmd},
		{name: "pending claims", new: NewQueryPendingClaimsCmd},
		{name: "domains", new: NewQueryDomainsCmd},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := test.new()
			for _, name := range []string{
				flags.FlagPage,
				flags.FlagPageKey,
				flags.FlagOffset,
				flags.FlagLimit,
				flags.FlagCountTotal,
				flags.FlagReverse,
			} {
				require.NotNilf(t, cmd.Flags().Lookup(name), "missing --%s", name)
			}
			limit, err := cmd.Flags().GetUint64(flags.FlagLimit)
			require.NoError(t, err)
			require.Equal(t, uint64(50), limit)
			require.Equal(t, "50", cmd.Flags().Lookup(flags.FlagLimit).DefValue)
			require.NotContains(t, cmd.Short, " all ")
		})
	}
}

func TestPaginatedQueryCommandsForwardPageRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		newCommand func() *cobra.Command
		args       []string
		pagination func(any) *sdkquery.PageRequest
	}{
		{
			name:       "facts",
			method:     "Facts",
			newCommand: NewQueryFactsCmd,
			pagination: func(req any) *sdkquery.PageRequest { return req.(*types.QueryFactsRequest).Pagination },
		},
		{
			name:       "facts by domain",
			method:     "FactsByDomain",
			newCommand: NewQueryFactsByDomainCmd,
			args:       []string{"physics"},
			pagination: func(req any) *sdkquery.PageRequest { return req.(*types.QueryFactsByDomainRequest).Pagination },
		},
		{
			name:       "facts by submitter",
			method:     "FactsBySubmitter",
			newCommand: NewQueryFactsBySubmitterCmd,
			args:       []string{"zrn1submitter"},
			pagination: func(req any) *sdkquery.PageRequest { return req.(*types.QueryFactsBySubmitterRequest).Pagination },
		},
		{
			name:       "pending claims",
			method:     "PendingClaims",
			newCommand: NewQueryPendingClaimsCmd,
			pagination: func(req any) *sdkquery.PageRequest { return req.(*types.QueryPendingClaimsRequest).Pagination },
		},
		{
			name:       "domains",
			method:     "Domains",
			newCommand: NewQueryDomainsCmd,
			pagination: func(req any) *sdkquery.PageRequest { return req.(*types.QueryDomainsRequest).Pagination },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientCtx, captured := queryCaptureClient(t)
			cmd := test.newCommand()
			cmd.SetContext(context.Background())
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			require.NoError(t, client.SetCmdClientContext(cmd, clientCtx))
			args := append([]string{}, test.args...)
			args = append(args, "--limit=7", "--offset=9", "--count-total", "--reverse")
			cmd.SetArgs(args)

			err := cmd.Execute()
			require.ErrorContains(t, err, "request captured")
			got := <-captured
			require.Equal(t, test.method, got.method)
			page := test.pagination(got.request)
			require.NotNil(t, page)
			require.Equal(t, uint64(9), page.Offset)
			require.Equal(t, uint64(7), page.Limit)
			require.True(t, page.CountTotal)
			require.True(t, page.Reverse)
		})
	}
}
