package main

import (
	"fmt"
	"io"
	"strings"
)

const (
	severityError   = "error"
	severityWarning = "warning"
)

// Evidence binds a census to operator-recorded consensus evidence. The app
// hash is not part of the normal zeroned export document and must be captured
// independently at the same height.
type Evidence struct {
	ExportHeight string `json:"export_height"`
	AppHash      string `json:"app_hash"`
	InputSHA256  string `json:"input_sha256"`
}

type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
	Subject  string `json:"subject,omitempty"`
}

type Coverage struct {
	InputKind    string   `json:"input_kind"`
	Proves       []string `json:"proves"`
	DoesNotProve []string `json:"does_not_prove"`
}

type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type ChannelRef struct {
	PortID    string `json:"port_id"`
	ChannelID string `json:"channel_id"`
}

func (ref ChannelRef) String() string {
	return ref.PortID + "/" + ref.ChannelID
}

type PacketRef struct {
	PortID    string `json:"port_id"`
	ChannelID string `json:"channel_id"`
	Sequence  string `json:"sequence"`
}

func (ref PacketRef) String() string {
	return ref.PortID + "/" + ref.ChannelID + "/" + ref.Sequence
}

type PacketFeeRecord struct {
	Packet        PacketRef `json:"packet"`
	RefundAddress string    `json:"refund_address"`
	Relayers      []string  `json:"relayers"`
	RecvFee       []Coin    `json:"recv_fee"`
	AckFee        []Coin    `json:"ack_fee"`
	TimeoutFee    []Coin    `json:"timeout_fee"`
	EscrowAmount  []Coin    `json:"escrow_amount"`
}

type PayeeRecord struct {
	ChannelID string `json:"channel_id"`
	Relayer   string `json:"relayer"`
	Payee     string `json:"payee"`
}

type CounterpartyPayeeRecord struct {
	ChannelID         string `json:"channel_id"`
	Relayer           string `json:"relayer"`
	CounterpartyPayee string `json:"counterparty_payee"`
}

type ForwardRelayerRecord struct {
	Address string    `json:"address"`
	Packet  PacketRef `json:"packet"`
}

type FeeIBCReport struct {
	ModuleAccountAddress         string                    `json:"module_account_address"`
	AuthModuleAccountExported    bool                      `json:"auth_module_account_exported"`
	ModuleAccountAddressSource   string                    `json:"module_account_address_source"`
	ModuleAccountBalances        []Coin                    `json:"module_account_balances"`
	CalculatedEscrowObligations  []Coin                    `json:"calculated_escrow_obligations"`
	FeeEnabledChannels           []ChannelRef              `json:"fee_enabled_channels"`
	PacketFees                   []PacketFeeRecord         `json:"packet_fees"`
	RegisteredPayees             []PayeeRecord             `json:"registered_payees"`
	RegisteredCounterpartyPayees []CounterpartyPayeeRecord `json:"registered_counterparty_payees"`
	ForwardRelayers              []ForwardRelayerRecord    `json:"forward_relayers"`
}

type ChannelRecord struct {
	PortID          string     `json:"port_id"`
	ChannelID       string     `json:"channel_id"`
	State           string     `json:"state"`
	Ordering        string     `json:"ordering"`
	Counterparty    ChannelRef `json:"counterparty"`
	ConnectionHops  []string   `json:"connection_hops"`
	Version         string     `json:"version"`
	FeeVersion      string     `json:"fee_version,omitempty"`
	AppVersion      string     `json:"app_version,omitempty"`
	UpgradeSequence string     `json:"upgrade_sequence"`
}

type PacketStateRecord struct {
	Packet     PacketRef `json:"packet"`
	DataBytes  int       `json:"data_bytes"`
	DataSHA256 string    `json:"data_sha256"`
}

type PacketSequenceRecord struct {
	Channel  ChannelRef `json:"channel"`
	Sequence string     `json:"sequence"`
}

type ExportedPacketStateReport struct {
	Commitments      []PacketStateRecord    `json:"commitments"`
	Acknowledgements []PacketStateRecord    `json:"acknowledgements"`
	Receipts         []PacketStateRecord    `json:"receipts"`
	SendSequences    []PacketSequenceRecord `json:"send_sequences"`
	RecvSequences    []PacketSequenceRecord `json:"recv_sequences"`
	AckSequences     []PacketSequenceRecord `json:"ack_sequences"`
}

type StateCount struct {
	State string `json:"state"`
	Count int    `json:"count"`
}

