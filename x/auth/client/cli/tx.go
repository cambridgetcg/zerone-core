package cli

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
		CmdRegistrationSignBytes(),
		CmdVerifyRegistrationProof(),
		CmdRotationSignBytes(),
		CmdRotateKey(),
		CmdFreezeAccount(),
		CmdUnfreezeAccount(),
	)

	return txCmd
}

// onboardIdentity is the on-disk shape of a generated Zerone identity and
// initial operational key. The private key never leaves this file; the chain
// only ever sees the public key.
type onboardIdentity struct {
	Address       string `json:"address"`
	Did           string `json:"did"`
	PublicKeyHex  string `json:"public_key_hex"`
	PrivateKeyHex string `json:"private_key_hex"`
	Note          string `json:"note"`
}

const maxOnboardIdentityBytes = 16 << 10

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
file you must keep (it is also the initial operational key), and broadcasts
MsgRegisterAccount. The currently active operational private key authorizes
each later rotation: after rotating, securely retain and back up the new
operational private key because this original file alone cannot authorize a
subsequent rotation.

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
			// Bind and validate the full directory chain before key generation.
			// Creation uses openat(O_EXCL|O_NOFOLLOW) against that held parent, so
			// path replacement cannot redirect or overwrite private material.
			guide := cmd.ErrOrStderr()
			var ident onboardIdentity
			identityLocation, err := secureOnboardIdentityLocation(identityOut, true)
			if err != nil {
				return fmt.Errorf("identity custody directory for %s is unsafe or unavailable: %w", identityOut, err)
			}
			defer identityLocation.close()

			ident, err = readOnboardIdentityAt(identityLocation, from)
			if err == nil {
				fmt.Fprintln(guide, "• reusing identity from "+identityOut)
			} else if errors.Is(err, os.ErrNotExist) {
				pub, priv, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					return fmt.Errorf("keygen failed: %w", err)
				}
				ident = onboardIdentity{
					Address:       from,
					Did:           "did:zrn:" + hex.EncodeToString(pub),
					PublicKeyHex:  hex.EncodeToString(pub),
					PrivateKeyHex: hex.EncodeToString(priv.Seed()),
					Note:          "KEEP THIS FILE. It holds your immutable identity and initial operational private key. After rotation, also retain every newly current operational private key; the chain only sees public keys.",
				}
				if err := persistOnboardIdentityAt(identityLocation, ident); err != nil {
					return fmt.Errorf("could not persist identity before broadcasting (refusing to register a key we might lose; a concurrent onboard may hold %s): %w", identityOut, err)
				}
				fmt.Fprintln(guide, "• identity generated and saved: "+identityOut)
			} else {
				return fmt.Errorf("identity file %s exists but is unsafe or unreadable: %w (move it aside to generate a fresh key)", identityOut, err)
			}
			if err := identityLocation.assertReachable(); err != nil {
				return fmt.Errorf("identity custody path changed before proof signing: %w", err)
			}

			fmt.Fprintln(guide, "• DID: "+ident.Did)
			fmt.Fprintln(guide, "• registering account type '"+accountType+"' for "+from+" …")

			identitySeed, err := hex.DecodeString(ident.PrivateKeyHex)
			if err != nil || len(identitySeed) != ed25519.SeedSize {
				return fmt.Errorf("identity file %s has a malformed private key", identityOut)
			}
			identityProofSignature, err := signAccountRegistrationProof(
				identitySeed,
				clientCtx.ChainID,
				from,
				ident.Did,
				accountType,
				"",
			)
			if err != nil {
				return fmt.Errorf("could not sign identity registration proof: %w", err)
			}

			msg := &types.MsgRegisterAccount{
				Sender:                 from,
				Did:                    ident.Did,
				PublicKey:              ident.PublicKeyHex,
				AccountType:            accountType,
				IdentityProofSignature: identityProofSignature,
			}
			if err := identityLocation.assertReachable(); err != nil {
				return fmt.Errorf("identity custody path changed before broadcast: %w", err)
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
			fmt.Fprintln(guide, "  after key rotation securely retain and back up the newly current operational private key; this identity file alone cannot authorize the next rotation")
			return nil
		},
	}

	cmd.Flags().String("identity-out", "", "path for the generated identity file (default: <home>/identities/<from-address>.ed25519.json)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func signAccountRegistrationProof(
	identitySeed []byte,
	chainID string,
	sender string,
	did string,
	accountType string,
	metadata string,
) ([]byte, error) {
	if len(identitySeed) != ed25519.SeedSize {
		return nil, fmt.Errorf("identity seed must be %d bytes", ed25519.SeedSize)
	}
	identityPrivateKey := ed25519.NewKeyFromSeed(identitySeed)
	identityPublicKey := identityPrivateKey.Public().(ed25519.PublicKey)
	proofBytes, err := types.AccountRegistrationProofSignBytes(
		chainID,
		sender,
		did,
		identityPublicKey,
		accountType,
		metadata,
	)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(identityPrivateKey, proofBytes), nil
}

