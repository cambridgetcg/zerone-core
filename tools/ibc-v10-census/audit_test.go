package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	testFeeAddress = "zrn176rcyfn5k9d0wcxel3kmwvxh0hy3xcweksmdhj"
	testLocalA     = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf"
	testLocalB     = "zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh"
	testAppHash    = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
)

func TestAuditCleanExportStillRequiresRawStoreRehearsal(t *testing.T) {
	input := marshalFixture(t, cleanFixture())
	report, err := auditDocument(input, "fixture.json", Evidence{
		ExportHeight: "123",
		AppHash:      testAppHash,
	})
	if err != nil {
		t.Fatalf("audit clean export: %v", err)
	}
	if report.Schema != reportSchema {
		t.Fatalf("schema = %q, want %q", report.Schema, reportSchema)
	}
	if report.Coverage.InputKind != "zeroned_export" {
		t.Fatalf("input kind = %q", report.Coverage.InputKind)
	}
	if report.Summary.Errors != 1 || report.Summary.Warnings != 0 {
		t.Fatalf("findings = %d errors, %d warnings", report.Summary.Errors, report.Summary.Warnings)
	}
	assertFinding(t, report, "OLD_DATABASE_REHEARSAL_REQUIRED")
	if report.Complete || report.UpgradeReady {
		t.Fatalf("export-only report must remain incomplete and not upgrade-ready: %+v", report)
	}
	digest := sha256.Sum256(input)
	if report.Evidence.InputSHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("input digest = %q", report.Evidence.InputSHA256)
	}
	if report.FeeIBC.ModuleAccountAddress != testFeeAddress {
		t.Fatalf("fee module address = %q", report.FeeIBC.ModuleAccountAddress)
	}
	if len(report.FeeIBC.ModuleAccountBalances) != 0 {
		t.Fatalf("unexpected fee balance: %+v", report.FeeIBC.ModuleAccountBalances)
	}
	if report.Channels == nil {
		t.Fatal("empty channel report must encode as [] rather than null")
	}
	if !containsText(report.Coverage.DoesNotProve, "persistent locked key") {
		t.Fatal("coverage does not name omitted feeibc locked key")
	}
	if !containsText(report.Coverage.DoesNotProve, "bilateral in-flight clearance") {
		t.Fatal("coverage does not require bilateral clearance")
	}
}

func TestAuditEnumeratesV8FeeAndPacketState(t *testing.T) {
	fixture := cleanFixture()
	populateFeeAndPacketState(fixture)
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{
		ExportHeight: "456",
		AppHash:      testAppHash,
	})
	if err != nil {
		t.Fatalf("audit populated export: %v", err)
	}

	if report.Summary.FeeEnabledChannels != 1 ||
		report.Summary.IdentifiedPacketFeeRecords != 1 ||
		report.Summary.PacketFees != 1 ||
		report.Summary.RegisteredPayees != 1 ||
		report.Summary.RegisteredCounterpartyPayees != 1 ||
		report.Summary.ForwardRelayers != 1 {
		t.Fatalf("unexpected fee summary: %+v", report.Summary)
	}
	if report.Summary.OutstandingCommitments != 1 ||
		report.Summary.ExportedAcknowledgements != 1 ||
		report.Summary.ExportedReceipts != 1 {
		t.Fatalf("unexpected packet-state summary: %+v", report.Summary)
	}
	if got := formatCoins(report.FeeIBC.CalculatedEscrowObligations); got != "5uzrn" {
		t.Fatalf("calculated escrow = %q, want 5uzrn", got)
	}
	if got := formatCoins(report.FeeIBC.ModuleAccountBalances); got != "5uzrn" {
		t.Fatalf("module balance = %q, want 5uzrn", got)
	}
	if report.Channels[0].FeeVersion != "ics29-1" || report.Channels[0].AppVersion != "ics20-1" {
		t.Fatalf("fee metadata not decoded: %+v", report.Channels[0])
	}
	if report.ExportedPacketState.Commitments[0].DataSHA256 == "" ||
		report.ExportedPacketState.Acknowledgements[0].DataSHA256 == "" ||
		report.ExportedPacketState.Receipts[0].DataSHA256 == "" {
		t.Fatal("packet-state digests must be populated")
	}
	for _, code := range []string{
		"FEE_ENABLED_CHANNELS_PRESENT",
		"IDENTIFIED_PACKET_FEES_PRESENT",
		"REGISTERED_PAYEES_PRESENT",
		"REGISTERED_COUNTERPARTY_PAYEES_PRESENT",
		"FORWARD_RELAYER_STATE_PRESENT",
		"FEEIBC_MODULE_BALANCE_NONZERO",
		"ICS29_CHANNEL_VERSIONS_PRESENT",
		"OUTSTANDING_PACKET_COMMITMENTS_PRESENT",
	} {
		assertFinding(t, report, code)
	}
	if hasFinding(report, "EXPORTED_ACKNOWLEDGEMENTS_PENDING") ||
		hasFinding(report, "EXPORTED_RECEIPTS_PENDING") {
		t.Fatal("acknowledgements and receipts must not be mislabeled as pending")
	}
}