type Summary struct {
	FeeEnabledChannels           int          `json:"fee_enabled_channels"`
	IdentifiedPacketFeeRecords   int          `json:"identified_packet_fee_records"`
	PacketFees                   int          `json:"packet_fees"`
	RegisteredPayees             int          `json:"registered_payees"`
	RegisteredCounterpartyPayees int          `json:"registered_counterparty_payees"`
	ForwardRelayers              int          `json:"forward_relayers"`
	FeeIBCNonzeroBalanceDenoms   int          `json:"feeibc_nonzero_balance_denoms"`
	Channels                     int          `json:"channels"`
	NextChannelSequence          string       `json:"next_channel_sequence"`
	ChannelStates                []StateCount `json:"channel_states"`
	FlushingChannels             int          `json:"flushing_channels"`
	OutstandingCommitments       int          `json:"outstanding_commitments"`
	ExportedAcknowledgements     int          `json:"exported_acknowledgements"`
	ExportedReceipts             int          `json:"exported_receipts"`
	Errors                       int          `json:"errors"`
	Warnings                     int          `json:"warnings"`
}

type Report struct {
	Schema              string                    `json:"schema"`
	Source              string                    `json:"source"`
	Complete            bool                      `json:"complete"`
	UpgradeReady        bool                      `json:"upgrade_ready"`
	Evidence            Evidence                  `json:"evidence"`
	Coverage            Coverage                  `json:"coverage"`
	Summary             Summary                   `json:"summary"`
	FeeIBC              FeeIBCReport              `json:"feeibc"`
	Channels            []ChannelRecord           `json:"channels"`
	ExportedPacketState ExportedPacketStateReport `json:"exported_packet_state"`
	Findings            []Finding                 `json:"findings"`
}

func printText(output io.Writer, report Report) error {
	var writeErr error
	writef := func(format string, args ...any) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(output, format, args...)
	}
	writef("Zerone IBC-Go v10 preflight census: %s\n", report.Source)
	writef("Complete: %t; upgrade ready: %t\n", report.Complete, report.UpgradeReady)
	writef(
		"Evidence: height=%s app_hash=%s input_sha256=%s\n",
		report.Evidence.ExportHeight,
		report.Evidence.AppHash,
		report.Evidence.InputSHA256,
	)
	writef(
		"ICS-29: %d fee-enabled channels, %d identified fee records (%d packet fees), %d payees, %d counterparty payees, %d forward relayers\n",
		report.Summary.FeeEnabledChannels,
		report.Summary.IdentifiedPacketFeeRecords,
		report.Summary.PacketFees,
		report.Summary.RegisteredPayees,
		report.Summary.RegisteredCounterpartyPayees,
		report.Summary.ForwardRelayers,
	)
	writef("feeibc module account: %s\n", printable(report.FeeIBC.ModuleAccountAddress))
	writef("feeibc balances: %s\n", formatCoins(report.FeeIBC.ModuleAccountBalances))
	writef("calculated escrow obligations: %s\n", formatCoins(report.FeeIBC.CalculatedEscrowObligations))
	writef("Channels: %d", report.Summary.Channels)
	for _, state := range report.Summary.ChannelStates {
		writef(" %s=%d", state.State, state.Count)
	}
	writef("\n")
	writef(
		"Exported packet state: commitments=%d acknowledgements=%d receipts=%d\n",
		report.Summary.OutstandingCommitments,
		report.Summary.ExportedAcknowledgements,
		report.Summary.ExportedReceipts,
	)
	for _, finding := range report.Findings {
		context := make([]string, 0, 2)
		if finding.Subject != "" {
			context = append(context, "subject="+finding.Subject)
		}
		if finding.Location != "" {
			context = append(context, "at="+finding.Location)
		}
		suffix := ""
		if len(context) > 0 {
			suffix = " (" + strings.Join(context, ", ") + ")"
		}
		writef("%s %s: %s%s\n", strings.ToUpper(finding.Severity), finding.Code, finding.Message, suffix)
	}
	writef("Coverage boundary (not proved by export):\n")
	for _, limitation := range report.Coverage.DoesNotProve {
		writef("- %s\n", limitation)
	}
	writef("Result: %d errors, %d warnings\n", report.Summary.Errors, report.Summary.Warnings)
	return writeErr
}

func printable(value string) string {
	if value == "" {
		return "<not found>"
	}
	return value
}

func formatCoins(coins []Coin) string {
	if len(coins) == 0 {
		return "none"
	}
	parts := make([]string, len(coins))
	for i, coin := range coins {
		parts[i] = coin.Amount + coin.Denom
	}
	return strings.Join(parts, ",")
}
