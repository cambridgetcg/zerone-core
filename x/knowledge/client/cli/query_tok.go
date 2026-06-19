package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// CmdBundleToK supports three sub-forms (one per selector variant for
// CLI ergonomics; the gRPC accepts the union directly).
func CmdBundleToK() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle-tok",
		Short: "Extract a ToK subgraph (TC1: the graph is the headline)",
	}
	cmd.AddCommand(cmdBundleToKRootedSubtree())
	cmd.AddCommand(cmdBundleToKAncestorCone())
	cmd.AddCommand(cmdBundleToKFrontier())
	cmd.AddCommand(cmdBundleToKCascadeReplay())
	return cmd
}

func cmdBundleToKRootedSubtree() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rooted-subtree [root-fact-id]",
		Short: "Bundle the descendants of a root fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			depth, _ := cmd.Flags().GetUint32("max-depth")
			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{
					RootedSubtree: &types.RootedSubtreeSelector{
						RootFactId: args[0], MaxDepth: depth,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint32("max-depth", 5, "max descendant depth (capped at 32)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdBundleToKAncestorCone() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ancestor-cone [leaf-fact-id]",
		Short: "Bundle the ancestor cone from a leaf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			depth, _ := cmd.Flags().GetUint32("max-depth")
			paths, _ := cmd.Flags().GetUint32("max-paths")
			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_AncestorCone{
					AncestorCone: &types.AncestorConeSelector{
						LeafFactId: args[0], MaxDepth: depth, MaxPaths: paths,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint32("max-depth", 5, "max ancestor depth (capped at 32)")
	cmd.Flags().Uint32("max-paths", 10, "max paths (capped at 256)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdBundleToKFrontier() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "frontier [domain]",
		Short: "Bundle the latest facts in a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			sinceStr, _ := cmd.Flags().GetString("since-block")
			limit, _ := cmd.Flags().GetUint32("limit")
			since, err := strconv.ParseUint(sinceStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid --since-block: %w", err)
			}
			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_Frontier{
					Frontier: &types.FrontierSelector{
						Domain: args[0], SinceBlock: since, Limit: limit,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().String("since-block", "0", "include facts accepted at/after this block")
	cmd.Flags().Uint32("limit", 1024, "max facts (capped at 8192)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdBundleToKCascadeReplay() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cascade-replay [disproven-fact-id]",
		Short: "Bundle the disproval-graph from a DISPROVEN root (TC4: the graph carries its disprovals)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			depth, _ := cmd.Flags().GetUint32("max-depth")
			withVind, _ := cmd.Flags().GetBool("include-vindications")
			withSup, _ := cmd.Flags().GetBool("include-supersessions")
			withHist, _ := cmd.Flags().GetBool("include-status-history")

			req := &types.QueryBundleToKRequest{
				Selector: &types.ToKSelector{Variant: &types.ToKSelector_CascadeReplay{
					CascadeReplay: &types.CascadeReplaySelector{
						DisprovenFactId:      args[0],
						MaxDepth:             depth,
						IncludeVindications:  withVind,
						IncludeSupersessions: withSup,
						IncludeStatusHistory: withHist,
					},
				}},
			}
			res, err := types.NewQueryClient(clientCtx).BundleToK(cmd.Context(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint32("max-depth", 1, "cascade depth (1 = first-hop only; cap 3)")
	cmd.Flags().Bool("include-vindications", false, "ship VindicationRecord entries")
	cmd.Flags().Bool("include-supersessions", false, "walk SUPERSEDES chain")
	cmd.Flags().Bool("include-status-history", false, "ship StatusTransition timelines per node")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