// CmdRegisterAccount registers a new Zerone account.
func CmdRegisterAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account [did] [public-key] [account-type] [identity-proof-signature-hex]",
		Short: "Register a new Zerone account with DID mapping",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			opKeyHash, _ := cmd.Flags().GetString("operational-key-hash")
			metadata, _ := cmd.Flags().GetString("metadata")
			identityProofSignature, err := hex.DecodeString(args[3])
			if err != nil {
				return fmt.Errorf("invalid identity proof signature hex: %w", err)
			}

			msg := &types.MsgRegisterAccount{
				Sender:                 clientCtx.GetFromAddress().String(),
				Did:                    args[0],
				PublicKey:              args[1],
				AccountType:            args[2],
				OperationalKeyHash:     opKeyHash,
				Metadata:               metadata,
				IdentityProofSignature: identityProofSignature,
			}
			if err := verifyAccountRegistrationProof(clientCtx.ChainID, msg); err != nil {
				return fmt.Errorf("identity proof preflight failed: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("operational-key-hash", "", "Hash of initial operational key")
	cmd.Flags().String("metadata", "", "Account metadata (JSON string)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdRegistrationSignBytes emits the domain-separated bytes the identity key
// must sign before CmdRegisterAccount is broadcast. It never reads or writes a
// private key.
func CmdRegistrationSignBytes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registration-sign-bytes [did] [public-key] [account-type]",
		Short: "Print exact account-registration identity proof bytes as hex",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			identityPublicKey, err := types.DecodeEd25519PublicKeyHex(args[1])
			if err != nil {
				return fmt.Errorf("invalid identity public key hex: %w", err)
			}
			metadata, err := cmd.Flags().GetString("metadata")
			if err != nil {
				return err
			}
			signBytes, err := types.AccountRegistrationProofSignBytes(
				clientCtx.ChainID,
				clientCtx.GetFromAddress().String(),
				args[0],
				identityPublicKey,
				args[2],
				metadata,
			)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), hex.EncodeToString(signBytes))
			return err
		},
	}

	cmd.Flags().String("metadata", "", "Account metadata (must match register-account exactly)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdVerifyRegistrationProof performs a fully offline proof preflight before
// a pre-signed registration transaction is submitted. This avoids consuming a
// fee and account sequence for malformed proof material.
func CmdVerifyRegistrationProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-registration-proof [sender] [did] [public-key] [account-type] [signature-hex]",
		Short: "Verify an account-registration identity proof offline",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainID, err := cmd.Flags().GetString(flags.FlagChainID)
			if err != nil {
				return err
			}
			metadata, err := cmd.Flags().GetString("metadata")
			if err != nil {
				return err
			}
			signature, err := hex.DecodeString(args[4])
			if err != nil {
				return fmt.Errorf("invalid identity proof signature hex: %w", err)
			}
			msg := &types.MsgRegisterAccount{
				Sender:                 args[0],
				Did:                    args[1],
				PublicKey:              args[2],
				AccountType:            args[3],
				Metadata:               metadata,
				IdentityProofSignature: signature,
			}
			if err := verifyAccountRegistrationProof(chainID, msg); err != nil {
				return fmt.Errorf("invalid registration proof: %w", err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "registration proof valid")
			return err
		},
	}
	cmd.Flags().String(flags.FlagChainID, "", "chain ID encoded into the registration proof (required)")
	cmd.Flags().String("metadata", "", "account metadata (must match register-account exactly)")
	_ = cmd.MarkFlagRequired(flags.FlagChainID)
	return cmd
}