func TestFeeTotalUsesDenomwiseMaxOfRecvPlusAckAndTimeout(t *testing.T) {
	fixture := cleanFixture()
	populateFeeAndPacketState(fixture)
	fee := feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_fees"].([]any)[0].(map[string]any)["fee"].(map[string]any)
	fee["recv_fee"] = []any{
		map[string]any{"denom": "uatom", "amount": "7"},
		map[string]any{"denom": "uzrn", "amount": "1"},
	}
	fee["ack_fee"] = []any{
		map[string]any{"denom": "uatom", "amount": "2"},
		map[string]any{"denom": "uzrn", "amount": "2"},
	}
	fee["timeout_fee"] = []any{
		map[string]any{"denom": "uatom", "amount": "4"},
		map[string]any{"denom": "uzrn", "amount": "5"},
	}
	bankState(fixture)["balances"] = []any{map[string]any{
		"address": testFeeAddress,
		"coins": []any{
			map[string]any{"denom": "uatom", "amount": "9"},
			map[string]any{"denom": "uzrn", "amount": "5"},
		},
	}}

	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit multi-denom fee: %v", err)
	}
	if got := formatCoins(report.FeeIBC.CalculatedEscrowObligations); got != "9uatom,5uzrn" {
		t.Fatalf("calculated escrow = %q", got)
	}
	if hasFinding(report, "FEEIBC_ESCROW_BALANCE_MISMATCH") {
		t.Fatal("correct module balance reported as mismatch")
	}
}

func TestAuditFlagsEscrowMismatchWithoutMisclassifyingPreSendFee(t *testing.T) {
	fixture := cleanFixture()
	populateFeeAndPacketState(fixture)
	bankState(fixture)["balances"] = []any{map[string]any{
		"address": testFeeAddress,
		"coins":   []any{map[string]any{"denom": "uzrn", "amount": "4"}},
	}}
	channelState(fixture)["commitments"] = []any{}

	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit mismatch: %v", err)
	}
	assertFinding(t, report, "FEEIBC_ESCROW_BALANCE_MISMATCH")
	if hasFinding(report, "PACKET_FEE_WITHOUT_COMMITMENT") {
		t.Fatal("a valid pre-send fee escrow must not be called inconsistent")
	}
}

func TestAuditFlagsBothV10BlockingChannelStates(t *testing.T) {
	fixture := cleanFixture()
	channelState(fixture)["channels"] = []any{
		channelRecord("channel-2", "STATE_FLUSHCOMPLETE", "ics20-1"),
		channelRecord("channel-1", "STATE_FLUSHING", "ics20-1"),
	}

	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit flushing channels: %v", err)
	}
	if report.Summary.FlushingChannels != 2 {
		t.Fatalf("flushing channels = %d", report.Summary.FlushingChannels)
	}
	assertFinding(t, report, "CHANNEL_UPGRADE_STATE_BLOCKS_V10_MIGRATION")
	if report.Channels[0].ChannelID != "channel-1" || report.Channels[1].ChannelID != "channel-2" {
		t.Fatalf("channels are not deterministic: %+v", report.Channels)
	}
}

