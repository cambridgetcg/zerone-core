package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/private_corpus/types"
)

const (
	flagDescription = "description"
	flagPolicyURL   = "policy-url"
	flagEndpoint    = "endpoint"
	flagDisplayName = "display-name"
	flagPubkey      = "pubkey"
	flagStatus      = "status"
	flagItemCount   = "item-count"
	flagSizeBytes   = "size-bytes"
	flagManifestID  = "manifest-id"
	flagAccessNote  = "access-note"
)

// NewTxCmd returns the transaction commands for the private_corpus module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Private corpus module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterVaultCmd(),
		NewUpdateVaultCmd(),
		NewPublishManifestCmd(),
		NewWithdrawManifestCmd(),
		NewRecordAccessCmd(),
	)

	return txCmd
}

// NewRegisterVaultCmd creates a CLI command for MsgRegisterVault.
func NewRegisterVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-vault [vault-id] [display-name] [operator-pubkey]",
		Short: "Register a new off-chain vault that you operate",
		Long: `Register a new vault on-chain. The chain anchors the vault's identity
(id, operator address, display name, public key) so off-chain readers can
verify items they receive from your vault server. The data itself is NOT
stored on-chain; you run the server.

The operator-pubkey is the public key you will use to sign vault server
responses. Format is operator-chosen (e.g., "ed25519:base64key" or PEM).

Optional flags:
  --description    Public description visible on-chain
  --policy-url     URL where you publish your access policy
  --endpoint       HTTPS URL of your vault server`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			description, _ := cmd.Flags().GetString(flagDescription)
			policyURL, _ := cmd.Flags().GetString(flagPolicyURL)
			endpoint, _ := cmd.Flags().GetString(flagEndpoint)

			msg := &types.MsgRegisterVault{
				Operator:        clientCtx.GetFromAddress().String(),
				Id:              args[0],
				DisplayName:     args[1],
				OperatorPubkey:  args[2],
				Description:     description,
				AccessPolicyUrl: policyURL,
				ServerEndpoint:  endpoint,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagDescription, "", "Public description of the vault")
	cmd.Flags().String(flagPolicyURL, "", "URL where you publish access policy")
	cmd.Flags().String(flagEndpoint, "", "HTTPS URL of your vault server")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateVaultCmd creates a CLI command for MsgUpdateVault.
func NewUpdateVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-vault [vault-id]",
		Short: "Update metadata or status of a vault you operate",
		Long: `Update one or more fields on an existing vault. Use --status to change
lifecycle state: active, paused, or deprecated.

Each flag is optional; omitted flags leave the field unchanged.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			displayName, _ := cmd.Flags().GetString(flagDisplayName)
			description, _ := cmd.Flags().GetString(flagDescription)
			policyURL, _ := cmd.Flags().GetString(flagPolicyURL)
			pubkey, _ := cmd.Flags().GetString(flagPubkey)
			endpoint, _ := cmd.Flags().GetString(flagEndpoint)
			statusStr, _ := cmd.Flags().GetString(flagStatus)

			status := types.VaultStatus_VAULT_STATUS_UNSPECIFIED
			switch statusStr {
			case "":
				// leave unchanged
			case "active":
				status = types.VaultStatus_VAULT_STATUS_ACTIVE
			case "paused":
				status = types.VaultStatus_VAULT_STATUS_PAUSED
			case "deprecated":
				status = types.VaultStatus_VAULT_STATUS_DEPRECATED
			default:
				return fmt.Errorf("invalid --status %q; expected active, paused, or deprecated", statusStr)
			}

			msg := &types.MsgUpdateVault{
				Operator:        clientCtx.GetFromAddress().String(),
				VaultId:         args[0],
				DisplayName:     displayName,
				Description:     description,
				AccessPolicyUrl: policyURL,
				OperatorPubkey:  pubkey,
				ServerEndpoint:  endpoint,
				Status:          status,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagDisplayName, "", "New display name")
	cmd.Flags().String(flagDescription, "", "New public description")
	cmd.Flags().String(flagPolicyURL, "", "New access policy URL")
	cmd.Flags().String(flagPubkey, "", "New operator public key")
	cmd.Flags().String(flagEndpoint, "", "New server endpoint URL")
	cmd.Flags().String(flagStatus, "", "New status: active | paused | deprecated")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPublishManifestCmd creates a CLI command for MsgPublishManifest.
func NewPublishManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish-manifest [vault-id] [manifest-id] [version] [content-hash]",
		Short: "Publish a manifest hash for a snapshot of vault contents",
		Long: `Publish a manifest hash that anchors a snapshot of your vault's
contents. The content-hash is operator-chosen (Merkle root or flat hash);
it is hex-encoded and 32..128 bytes when decoded.

Convention for manifest-id: "<vault-id>#<n>" where n is monotonically
increasing per vault (e.g., "love-corpus#42").`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			itemCount, _ := cmd.Flags().GetUint64(flagItemCount)
			sizeBytes, _ := cmd.Flags().GetUint64(flagSizeBytes)
			description, _ := cmd.Flags().GetString(flagDescription)

			msg := &types.MsgPublishManifest{
				Operator:    clientCtx.GetFromAddress().String(),
				VaultId:     args[0],
				ManifestId:  args[1],
				Version:     args[2],
				ContentHash: args[3],
				ItemCount:   itemCount,
				SizeBytes:   sizeBytes,
				Description: description,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint64(flagItemCount, 0, "Number of items in the manifest (informational)")
	cmd.Flags().Uint64(flagSizeBytes, 0, "Total size of the manifest content in bytes (informational)")
	cmd.Flags().String(flagDescription, "", "Short description of this manifest version")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewWithdrawManifestCmd creates a CLI command for MsgWithdrawManifest.
func NewWithdrawManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-manifest [manifest-id] [reason]",
		Short: "Mark a previously-published manifest as withdrawn",
		Long: `Mark a manifest as withdrawn. The manifest record stays on-chain (the
chain's audit trail is forward-only) but the status flips to WITHDRAWN
and the reason is recorded. Off-chain readers should treat withdrawn
manifests as no longer authoritative for new training runs.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgWithdrawManifest{
				Operator:   clientCtx.GetFromAddress().String(),
				ManifestId: args[0],
				Reason:     args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRecordAccessCmd creates a CLI command for MsgRecordAccess.
func NewRecordAccessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record-access [vault-id] [accessor]",
		Short: "Record (opt-in audit) that you granted access to a party",
		Long: `Optionally record that you granted access to a specific party at the
current block. The chain neither requires nor verifies these records;
they are a transparency mechanism the operator may use to make their
access pattern public.

The accessor must be a valid bech32 address. Use --manifest-id if the
access was scoped to a specific manifest, --note for a free-form
description.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			manifestID, _ := cmd.Flags().GetString(flagManifestID)
			note, _ := cmd.Flags().GetString(flagAccessNote)

			msg := &types.MsgRecordAccess{
				Operator:   clientCtx.GetFromAddress().String(),
				VaultId:    args[0],
				Accessor:   args[1],
				ManifestId: manifestID,
				Note:       note,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagManifestID, "", "Manifest the access was scoped to (optional)")
	cmd.Flags().String(flagAccessNote, "", "Free-form access note (optional; flag named --access-note to avoid colliding with the global --note tx flag)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