func verifyAccountRegistrationProof(chainID string, msg *types.MsgRegisterAccount) error {
	if msg == nil {
		return fmt.Errorf("registration message cannot be nil")
	}
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	identityPublicKey, err := types.DecodeEd25519PublicKeyHex(msg.PublicKey)
	if err != nil {
		return err
	}
	proofBytes, err := types.AccountRegistrationProofSignBytes(
		chainID,
		msg.Sender,
		msg.Did,
		identityPublicKey,
		msg.AccountType,
		msg.Metadata,
	)
	if err != nil {
		return err
	}
	return types.VerifyEd25519Signature(identityPublicKey, proofBytes, msg.IdentityProofSignature)
}

// CmdRotateKey rotates the operational key.
func CmdRotateKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-key [new-op-key-hex] [auth-sig-hex] [new-key-confirmation-sig-hex]",
		Short: "Rotate the independent Ed25519 operational key",
		Long: `Rotate the independent Ed25519 operational key using authorization
from the current key and proof of possession by the proposed new key. Securely
retain and back up the new operational private key before broadcasting: it is
required to authorize every subsequent rotation, and social recovery is not
implemented on chain.`,
		Args: cobra.ExactArgs(3),
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
			newKeyConfirmationSig, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("invalid new key confirmation signature hex: %w", err)
			}

			expiresAtUnix, err := cmd.Flags().GetInt64("authorization-expires-at-unix")
			if err != nil {
				return err
			}

			msg := &types.MsgRotateKey{
				Sender:                      clientCtx.GetFromAddress().String(),
				NewOperationalKey:           newKey,
				AuthorizationSignature:      authSig,
				AuthorizationExpiresAtUnix:  expiresAtUnix,
				NewKeyConfirmationSignature: newKeyConfirmationSig,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Int64(
		"authorization-expires-at-unix",
		0,
		"expiry from the signed authorization (Unix seconds; required and at most 10 minutes after consensus block time)",
	)
	_ = cmd.MarkFlagRequired("authorization-expires-at-unix")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRotationSignBytes emits the domain-separated bytes either the current
// Ed25519 operational key must authorize or the proposed new key must accept
// before CmdRotateKey is broadcast. It never reads or writes a private key.
func CmdRotationSignBytes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotation-sign-bytes [new-op-key-hex]",
		Short: "Print exact operational-key rotation proof bytes as hex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			newKey, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid new key hex: %w", err)
			}
			currentVersion, err := cmd.Flags().GetUint32("current-key-version")
			if err != nil {
				return err
			}
			expiresAtUnix, err := cmd.Flags().GetInt64("authorization-expires-at-unix")
			if err != nil {
				return err
			}
			proof, err := cmd.Flags().GetString("proof")
			if err != nil {
				return err
			}

			var signBytes []byte
			switch proof {
			case "authorization":
				signBytes, err = types.KeyRotationAuthorizationSignBytes(
					clientCtx.ChainID,
					clientCtx.GetFromAddress().String(),
					currentVersion,
					newKey,
					expiresAtUnix,
				)
			case "acceptance":
				signBytes, err = types.KeyRotationAcceptanceSignBytes(
					clientCtx.ChainID,
					clientCtx.GetFromAddress().String(),
					currentVersion,
					newKey,
					expiresAtUnix,
				)
			default:
				return fmt.Errorf("proof must be authorization or acceptance (got %q)", proof)
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), hex.EncodeToString(signBytes))
			return err
		},
	}
	cmd.Flags().Uint32("current-key-version", 0, "current on-chain operational key version (required)")
	cmd.Flags().Int64("authorization-expires-at-unix", 0, "authorization expiry in Unix seconds (required)")
	cmd.Flags().String("proof", "authorization", "proof bytes to emit: authorization (current key) or acceptance (new key)")
	_ = cmd.MarkFlagRequired("current-key-version")
	_ = cmd.MarkFlagRequired("authorization-expires-at-unix")
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
