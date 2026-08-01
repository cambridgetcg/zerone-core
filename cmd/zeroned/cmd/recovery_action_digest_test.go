package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRecoveryActionDigestGoldenVectors(t *testing.T) {
	const (
		submitter = "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r"
		planSHA   = "5e3351b1bc7f14dde7e1a54116417fd391586f87aa0b44b44118121577491d72"
	)
	tests := []struct {
		name              string
		action            string
		expectedType      string
		expectedActionSHA string
		expectedAnyValue  string
	}{
		{
			name:              "software upgrade",
			action:            "software-upgrade",
			expectedType:      "software_upgrade",
			expectedActionSHA: "4be4901d22b6ac84e33730b1f963522655c618fbd3727fa2020d16b66ece5344",
			expectedAnyValue:  "Cip6cm4xMGQwN3kyNjVnbW11dnQ0ejB3OWF3ODgwam5zcjcwMGo0N3R0ODkSOQoLcmVjb3ZlcnktdjISCwiAkrjDmP7///8BGIakPCIZeyJhcnRpZmFjdCI6InNoYTI1NjphYmMifQ==",
		},
		{
			name:              "cancel upgrade",
			action:            "cancel-upgrade",
			expectedType:      "cancel_upgrade",
			expectedActionSHA: "d7cd3d22085075b29605f662548ad856e3e23285f1c498e0ae700b5bae615c9f",
			expectedAnyValue:  "Cip6cm4xMGQwN3kyNjVnbW11dnQ0ejB3OWF3ODgwam5zcjcwMGo0N3R0ODk=",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := recoveryActionDigestCmd()
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs([]string{
				test.action,
				"--plan-name", "recovery-v2",
				"--plan-height", "987654",
				"--plan-info", `{"artifact":"sha256:abc"}`,
				"--submitter", submitter,
				"--deposit", "1000000uzrn",
				"--title", "Emergency recovery v2",
				"--summary", "Guardian-bound forward recovery",
			})
			if err := command.Execute(); err != nil {
				t.Fatalf("execute digest command: %v\n%s", err, output.String())
			}

			var decoded recoveryActionDigestOutput
			if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatalf(
					"decode command output: %v\n%s",
					err,
					output.String(),
				)
			}
			if decoded.SchemaVersion !=
				"zerone.recovery-action-digest/v1" ||
				decoded.ActionType != test.expectedType ||
				decoded.AuthorizedSubmitter != submitter ||
				decoded.ActionSHA256 != test.expectedActionSHA ||
				decoded.UpgradePlanSHA256 != planSHA ||
				decoded.ActionAny.ValueBase64 != test.expectedAnyValue ||
				len(decoded.ProposalJSON.Messages) != 1 ||
				!decoded.ProposalJSON.Expedited {
				t.Fatalf("golden recovery digest drifted: %+v", decoded)
			}
		})
	}
}

func TestRecoveryActionDigestRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		action string
		height string
	}{
		{name: "unknown action", action: "arbitrary", height: "100"},
		{name: "invalid plan height", action: "software-upgrade", height: "0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := recoveryActionDigestCmd()
			command.SetArgs([]string{
				test.action,
				"--plan-name", "recovery-v2",
				"--plan-height", test.height,
				"--submitter",
				"zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r",
				"--deposit", "1000000uzrn",
				"--title", "title",
				"--summary", "summary",
			})
			if err := command.Execute(); err == nil {
				t.Fatal("invalid recovery digest input was accepted")
			}
		})
	}
}
