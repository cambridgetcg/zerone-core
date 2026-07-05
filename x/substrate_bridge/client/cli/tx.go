package cli

import (
	"encoding/json"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: "substrate_bridge transactions",
	}
	cmd.AddCommand(
		cmdSubmitExternalAttestation(),
		cmdRegisterAdapter(),
		cmdSuspendAdapter(),
		cmdTombstoneAdapter(),
	)
	return cmd
}

// cmdRegisterAdapter registers an external-source adapter (e.g. the
// agenttool-invocation adapter). The message is authority-gated, so in
// production it is carried by a governance proposal; use --generate-only to
// emit the message for `tx gov submit-proposal`.
func cmdRegisterAdapter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-adapter [adapter-json-file]",
		Short: "Register an external-source adapter (authority-gated; carry via governance). File is a JSON AdapterRegistration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var reg types.AdapterRegistration
			if err := cctx.Codec.UnmarshalJSON(data, &reg); err != nil {
				return err
			}
			msg := &types.MsgRegisterAdapter{
				Authority: cctx.GetFromAddress().String(),
				Adapter:   &reg,
			}
			return tx.GenerateOrBroadcastTxCLI(cctx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSuspendAdapter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suspend-adapter [adapter-id] [reason]",
		Short: "Suspend an active adapter (authority-gated).",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgSuspendAdapter{
				Authority: cctx.GetFromAddress().String(),
				AdapterId: args[0],
				Reason:    args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(cctx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdTombstoneAdapter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tombstone-adapter [adapter-id] [reason]",
		Short: "Tombstone an adapter permanently (authority-gated).",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgTombstoneAdapter{
				Authority: cctx.GetFromAddress().String(),
				AdapterId: args[0],
				Reason:    args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(cctx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSubmitExternalAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-attestation [adapter-id] [work-class-id] [link-json-file] [bond-uzrn]",
		Short: "Submit an external attestation. link-json-file is a JSON-encoded SubstrateLink.",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			// Read SubstrateLink from JSON file.
			var link types.SubstrateLink
			if err := readJSONFile(args[2], &link); err != nil {
				return err
			}
			// The link's adapter_id must match the message; set it, then fill the
			// M2 re-derivable integrity hash so a caller never hand-computes the
			// sha256 canonical form — the keeper re-derives and verifies it.
			link.AdapterId = args[0]
			link.LinkHash = keeper.ComputeLinkHash(&link)
			msg := &types.MsgSubmitExternalAttestation{
				Submitter:   cctx.GetFromAddress().String(),
				AdapterId:   args[0],
				WorkClassId: args[1],
				Link:        &link,
				BondUzrn:    args[3],
			}
			return tx.GenerateOrBroadcastTxCLI(cctx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// readJSONFile reads a JSON file and unmarshals it into v.
func readJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