func TestAuditRejectsAmbiguousOrMalformedV8Schema(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "duplicate fee module account",
			mutate: func(fixture map[string]any) {
				account := authState(fixture)["accounts"].([]any)[0]
				authState(fixture)["accounts"] = append(authState(fixture)["accounts"].([]any), account)
			},
			want: "multiple auth ModuleAccount records",
		},
		{
			name: "fee name on wrong account type",
			mutate: func(fixture map[string]any) {
				authState(fixture)["accounts"].([]any)[0].(map[string]any)["@type"] = "/cosmos.auth.v1beta1.BaseAccount"
			},
			want: "names feeibc but has account type",
		},
		{
			name: "fee module account address mismatch",
			mutate: func(fixture map[string]any) {
				authState(fixture)["accounts"].([]any)[0].(map[string]any)["base_account"].(map[string]any)["address"] = testLocalA
			},
			want: "does not equal deterministic address",
		},
		{
			name: "fee module account permission",
			mutate: func(fixture map[string]any) {
				authState(fixture)["accounts"].([]any)[0].(map[string]any)["permissions"] = []any{"minter"}
			},
			want: "must have no permissions",
		},
		{
			name: "fee module account public key",
			mutate: func(fixture map[string]any) {
				authState(fixture)["accounts"].([]any)[0].(map[string]any)["base_account"].(map[string]any)["pub_key"] = map[string]any{
					"@type": "/cosmos.crypto.secp256k1.PubKey",
					"key":   "AA==",
				}
			},
			want: "base_account.pub_key must be null",
		},
		{
			name: "duplicate bank balance address",
			mutate: func(fixture map[string]any) {
				balance := map[string]any{"address": testFeeAddress, "coins": []any{map[string]any{"denom": "uzrn", "amount": "1"}}}
				bankState(fixture)["balances"] = []any{balance, balance}
			},
			want: "duplicate bank balance address",
		},
		{
			name: "missing forward relayers field",
			mutate: func(fixture map[string]any) {
				delete(feeState(fixture), "forward_relayers")
			},
			want: "required field feeibc.forward_relayers is missing",
		},
		{
			name: "unexpected fee field",
			mutate: func(fixture map[string]any) {
				feeState(fixture)["locked"] = true
			},
			want: "unexpected field feeibc.locked",
		},
		{
			name: "numeric protobuf sequence",
			mutate: func(fixture map[string]any) {
				populateFeeAndPacketState(fixture)
				feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_id"].(map[string]any)["sequence"] = float64(42)
			},
			want: "must be a JSON string",
		},
		{
			name: "unknown channel state",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("channel-1", "OPEN", "ics20-1")}
			},
			want: "unknown IBC-Go v8 value",
		},
		{
			name: "nonzero channel upgrade timeout height",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["params"].(map[string]any)["upgrade_timeout"].(map[string]any)["height"].(map[string]any)["revision_height"] = "1"
			},
			want: "upgrade_timeout.height must be zero",
		},
		{
			name: "zero channel upgrade timeout timestamp",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["params"].(map[string]any)["upgrade_timeout"].(map[string]any)["timestamp"] = "0"
			},
			want: "upgrade_timeout.timestamp must be nonzero",
		},
		{
			name: "invalid channel ordering for v10",
			mutate: func(fixture map[string]any) {
				record := channelRecord("channel-1", "STATE_OPEN", "ics20-1")
				record["ordering"] = "ORDER_NONE_UNSPECIFIED"
				channelState(fixture)["channels"] = []any{record}
			},
			want: "would fail IBC-Go v10 Channel.ValidateBasic",
		},
		{
			name: "multiple connection hops",
			mutate: func(fixture map[string]any) {
				record := channelRecord("channel-1", "STATE_OPEN", "ics20-1")
				record["connection_hops"] = []any{"connection-0", "connection-1"}
				channelState(fixture)["channels"] = []any{record}
			},
			want: "requires exactly one",
		},
		{
			name: "invalid channel identifier",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("bad/id", "STATE_OPEN", "ics20-1")}
			},
			want: "not an ICS-24 identifier",
		},
		{
			name: "non SDK channel identifier",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("abcdefgh", "STATE_OPEN", "ics20-1")}
			},
			want: "SDK format channel-{N}",
		},
		{
			name: "noncanonical SDK channel identifier",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("channel-00", "STATE_OPEN", "ics20-1")}
			},
			want: "not canonical channel-{N}",
		},
		{
			name: "channel zero still requires next sequence",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("channel-0", "STATE_OPEN", "ics20-1")}
				channelState(fixture)["next_channel_sequence"] = "0"
			},
			want: "is not greater than maximum",
		},
		{
			name: "next channel sequence not ahead",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("channel-7", "STATE_OPEN", "ics20-1")}
				channelState(fixture)["next_channel_sequence"] = "7"
			},
			want: "is not greater than maximum",
		},
		{
			name: "invalid counterparty port identifier",
			mutate: func(fixture map[string]any) {
				record := channelRecord("channel-1", "STATE_OPEN", "ics20-1")
				record["counterparty"].(map[string]any)["port_id"] = "bad/port"
				channelState(fixture)["channels"] = []any{record}
			},
			want: "not an ICS-24 identifier",
		},
		{
			name: "invalid nonempty counterparty channel identifier",
			mutate: func(fixture map[string]any) {
				record := channelRecord("channel-1", "STATE_OPEN", "ics20-1")
				record["counterparty"].(map[string]any)["channel_id"] = "bad/id"
				channelState(fixture)["channels"] = []any{record}
			},
			want: "not an ICS-24 identifier",
		},
		{
			name: "invalid packet fee refund address",
			mutate: func(fixture map[string]any) {
				populateFeeAndPacketState(fixture)
				feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_fees"].([]any)[0].(map[string]any)["refund_address"] = "zrn1invalid"
			},
			want: "must be a Zerone zrn Bech32 account address",
		},
		{
			name: "SDK invalid fee denomination",
			mutate: func(fixture map[string]any) {
				populateFeeAndPacketState(fixture)
				feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_fees"].([]any)[0].(map[string]any)["fee"].(map[string]any)["recv_fee"].([]any)[0].(map[string]any)["denom"] = "x"
			},
			want: "not a Cosmos SDK v0.50 denomination",
		},
		{
			name: "fee amount above SDK bit bound",
			mutate: func(fixture map[string]any) {
				populateFeeAndPacketState(fixture)
				feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_fees"].([]any)[0].(map[string]any)["fee"].(map[string]any)["recv_fee"].([]any)[0].(map[string]any)["amount"] = "115792089237316195423570985008687907853269984665640564039457584007913129639936"
			},
			want: "exceeds the Cosmos SDK 256-bit integer bound",
		},
		{
			name: "fee total above SDK bit bound",
			mutate: func(fixture map[string]any) {
				populateFeeAndPacketState(fixture)
				fee := feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_fees"].([]any)[0].(map[string]any)["fee"].(map[string]any)
				fee["recv_fee"].([]any)[0].(map[string]any)["amount"] = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
				fee["ack_fee"].([]any)[0].(map[string]any)["amount"] = "1"
			},
			want: "fee total",
		},
		{
			name: "equal registered relayer and payee",
			mutate: func(fixture map[string]any) {
				feeState(fixture)["registered_payees"] = []any{map[string]any{
					"channel_id": "channel-1", "relayer": testLocalA, "payee": testLocalA,
				}}
			},
			want: "relayer and payee must differ",
		},
		{
			name: "blank counterparty payee",
			mutate: func(fixture map[string]any) {
				feeState(fixture)["registered_counterparty_payees"] = []any{map[string]any{
					"channel_id": "channel-1", "relayer": testLocalA, "counterparty_payee": "   ",
				}}
			},
			want: "must not be blank",
		},
		{
			name: "bad packet state base64",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["receipts"] = []any{packetState("channel-1", "1", "not-base64")}
			},
			want: "not canonical padded base64",
		},
		{
			name: "embedded fee metadata missing app version",
			mutate: func(fixture map[string]any) {
				channelState(fixture)["channels"] = []any{channelRecord("channel-1", "STATE_OPEN", `{"fee_version":"ics29-1"}`)}
			},
			want: "required field",
		},
		{
			name: "partial app state",
			mutate: func(fixture map[string]any) {
				delete(appState(fixture), "bank")
			},
			want: "missing the bank module",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := cleanFixture()
			test.mutate(fixture)
			_, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestChannelOpenInitAllowsEmptyCounterpartyChannelID(t *testing.T) {
	fixture := cleanFixture()
	record := channelRecord("channel-1", "STATE_INIT", "")
	record["counterparty"].(map[string]any)["channel_id"] = ""
	channelState(fixture)["channels"] = []any{record}
	if _, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{}); err != nil {
		t.Fatalf("valid INIT counterparty rejected: %v", err)
	}
}

