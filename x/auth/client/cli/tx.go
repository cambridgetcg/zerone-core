package cli

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/auth/types"
)

// GetTxCmd returns the transaction commands for this module.
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Zerone auth transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdOnboard(),
		CmdRegisterAccount(),
		CmdRotateKey(),
		CmdFreezeAccount(),
		CmdUnfreezeAccount(),
	)

	return txCmd
}

// onboardIdentity is the on-disk shape of a generated zerone identity.
// The private key never leaves this file; the chain only ever sees the pubkey.
type onboardIdentity struct {
	Address       string `json:"address"`
	Did           string `json:"did"`
	PublicKeyHex  string `json:"public_key_hex"`
	PrivateKeyHex string `json:"private_key_hex"`
	Note          string `json:"note"`
}

// CmdOnboard is the one-shot hospitable door: it generates (or reuses) an
// ed25519 identity keypair, derives the self-certifying did:zrn, persists the
// identity file BEFORE broadcasting (so a failed tx never orphans a key), and
// registers the account. Everything register-account does, minus the gauntlet.
func CmdOnboard() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onboard [account-type]",
		Short: "One-shot registration: generate an identity key, derive your DID, register",
		Long: `Generates an ed25519 identity keypair (separate from your tx-signing key),
derives the self-certifying DID (did:zrn:<pubkey-hex>), saves the identity to a
file you must keep (it is your future key-rotation anchor), and broadcasts
MsgRegisterAccount.

account-type defaults to "agent". "agent" and "human" can submit claims and
witness; "contract" and "system" cannot (CanSubmitClaims=false).

Re-running with the same --from and identity file reuses the saved key, so a
failed or interrupted registration is safe to retry.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			accountType := "agent"
			if len(args) == 1 {
				accountType = args[0]
			}
			switch accountType {
			case "agent", "human":
			case "contract", "system":
				fmt.Fprintln(cmd.ErrOrStderr(), "⚠ note: account type '"+accountType+"' cannot submit claims or witness (CanSubmitClaims=false). Use 'agent' or 'human' if you want to participate in the knowledge pipeline.")
			default:
				return fmt.Errorf("account_type must be agent, human, contract, or system (got %q)", accountType)
			}

			from := clientCtx.GetFromAddress().String()
			if from == "" {
				return fmt.Errorf("no --from key: onboarding needs a funded tx-signing key (create one with: zeroned keys add <name>)")
			}

			identityOut, _ := cmd.Flags().GetString("identity-out")
			if identityOut == "" {
				identityOut = filepath.Join(clientCtx.HomeDir, "identities", from+".ed25519.json")
			}

			// Reuse an existing identity file (idempotent retry); generate otherwise.
			// Creation uses O_EXCL so two concurrent onboards can never silently
			// overwrite each other's key; reuse verifies pub derives from priv.
			guide := cmd.ErrOrStderr()
			var ident onboardIdentity
			if raw, err := os.ReadFile(identityOut); err == nil {
				if err := json.Unmarshal(raw, &ident); err != nil {
					return fmt.Errorf("identity file %s exists but is unreadable: %w (move it aside to generate a fresh key)", identityOut, err)
				}
				if ident.Address != "" && ident.Address != from {
					return fmt.Errorf("identity file %s belongs to %s, not %s — pass --identity-out to use a different file", identityOut, ident.Address, from)
				}
				seed, err := hex.DecodeString(ident.PrivateKeyHex)
				if err != nil || len(seed) != ed25519.SeedSize {
					return fmt.Errorf("identity file %s has a malformed private key — move it aside to generate a fresh one", identityOut)
				}
				derived := hex.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey))
				if derived != ident.PublicKeyHex || ident.Did != "did:zrn:"+derived {
					return fmt.Errorf("identity file %s is inconsistent (public key does not derive from private key) — refusing to register it", identityOut)
				}
				fmt.Fprintln(guide, "• reusing identity from "+identityOut)
			} else {
				pub, priv, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					return fmt.Errorf("keygen failed: %w", err)
				}
				ident = onboardIdentity{
					Address:       from,
					Did:           "did:zrn:" + hex.EncodeToString(pub),
					PublicKeyHex:  hex.EncodeToString(pub),
					PrivateKeyHex: hex.EncodeToString(priv.Seed()),
					Note:          "KEEP THIS FILE. The private key is your identity/rotation anchor; the chain only ever sees the public key.",
				}
				if err := os.MkdirAll(filepath.Dir(identityOut), 0o700); err != nil {
					return err
				}
				raw, _ := json.MarshalIndent(ident, "", "  ")
				f, err := os.OpenFile(identityOut, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
				if err != nil {
					return fmt.Errorf("could not persist identity before broadcasting (refusing to register a key we might lose; a concurrent onboard may hold %s): %w", identityOut, err)
				}
				if _, err := f.Write(raw); err != nil {
					f.Close()
					return fmt.Errorf("identity write failed: %w", err)
				}
				if err := f.Close(); err != nil {
					return fmt.Errorf("identity write failed: %w", err)
				}
				fmt.Fprintln(guide, "• identity generated and saved: "+identityOut)
			}

			fmt.Fprintln(guide, "• DID: "+ident.Did)
			fmt.Fprintln(guide, "• registering account type '"+accountType+"' for "+from+" …")

			msg := &types.MsgRegisterAccount{
				Sender:      from,
				Did:         ident.Did,
				PublicKey:   ident.PublicKeyHex,
				AccountType: accountType,
			}
			if err := tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg); err != nil {
				return err
			}

			genOnly, _ := cmd.Flags().GetBool(flags.FlagGenerateOnly)
			dryRun, _ := cmd.Flags().GetBool(flags.FlagDryRun)
			fmt.Fprintln(guide, "")
			if genOnly || dryRun {
				fmt.Fprintln(guide, "Tx NOT broadcast (generate-only/dry-run); the identity file is saved and will be reused on the real run.")
				return nil
			}
			fmt.Fprintln(guide, "If the response above shows code: 0, you are registered (verify: zeroned q zerone_auth account "+from+").")
			fmt.Fprintln(guide, "What you can do now:")
			fmt.Fprintln(guide, "  submit a claim     zeroned tx knowledge submit-claim \"<fact>\" <domain> <category> <fee-uzrn> --from "+from)
			fmt.Fprintln(guide, "                     (check the real fee first: zeroned q knowledge effective-fees — network pacing can raise it)")
			fmt.Fprintln(guide, "  witness a round    zeroned tx knowledge submit-commitment <round-id> --vote accept|reject --from "+from)
			fmt.Fprintln(guide, "                     (witnessing currently requires a ≥100 ZRN balance at commit time)")
			fmt.Fprintln(guide, "  watch a claim      zeroned q knowledge claim-watch <claim-id>")
			fmt.Fprintln(guide, "  keep safe          "+identityOut)
			return nil
		},
	}

	cmd.Flags().String("identity-out", "", "path for the generated identity file (default: <home>/identities/<from-address>.ed25519.json)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRegisterAccount registers a new Zerone account.
func CmdRegisterAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account [did] [public-key] [account-type]",
		Short: "Register a new Zerone account with DID mapping",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			opKeyHash, _ := cmd.Flags().GetString("operational-key-hash")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgRegisterAccount{
				Sender:             clientCtx.GetFromAddress().String(),
				Did:                args[0],
				PublicKey:          args[1],
				AccountType:        args[2],
				OperationalKeyHash: opKeyHash,
				Metadata:           metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("operational-key-hash", "", "Hash of initial operational key")
	cmd.Flags().String("metadata", "", "Account metadata (JSON string)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdRotateKey rotates the operational key.
func CmdRotateKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-key [new-op-key-hex] [auth-sig-hex]",
		Short: "Rotate operational key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newKey, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid new key hex: %w", err)
			}

			authSig, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid auth signature hex: %w", err)
			}

			msg := &types.MsgRotateKey{
				Sender:                 clientCtx.GetFromAddress().String(),
				NewOperationalKey:      newKey,
				AuthorizationSignature: authSig,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdFreezeAccount freezes an account.
func CmdFreezeAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze-account [address]",
		Short: "Freeze an account (self or authority)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			reason, _ := cmd.Flags().GetString("reason")

			msg := &types.MsgFreezeAccount{
				Sender:  clientCtx.GetFromAddress().String(),
				Address: args[0],
				Reason:  reason,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("reason", "", "Reason for freezing")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdUnfreezeAccount unfreezes an account.
func CmdUnfreezeAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze-account [address]",
		Short: "Unfreeze a frozen account (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnfreezeAccount{
				Authority: clientCtx.GetFromAddress().String(),
				Address:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

