package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: "Query substrate_bridge state",
	}
	cmd.AddCommand(cmdQueryParams())
	cmd.AddCommand(cmdQueryAdapter())
	cmd.AddCommand(cmdQueryAdapters())
	cmd.AddCommand(cmdQueryAttestation())
	cmd.AddCommand(cmdQueryLineageForward())
	cmd.AddCommand(cmdQueryLineageBackward())
	cmd.AddCommand(cmdQueryLineageAccumulator())
	return cmd
}

func cmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Show module params",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cctx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(cctx).Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQueryAdapter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adapter [adapter-id]",
		Short: "Show a registered adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, _ := client.GetClientQueryContext(cmd)
			res, err := types.NewQueryClient(cctx).Adapter(cmd.Context(), &types.QueryAdapterRequest{AdapterId: args[0]})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// parseAdapterStatus accepts the short form an operator actually types
// ("active") as well as the full enum name ("ADAPTER_STATUS_ACTIVE"), in any
// case. An unknown value is rejected by name rather than silently falling back
// to "no filter" — a filter that quietly does nothing is worse than an error,
// because the caller cannot tell an empty result from an ignored request.
func parseAdapterStatus(s string) (types.AdapterStatus, error) {
	if s == "" {
		return types.AdapterStatus_ADAPTER_STATUS_UNSPECIFIED, nil
	}
	name := strings.ToUpper(s)
	if !strings.HasPrefix(name, "ADAPTER_STATUS_") {
		name = "ADAPTER_STATUS_" + name
	}
	v, ok := types.AdapterStatus_value[name]
	if !ok {
		valid := make([]string, 0, len(types.AdapterStatus_value))
		for k := range types.AdapterStatus_value {
			valid = append(valid, strings.TrimPrefix(k, "ADAPTER_STATUS_"))
		}
		sort.Strings(valid)
		return 0, fmt.Errorf("unknown adapter status %q (valid: %s)", s, strings.Join(valid, ", "))
	}
	return types.AdapterStatus(v), nil
}

func cmdQueryAdapters() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "adapters",
		Short: "List adapters (optionally filtered by status)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cctx, _ := client.GetClientQueryContext(cmd)
			filter, err := parseAdapterStatus(status)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(cctx).Adapters(cmd.Context(), &types.QueryAdaptersRequest{StatusFilter: filter})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "only list adapters with this status (active, suspended, tombstoned)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQueryAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation [attestation-id]",
		Short: "Show an external attestation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, _ := client.GetClientQueryContext(cmd)
			res, err := types.NewQueryClient(cctx).Attestation(cmd.Context(), &types.QueryAttestationRequest{AttestationId: args[0]})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQueryLineageForward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lineage-forward [attestation-id]",
		Short: "Walk forward lineage (downstream uses)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, _ := client.GetClientQueryContext(cmd)
			res, err := types.NewQueryClient(cctx).LineageForwardWalk(cmd.Context(), &types.QueryLineageForwardWalkRequest{AttestationId: args[0]})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQueryLineageBackward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lineage-backward [attestation-id]",
		Short: "Walk backward lineage (upstream cites)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, _ := client.GetClientQueryContext(cmd)
			res, err := types.NewQueryClient(cctx).LineageBackwardWalk(cmd.Context(), &types.QueryLineageBackwardWalkRequest{AttestationId: args[0]})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQueryLineageAccumulator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lineage-accumulator [attestation-id]",
		Short: "Cumulative lineage royalty income for an attestation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, _ := client.GetClientQueryContext(cmd)
			res, err := types.NewQueryClient(cctx).LineageAccumulator(cmd.Context(), &types.QueryLineageAccumulatorRequest{AttestationId: args[0]})
			if err != nil {
				return err
			}
			return cctx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