func TestReceiptMayHaveEmptyDataButAcknowledgementMayNot(t *testing.T) {
	fixture := cleanFixture()
	channelState(fixture)["channels"] = []any{channelRecord("channel-1", "STATE_OPEN", "ics20-1")}
	channelState(fixture)["receipts"] = []any{packetState("channel-1", "1", "")}
	if _, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{}); err != nil {
		t.Fatalf("v8-valid empty receipt rejected: %v", err)
	}
	channelState(fixture)["acknowledgements"] = []any{packetState("channel-1", "1", "")}
	if _, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{}); err == nil ||
		!strings.Contains(err.Error(), "empty data") {
		t.Fatalf("empty acknowledgement error = %v", err)
	}
}

func TestAuditRejectsDuplicateJSONKeys(t *testing.T) {
	data := marshalFixture(t, cleanFixture())
	needle := `"identified_fees":[]`
	replacement := `"identified_fees":[],"identified_fees":[]`
	data = []byte(strings.Replace(string(data), needle, replacement, 1))
	_, err := auditDocument(data, "fixture.json", Evidence{})
	if err == nil || !strings.Contains(err.Error(), `duplicate JSON key "identified_fees"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestLazyFeeModuleAccountUsesLockedDeterministicAddress(t *testing.T) {
	fixture := cleanFixture()
	authState(fixture)["accounts"] = []any{}
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit unused lazy account: %v", err)
	}
	if report.FeeIBC.AuthModuleAccountExported {
		t.Fatal("missing lazy account reported as exported")
	}
	if report.FeeIBC.ModuleAccountAddress != feeModuleAddress {
		t.Fatalf("derived address = %q", report.FeeIBC.ModuleAccountAddress)
	}
	assertFinding(t, report, "FEEIBC_AUTH_MODULE_ACCOUNT_NOT_EXPORTED")
	if hasFinding(report, "FEEIBC_AUTH_MODULE_ACCOUNT_REQUIRED_BY_STATE") {
		t.Fatal("unused lazy module account absence must not be elevated")
	}

	populateFeeAndPacketState(fixture)
	report, err = auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit active lazy account: %v", err)
	}
	assertFinding(t, report, "FEEIBC_AUTH_MODULE_ACCOUNT_REQUIRED_BY_STATE")
}

func TestFeeModuleAddressMatchesCosmosSDKFixedVector(t *testing.T) {
	digest := sha256.Sum256([]byte(feeModuleName))
	if got := hex.EncodeToString(digest[:20]); got != feeModuleAddressHashHex {
		t.Fatalf("SHA256 feeibc truncated = %s, want %s", got, feeModuleAddressHashHex)
	}
	moduleAddress := authtypes.NewModuleAddress(feeModuleName)
	if got := hex.EncodeToString(moduleAddress); got != feeModuleAddressHashHex {
		t.Fatalf("SDK module address bytes = %s, want %s", got, feeModuleAddressHashHex)
	}
	rendered, err := bech32.ConvertAndEncode("zrn", moduleAddress)
	if err != nil {
		t.Fatalf("render module address: %v", err)
	}
	if rendered != feeModuleAddress {
		t.Fatalf("SDK module address = %s, want %s", rendered, feeModuleAddress)
	}
}

func TestAuditRequiresRawStoreEvidenceForNonzeroUpgradeSequence(t *testing.T) {
	fixture := cleanFixture()
	record := channelRecord("channel-7", "STATE_OPEN", "ics20-1")
	record["upgrade_sequence"] = "3"
	channelState(fixture)["channels"] = []any{record}
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit upgrade sequence: %v", err)
	}
	assertFinding(t, report, "CHANNEL_UPGRADE_SEQUENCE_AMBIGUOUS")
}

func TestAuditCrossChecksPacketAndSequenceChannelReferences(t *testing.T) {
	fixture := cleanFixture()
	channelState(fixture)["channels"] = []any{channelRecord("channel-7", "STATE_OPEN", "ics20-1")}
	channelState(fixture)["commitments"] = []any{packetState("channel-8", "1", "AQ==")}
	channelState(fixture)["send_sequences"] = []any{packetSequence("channel-9", "1")}
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit missing channel references: %v", err)
	}
	assertFinding(t, report, "PACKET_STATE_CHANNEL_MISSING")
	assertFinding(t, report, "PACKET_SEQUENCE_CHANNEL_MISSING")
}

func TestAuditReportsV8InvalidPacketFeeRelayerAllowlist(t *testing.T) {
	fixture := cleanFixture()
	populateFeeAndPacketState(fixture)
	feeState(fixture)["identified_fees"].([]any)[0].(map[string]any)["packet_fees"].([]any)[0].(map[string]any)["relayers"] = []any{testLocalB}
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit relayer allowlist: %v", err)
	}
	assertFinding(t, report, "PACKET_FEE_RELAYER_ALLOWLIST_PRESENT")
}

func TestLogicalReportOrderingDoesNotFollowExportArrayOrder(t *testing.T) {
	fixture := cleanFixture()
	channelState(fixture)["channels"] = []any{
		channelRecord("channel-10", "STATE_OPEN", "ics20-1"),
		channelRecord("channel-2", "STATE_OPEN", "ics20-1"),
		channelRecord("channel-1", "STATE_OPEN", "ics20-1"),
	}
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	got := []string{
		report.Channels[0].ChannelID,
		report.Channels[1].ChannelID,
		report.Channels[2].ChannelID,
	}
	want := []string{"channel-1", "channel-10", "channel-2"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("channel order = %v, want %v", got, want)
	}
}

func cleanFixture() map[string]any {
	return map[string]any{
		"app_state": map[string]any{
			"auth": map[string]any{
				"params": map[string]any{
					"max_memo_characters":       "256",
					"tx_sig_limit":              "7",
					"tx_size_cost_per_byte":     "10",
					"sig_verify_cost_ed25519":   "590",
					"sig_verify_cost_secp256k1": "1000",
				},
				"accounts": []any{map[string]any{
					"@type": "/cosmos.auth.v1beta1.ModuleAccount",
					"base_account": map[string]any{
						"address":        testFeeAddress,
						"pub_key":        nil,
						"account_number": "7",
						"sequence":       "0",
					},
					"name":        "feeibc",
					"permissions": []any{},
				}},
			},
			"bank": map[string]any{
				"params": map[string]any{
					"send_enabled":         []any{},
					"default_send_enabled": true,
				},
				"balances":       []any{},
				"supply":         []any{},
				"denom_metadata": []any{},
				"send_enabled":   []any{},
			},
			"feeibc": map[string]any{
				"identified_fees":                []any{},
				"fee_enabled_channels":           []any{},
				"registered_payees":              []any{},
				"registered_counterparty_payees": []any{},
				"forward_relayers":               []any{},
			},
			"ibc": map[string]any{
				"client_genesis":     map[string]any{},
				"connection_genesis": map[string]any{},
				"channel_genesis": map[string]any{
					"channels":              []any{},
					"acknowledgements":      []any{},
					"commitments":           []any{},
					"receipts":              []any{},
					"send_sequences":        []any{},
					"recv_sequences":        []any{},
					"ack_sequences":         []any{},
					"next_channel_sequence": "100",
					"params": map[string]any{
						"upgrade_timeout": map[string]any{
							"height": map[string]any{
								"revision_number": "0",
								"revision_height": "0",
							},
							"timestamp": "600000000000",
						},
					},
				},
			},
		},
	}
}

func populateFeeAndPacketState(fixture map[string]any) {
	feeState(fixture)["identified_fees"] = []any{map[string]any{
		"packet_id": packetRef("channel-7", "42"),
		"packet_fees": []any{map[string]any{
			"fee": map[string]any{
				"recv_fee":    []any{map[string]any{"denom": "uzrn", "amount": "1"}},
				"ack_fee":     []any{map[string]any{"denom": "uzrn", "amount": "2"}},
				"timeout_fee": []any{map[string]any{"denom": "uzrn", "amount": "5"}},
			},
			"refund_address": testLocalA,
			"relayers":       []any{},
		}},
	}}
	feeState(fixture)["fee_enabled_channels"] = []any{map[string]any{
		"port_id": "transfer", "channel_id": "channel-7",
	}}
	feeState(fixture)["registered_payees"] = []any{map[string]any{
		"channel_id": "channel-7", "relayer": testLocalA, "payee": testLocalB,
	}}
	feeState(fixture)["registered_counterparty_payees"] = []any{map[string]any{
		"channel_id": "channel-7", "relayer": testLocalA, "counterparty_payee": "other1payee",
	}}
	feeState(fixture)["forward_relayers"] = []any{map[string]any{
		"address": testLocalA, "packet_id": packetRef("channel-7", "42"),
	}}
	bankState(fixture)["balances"] = []any{map[string]any{
		"address": testFeeAddress,
		"coins":   []any{map[string]any{"denom": "uzrn", "amount": "5"}},
	}}
	channelState(fixture)["channels"] = []any{channelRecord(
		"channel-7",
		"STATE_OPEN",
		`{"fee_version":"ics29-1","app_version":"ics20-1"}`,
	)}
	channelState(fixture)["commitments"] = []any{packetState("channel-7", "42", "AwQ=")}
	channelState(fixture)["acknowledgements"] = []any{packetState("channel-7", "40", "AQI=")}
	channelState(fixture)["receipts"] = []any{packetState("channel-7", "41", "AQ==")}
	channelState(fixture)["send_sequences"] = []any{packetSequence("channel-7", "43")}
	channelState(fixture)["recv_sequences"] = []any{packetSequence("channel-7", "42")}
	channelState(fixture)["ack_sequences"] = []any{packetSequence("channel-7", "41")}
}

func channelRecord(channelID, state, version string) map[string]any {
	return map[string]any{
		"state":            state,
		"ordering":         "ORDER_UNORDERED",
		"counterparty":     map[string]any{"port_id": "transfer", "channel_id": "channel-counterparty"},
		"connection_hops":  []any{"connection-0"},
		"version":          version,
		"port_id":          "transfer",
		"channel_id":       channelID,
		"upgrade_sequence": "0",
	}
}

func packetRef(channelID, sequence string) map[string]any {
	return map[string]any{"port_id": "transfer", "channel_id": channelID, "sequence": sequence}
}

func packetState(channelID, sequence, data string) map[string]any {
	state := packetRef(channelID, sequence)
	state["data"] = data
	return state
}

func packetSequence(channelID, sequence string) map[string]any {
	return packetRef(channelID, sequence)
}

func appState(fixture map[string]any) map[string]any {
	return fixture["app_state"].(map[string]any)
}

func authState(fixture map[string]any) map[string]any {
	return appState(fixture)["auth"].(map[string]any)
}

func bankState(fixture map[string]any) map[string]any {
	return appState(fixture)["bank"].(map[string]any)
}

func feeState(fixture map[string]any) map[string]any {
	return appState(fixture)["feeibc"].(map[string]any)
}

func channelState(fixture map[string]any) map[string]any {
	return appState(fixture)["ibc"].(map[string]any)["channel_genesis"].(map[string]any)
}

func marshalFixture(t *testing.T, fixture map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}

func assertFinding(t *testing.T, report Report, code string) {
	t.Helper()
	if !hasFinding(report, code) {
		data, _ := json.MarshalIndent(report.Findings, "", "  ")
		t.Fatalf("finding %s missing from %s", code, data)
	}
}

func hasFinding(report Report, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func containsText(values []string, substring string) bool {
	for _, value := range values {
		if strings.Contains(value, substring) {
			return true
		}
	}
	return false
}

func TestTextOutputDistinguishesExportedPacketState(t *testing.T) {
	fixture := cleanFixture()
	populateFeeAndPacketState(fixture)
	report, err := auditDocument(marshalFixture(t, fixture), "fixture.json", Evidence{
		ExportHeight: "456",
		AppHash:      testAppHash,
	})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	var output bytes.Buffer
	if err := printText(&output, report); err != nil {
		t.Fatalf("print text: %v", err)
	}
	if !strings.Contains(output.String(), "Exported packet state: commitments=1 acknowledgements=1 receipts=1") {
		t.Fatalf("text output missing lifecycle-neutral packet counts:\n%s", output.String())
	}
}
