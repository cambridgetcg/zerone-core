package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/spf13/cobra"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/zerone-chain/zerone/app"
)

const (
	recoveryDigestFlagPlanName  = "plan-name"
	recoveryDigestFlagHeight    = "plan-height"
	recoveryDigestFlagPlanInfo  = "plan-info"
	recoveryDigestFlagSubmitter = "submitter"
	recoveryDigestFlagDeposit   = "deposit"
	recoveryDigestFlagTitle     = "title"
	recoveryDigestFlagSummary   = "summary"
	recoveryDigestFlagMetadata  = "metadata"
)

type recoveryActionAnyJSON struct {
	TypeURL     string `json:"type_url"`
	ValueBase64 string `json:"value_base64"`
}

type recoveryPlanJSON struct {
	Name   string `json:"name"`
	Height string `json:"height"`
	Info   string `json:"info,omitempty"`
}

type recoverySoftwareUpgradeJSON struct {
	Type      string           `json:"@type"`
	Authority string           `json:"authority"`
	Plan      recoveryPlanJSON `json:"plan"`
}

type recoveryCancelUpgradeJSON struct {
	Type      string `json:"@type"`
	Authority string `json:"authority"`
}

type recoveryProposalJSON struct {
	Messages  []json.RawMessage `json:"messages"`
	Metadata  string            `json:"metadata"`
	Deposit   string            `json:"deposit"`
	Title     string            `json:"title"`
	Summary   string            `json:"summary"`
	Expedited bool              `json:"expedited"`
}

type recoveryActionDigestOutput struct {
	SchemaVersion       string                `json:"schema_version"`
	ActionType          string                `json:"action_type"`
	AuthorizedSubmitter string                `json:"authorized_submitter"`
	ActionSHA256        string                `json:"action_sha256"`
	UpgradePlanSHA256   string                `json:"upgrade_plan_sha256"`
	ActionAny           recoveryActionAnyJSON `json:"action_any"`
	TargetPlan          recoveryPlanJSON      `json:"target_plan"`
	ProposalJSON        recoveryProposalJSON  `json:"proposal_json"`
}

func recoveryActionDigestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recovery-action-digest [software-upgrade|cancel-upgrade]",
		Short: "Build and hash an exact Guardian-authorized recovery action",
		Long: `Construct the exact protobuf Any admitted by the halted-chain
recovery lane and print its domain-separated action and target-plan SHA-256
digests together with an SDK-governance proposal JSON object. This command is
offline and read-only; independent Guardians should compare its complete output
before signing a recovery-authorization ceremony.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := buildRecoveryActionDigestOutput(cmd, args[0])
			if err != nil {
				return err
			}
			encoded, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf(
					"encode recovery action digest output: %w",
					err,
				)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
			return err
		},
	}
	flags := cmd.Flags()
	flags.String(recoveryDigestFlagPlanName, "", "exact target upgrade plan name")
	flags.Int64(recoveryDigestFlagHeight, 0, "exact target upgrade plan height")
	flags.String(recoveryDigestFlagPlanInfo, "", "exact target upgrade plan info")
	flags.String(recoveryDigestFlagSubmitter, "", "Guardian-authorized SDK governance submitter")
	flags.String(recoveryDigestFlagDeposit, "", "initial SDK governance deposit, for example 1000000uzrn")
	flags.String(recoveryDigestFlagTitle, "", "SDK governance proposal title")
	flags.String(recoveryDigestFlagSummary, "", "SDK governance proposal summary")
	flags.String(recoveryDigestFlagMetadata, "", "SDK governance proposal metadata")
	for _, name := range []string{
		recoveryDigestFlagPlanName,
		recoveryDigestFlagHeight,
		recoveryDigestFlagSubmitter,
		recoveryDigestFlagDeposit,
		recoveryDigestFlagTitle,
		recoveryDigestFlagSummary,
	} {
		_ = cmd.MarkFlagRequired(name)
	}
	return cmd
}

func buildRecoveryActionDigestOutput(
	cmd *cobra.Command,
	actionType string,
) (*recoveryActionDigestOutput, error) {
	planName, err := cmd.Flags().GetString(recoveryDigestFlagPlanName)
	if err != nil {
		return nil, err
	}
	planHeight, err := cmd.Flags().GetInt64(recoveryDigestFlagHeight)
	if err != nil {
		return nil, err
	}
	planInfo, err := cmd.Flags().GetString(recoveryDigestFlagPlanInfo)
	if err != nil {
		return nil, err
	}
	submitter, err := cmd.Flags().GetString(recoveryDigestFlagSubmitter)
	if err != nil {
		return nil, err
	}
	if _, err := sdk.AccAddressFromBech32(submitter); err != nil {
		return nil, fmt.Errorf("invalid authorized submitter: %w", err)
	}
	deposit, err := cmd.Flags().GetString(recoveryDigestFlagDeposit)
	if err != nil {
		return nil, err
	}
	if coins, err := sdk.ParseCoinsNormalized(deposit); err != nil ||
		coins.Empty() {
		return nil, fmt.Errorf(
			"invalid non-empty proposal deposit %q: %v",
			deposit,
			err,
		)
	}
	title, err := cmd.Flags().GetString(recoveryDigestFlagTitle)
	if err != nil {
		return nil, err
	}
	summary, err := cmd.Flags().GetString(recoveryDigestFlagSummary)
	if err != nil {
		return nil, err
	}
	metadata, err := cmd.Flags().GetString(recoveryDigestFlagMetadata)
	if err != nil {
		return nil, err
	}

	plan := upgradetypes.Plan{
		Name:   planName,
		Height: planHeight,
		Info:   planInfo,
	}
	if err := plan.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid target upgrade plan: %w", err)
	}
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	var (
		action     *codectypes.Any
		actionJSON json.RawMessage
	)
	switch actionType {
	case "software-upgrade":
		msg := &upgradetypes.MsgSoftwareUpgrade{
			Authority: authority,
			Plan:      plan,
		}
		action, err = codectypes.NewAnyWithValue(msg)
		if err == nil {
			actionJSON, err = json.Marshal(recoverySoftwareUpgradeJSON{
				Type:      action.TypeUrl,
				Authority: authority,
				Plan: recoveryPlanJSON{
					Name:   plan.Name,
					Height: strconv.FormatInt(plan.Height, 10),
					Info:   plan.Info,
				},
			})
		}
	case "cancel-upgrade":
		msg := &upgradetypes.MsgCancelUpgrade{Authority: authority}
		action, err = codectypes.NewAnyWithValue(msg)
		if err == nil {
			actionJSON, err = json.Marshal(recoveryCancelUpgradeJSON{
				Type:      action.TypeUrl,
				Authority: authority,
			})
		}
	default:
		return nil, fmt.Errorf(
			"unsupported recovery action %q; use software-upgrade or cancel-upgrade",
			actionType,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("construct canonical recovery action: %w", err)
	}

	canonicalActionType := "software_upgrade"
	if actionType == "cancel-upgrade" {
		canonicalActionType = "cancel_upgrade"
	}
	targetPlan := recoveryPlanJSON{
		Name:   plan.Name,
		Height: strconv.FormatInt(plan.Height, 10),
		Info:   plan.Info,
	}
	return &recoveryActionDigestOutput{
		SchemaVersion:       "zerone.recovery-action-digest/v1",
		ActionType:          canonicalActionType,
		AuthorizedSubmitter: submitter,
		ActionSHA256:        app.RecoveryActionSHA256(action),
		UpgradePlanSHA256:   app.UpgradePlanSHA256(plan),
		ActionAny: recoveryActionAnyJSON{
			TypeURL:     action.TypeUrl,
			ValueBase64: base64.StdEncoding.EncodeToString(action.Value),
		},
		TargetPlan: targetPlan,
		ProposalJSON: recoveryProposalJSON{
			Messages:  []json.RawMessage{actionJSON},
			Metadata:  metadata,
			Deposit:   deposit,
			Title:     title,
			Summary:   summary,
			Expedited: true,
		},
	}, nil
}
