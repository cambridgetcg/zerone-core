package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

const (
	reportSchema         = "zerone.ibc-v10-census/v1"
	feeModuleAccountType = "/cosmos.auth.v1beta1.ModuleAccount"
	feeModuleName        = "feeibc"
	// Cosmos SDK v0.50's traditional module address is CometBFT
	// tmhash.SumTruncated("feeibc"), i.e. SHA256("feeibc")[:20], rendered with
	// Zerone's zrn HRP. This exact address was independently generated with
	// authtypes.NewModuleAddress("feeibc") under the Zerone encoding config.
	feeModuleAddress        = "zrn176rcyfn5k9d0wcxel3kmwvxh0hy3xcweksmdhj"
	feeModuleAddressHashHex = "f687822674b15af760d9fc6db730d77dc91361d9"
	ics29Version            = "ics29-1"
)

var (
	ibcIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9\._+\-#\[\]<>]+$`)
	channelIDPattern     = regexp.MustCompile(`^channel-[0-9]{1,20}$`)
	coinDenomPattern     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9/:._-]{2,127}$`)
)

type auditor struct {
	report                  Report
	identifiedFeeRecords    int
	relayerAllowlistRecords int
	feeAccountExported      bool
}

type parsedIBC struct {
	channels            []ChannelRecord
	nextChannelSequence string
	commitments         []PacketStateRecord
	acknowledgements    []PacketStateRecord
	receipts            []PacketStateRecord
	sendSequences       []PacketSequenceRecord
	recvSequences       []PacketSequenceRecord
	ackSequences        []PacketSequenceRecord
}

func newAuditor(source string, evidence Evidence, inputKind string) *auditor {
	return &auditor{
		report: Report{
			Schema:   reportSchema,
			Source:   source,
			Evidence: evidence,
			Coverage: Coverage{
				InputKind: inputKind,
				Proves: []string{
					"the exact exported IBC-Go v8 fee-enabled-channel, fee escrow, payee, channel, and packet-state records parsed by this report",
					"the bank balances at the deterministic feeibc Zerone module address, plus parity with a named auth ModuleAccount when one is exported",
					"the byte length and SHA-256 digest of exported commitment, acknowledgement, and receipt values",
				},
				DoesNotProve: []string{
					"that the operator-supplied export height and app hash match this file; preserve and verify the trusted-node command transcript separately",
					"live database keys, module-version metadata, mempool contents, or state changes committed after the export height",
					"the feeibc persistent locked key: IBC-Go v8 ExportGenesis omits it even though it records a severe-bug lock condition",
					"obsolete channel upgrade/counterparty-upgrade/error-receipt and pruningSequenceStart child keys omitted by v8 export; IBC-Go v10.7.0's bare-prefix Delete calls do not prove those children are removed",
					"recvStartSequence child keys omitted by v8 export; IBC-Go v10 still uses them for replay protection, so old-database rehearsal must prove they are preserved",
					"network or counterparty status: commitments are the clearest outstanding-send signal, retained acknowledgements are not necessarily pending, and unordered receipts are replay-prevention state",
					"bilateral in-flight clearance; that requires synchronized-height exports from both channel ends with zero commitments and an operational relay freeze",
					"whether packets, acknowledgements, or timeouts are currently being relayed outside the exported state",
					"successful old-database loading, v10 upgrade/pruning-key deletion, preserved packet-KV loading, restart, or packet lifecycle under the new binary; rehearse those against a copy of the validator database",
					"physical deletion of legacy IAVL data when a store key is removed from the commit root",
				},
			},
			FeeIBC: FeeIBCReport{
				ModuleAccountAddressSource:   "SHA256(\"feeibc\")[:20] fixed Zerone zrn Bech32 vector",
				ModuleAccountBalances:        []Coin{},
				CalculatedEscrowObligations:  []Coin{},
				FeeEnabledChannels:           []ChannelRef{},
				PacketFees:                   []PacketFeeRecord{},
				RegisteredPayees:             []PayeeRecord{},
				RegisteredCounterpartyPayees: []CounterpartyPayeeRecord{},
				ForwardRelayers:              []ForwardRelayerRecord{},
			},
			Channels: []ChannelRecord{},
			ExportedPacketState: ExportedPacketStateReport{
				Commitments:      []PacketStateRecord{},
				Acknowledgements: []PacketStateRecord{},
				Receipts:         []PacketStateRecord{},
				SendSequences:    []PacketSequenceRecord{},
				RecvSequences:    []PacketSequenceRecord{},
				AckSequences:     []PacketSequenceRecord{},
			},
			Findings: []Finding{},
		},
	}
}

func (a *auditor) add(severity, code, message, location, subject string) {
	a.report.Findings = append(a.report.Findings, Finding{
		Severity: severity,
		Code:     code,
		Message:  message,
		Location: location,
		Subject:  subject,
	})
}

// auditDocument accepts either the complete JSON document emitted by
// `zeroned export` or its complete app_state object. It deliberately refuses
// partial module objects because cross-module linkage is the point of this
// census.
func auditDocument(data []byte, source string, evidence Evidence) (Report, error) {
	if err := rejectDuplicateKeys(data); err != nil {
		return Report{}, err
	}
	inputDigest := sha256.Sum256(data)
	evidence.InputSHA256 = hex.EncodeToString(inputDigest[:])
	modules, inputKind, err := extractModules(data)
	if err != nil {
		return Report{}, err
	}
	a := newAuditor(source, evidence, inputKind)

	feeAccountAddress, feeAccountExported, err := parseFeeModuleAccount(modules["auth"])
	if err != nil {
		return Report{}, err
	}
	a.report.FeeIBC.ModuleAccountAddress = feeAccountAddress
	a.report.FeeIBC.AuthModuleAccountExported = feeAccountExported
	a.feeAccountExported = feeAccountExported
	if !feeAccountExported {
		a.add(
			severityWarning,
			"FEEIBC_AUTH_MODULE_ACCOUNT_NOT_EXPORTED",
			"auth.accounts has no feeibc ModuleAccount; the module is lazily created, so the census used its deterministic Zerone module address for the bank scan",
			"auth.accounts",
			feeAccountAddress,
		)
	}

	balances, err := parseFeeModuleBalance(modules["bank"], feeAccountAddress)
	if err != nil {
		return Report{}, err
	}
	a.report.FeeIBC.ModuleAccountBalances = balances

	if err := parseFeeIBC(modules["feeibc"], a); err != nil {
		return Report{}, err
	}
	ibcState, err := parseIBC(modules["ibc"])
	if err != nil {
		return Report{}, err
	}
	a.report.Channels = ibcState.channels
	a.report.Summary.NextChannelSequence = ibcState.nextChannelSequence
	a.report.ExportedPacketState = ExportedPacketStateReport{
		Commitments:      ibcState.commitments,
		Acknowledgements: ibcState.acknowledgements,
		Receipts:         ibcState.receipts,
		SendSequences:    ibcState.sendSequences,
		RecvSequences:    ibcState.recvSequences,
		AckSequences:     ibcState.ackSequences,
	}

	if err := a.auditCrossModuleState(); err != nil {
		return Report{}, err
	}
	a.finish()
	return a.report, nil
}

func extractModules(data []byte) (map[string]json.RawMessage, string, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, "", fmt.Errorf("decode input JSON: %w", err)
	}
	if root == nil {
		return nil, "", fmt.Errorf("schema ambiguity: input root must be a JSON object")
	}

	modules := root
	inputKind := "app_state"
	if appStateRaw, wrapped := root["app_state"]; wrapped {
		for _, module := range []string{"auth", "bank", "feeibc", "ibc"} {
			if _, ambiguous := root[module]; ambiguous {
				return nil, "", fmt.Errorf("schema ambiguity: both root.%s and root.app_state.%s could be audited", module, module)
			}
		}
		if err := json.Unmarshal(appStateRaw, &modules); err != nil || modules == nil {
			if err == nil {
				err = fmt.Errorf("value is not an object")
			}
			return nil, "", fmt.Errorf("decode app_state: %w", err)
		}
		inputKind = "zeroned_export"
	}

	for _, module := range []string{"auth", "bank", "feeibc", "ibc"} {
		raw, present := modules[module]
		if !present || len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, "", fmt.Errorf("schema ambiguity: complete app_state is missing the %s module", module)
		}
	}
	return modules, inputKind, nil
}

func parseFeeModuleAccount(raw json.RawMessage) (string, bool, error) {
	state, err := decodeObject(raw, "auth", []string{"params", "accounts"}, []string{"params", "accounts"})
	if err != nil {
		return "", false, err
	}
	accounts, err := decodeArray(state["accounts"], "auth.accounts")
	if err != nil {
		return "", false, err
	}

	var address string
	for i, accountRaw := range accounts {
		location := fmt.Sprintf("auth.accounts[%d]", i)
		var account map[string]json.RawMessage
		if err := json.Unmarshal(accountRaw, &account); err != nil || account == nil {
			if err == nil {
				err = fmt.Errorf("value is not an object")
			}
			return "", false, fmt.Errorf("schema ambiguity: decode %s: %w", location, err)
		}
		typeRaw, hasType := account["@type"]
		if !hasType {
			return "", false, fmt.Errorf("schema ambiguity: %s.@type is missing", location)
		}
		accountType, err := decodeString(typeRaw, location+".@type", false)
		if err != nil {
			return "", false, err
		}
		name := ""
		if nameRaw, hasName := account["name"]; hasName {
			name, err = decodeString(nameRaw, location+".name", false)
			if err != nil {
				return "", false, err
			}
		}
		if accountType != feeModuleAccountType {
			if name == feeModuleName {
				return "", false, fmt.Errorf(
					"schema ambiguity: %s names feeibc but has account type %q",
					location,
					accountType,
				)
			}
			continue
		}

		moduleAccount, err := decodeObject(
			accountRaw,
			location,
			[]string{"@type", "base_account", "name", "permissions"},
			[]string{"@type", "base_account", "name", "permissions"},
		)
		if err != nil {
			return "", false, err
		}
		moduleName, err := decodeString(moduleAccount["name"], location+".name", false)
		if err != nil {
			return "", false, err
		}
		permissions, err := decodeStringArray(moduleAccount["permissions"], location+".permissions")
		if err != nil {
			return "", false, err
		}
		baseAccount, err := decodeObject(
			moduleAccount["base_account"],
			location+".base_account",
			[]string{"address", "pub_key", "account_number", "sequence"},
			[]string{"address", "pub_key", "account_number", "sequence"},
		)
		if err != nil {
			return "", false, err
		}
		moduleAddress, err := decodeString(baseAccount["address"], location+".base_account.address", false)
		if err != nil {
			return "", false, err
		}
		if _, err := decodeUintString(baseAccount["account_number"], location+".base_account.account_number", false); err != nil {
			return "", false, err
		}
		if _, err := decodeUintString(baseAccount["sequence"], location+".base_account.sequence", false); err != nil {
			return "", false, err
		}
		if moduleName != feeModuleName {
			continue
		}
		if len(permissions) != 0 {
			return "", false, fmt.Errorf(
				"schema ambiguity: feeibc ModuleAccount must have no permissions, found %v",
				permissions,
			)
		}
		if !bytes.Equal(bytes.TrimSpace(baseAccount["pub_key"]), []byte("null")) {
			return "", false, fmt.Errorf("schema ambiguity: feeibc ModuleAccount base_account.pub_key must be null")
		}
		if address != "" {
			return "", false, fmt.Errorf(
				"schema ambiguity: multiple auth ModuleAccount records are named feeibc (%s and %s)",
				address,
				moduleAddress,
			)
		}
		if moduleAddress != feeModuleAddress {
			return "", false, fmt.Errorf(
				"schema ambiguity: feeibc ModuleAccount address %s does not equal deterministic address %s",
				moduleAddress,
				feeModuleAddress,
			)
		}
		address = moduleAddress
	}
	if address == "" {
		return feeModuleAddress, false, nil
	}
	return address, true, nil
}

func parseFeeModuleBalance(raw json.RawMessage, feeAddress string) ([]Coin, error) {
	state, err := decodeObject(
		raw,
		"bank",
		[]string{"params", "balances", "supply", "denom_metadata", "send_enabled"},
		[]string{"params", "balances", "supply", "denom_metadata", "send_enabled"},
	)
	if err != nil {
		return nil, err
	}
	if _, err := decodeObject(
		state["params"],
		"bank.params",
		[]string{"send_enabled", "default_send_enabled"},
		[]string{"send_enabled", "default_send_enabled"},
	); err != nil {
		return nil, err
	}
	for _, field := range []string{"supply", "denom_metadata", "send_enabled"} {
		if _, err := decodeArray(state[field], "bank."+field); err != nil {
			return nil, err
		}
	}
	balances, err := decodeArray(state["balances"], "bank.balances")
	if err != nil {
		return nil, err
	}
	seenAddresses := make(map[string]struct{}, len(balances))
	var feeBalance []Coin
	for i, balanceRaw := range balances {
		location := fmt.Sprintf("bank.balances[%d]", i)
		balance, err := decodeObject(
			balanceRaw,
			location,
			[]string{"address", "coins"},
			[]string{"address", "coins"},
		)
		if err != nil {
			return nil, err
		}
		address, err := decodeString(balance["address"], location+".address", false)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seenAddresses[address]; duplicate {
			return nil, fmt.Errorf("schema ambiguity: duplicate bank balance address %q", address)
		}
		seenAddresses[address] = struct{}{}
		coins, err := parseCoins(balance["coins"], location+".coins")
		if err != nil {
			return nil, err
		}
		if address == feeAddress {
			feeBalance = coins
		}
	}
	if feeBalance == nil {
		return []Coin{}, nil
	}
	return feeBalance, nil
}

func parseFeeIBC(raw json.RawMessage, a *auditor) error {
	state, err := decodeObject(
		raw,
		"feeibc",
		[]string{
			"identified_fees",
			"fee_enabled_channels",
			"registered_payees",
			"registered_counterparty_payees",
			"forward_relayers",
		},
		[]string{
			"identified_fees",
			"fee_enabled_channels",
			"registered_payees",
			"registered_counterparty_payees",
			"forward_relayers",
		},
	)
	if err != nil {
		return err
	}

	identified, err := decodeArray(state["identified_fees"], "feeibc.identified_fees")
	if err != nil {
		return err
	}
	a.identifiedFeeRecords = len(identified)
	identifiedPackets := make(map[string]struct{}, len(identified))
	for i, identifiedRaw := range identified {
		location := fmt.Sprintf("feeibc.identified_fees[%d]", i)
		entry, err := decodeObject(
			identifiedRaw,
			location,
			[]string{"packet_id", "packet_fees"},
			[]string{"packet_id", "packet_fees"},
		)
		if err != nil {
			return err
		}
		packet, err := parsePacketRef(entry["packet_id"], location+".packet_id")
		if err != nil {
			return err
		}
		if _, duplicate := identifiedPackets[packet.String()]; duplicate {
			return fmt.Errorf("schema ambiguity: duplicate identified fee packet %s", packet)
		}
		identifiedPackets[packet.String()] = struct{}{}
		packetFees, err := decodeArray(entry["packet_fees"], location+".packet_fees")
		if err != nil {
			return err
		}
		if len(packetFees) == 0 {
			a.add(
				severityError,
				"IDENTIFIED_PACKET_FEES_EMPTY",
				"identified packet fee record contains no PacketFee entries",
				location+".packet_fees",
				packet.String(),
			)
		}
		for j, packetFeeRaw := range packetFees {
			record, total, relayersPresent, err := parsePacketFee(
				packetFeeRaw,
				packet,
				fmt.Sprintf("%s.packet_fees[%d]", location, j),
			)
			if err != nil {
				return err
			}
			if len(total) == 0 {
				a.add(
					severityError,
					"PACKET_FEE_ZERO",
					"packet fee has no positive receive, acknowledgement, or timeout escrow obligation",
					fmt.Sprintf("%s.packet_fees[%d].fee", location, j),
					packet.String(),
				)
			}
			if relayersPresent {
				a.relayerAllowlistRecords++
			}
			a.report.FeeIBC.PacketFees = append(a.report.FeeIBC.PacketFees, record)
		}
	}

	enabled, err := decodeArray(state["fee_enabled_channels"], "feeibc.fee_enabled_channels")
	if err != nil {
		return err
	}
	enabledKeys := make(map[string]struct{}, len(enabled))
	for i, enabledRaw := range enabled {
		location := fmt.Sprintf("feeibc.fee_enabled_channels[%d]", i)
		channel, err := parseChannelRef(enabledRaw, location)
		if err != nil {
			return err
		}
		if _, duplicate := enabledKeys[channel.String()]; duplicate {
			return fmt.Errorf("schema ambiguity: duplicate fee-enabled channel %s", channel)
		}
		enabledKeys[channel.String()] = struct{}{}
		a.report.FeeIBC.FeeEnabledChannels = append(a.report.FeeIBC.FeeEnabledChannels, channel)
	}

	payees, err := decodeArray(state["registered_payees"], "feeibc.registered_payees")
	if err != nil {
		return err
	}
	payeeKeys := make(map[string]struct{}, len(payees))
	for i, payeeRaw := range payees {
		location := fmt.Sprintf("feeibc.registered_payees[%d]", i)
		object, err := decodeObject(
			payeeRaw,
			location,
			[]string{"channel_id", "relayer", "payee"},
			[]string{"channel_id", "relayer", "payee"},
		)
		if err != nil {
			return err
		}
		channelID, err := decodeString(object["channel_id"], location+".channel_id", false)
		if err != nil {
			return err
		}
		if err := validateIBCIdentifier(channelID, 8, 64, location+".channel_id"); err != nil {
			return err
		}
		relayer, err := decodeString(object["relayer"], location+".relayer", false)
		if err != nil {
			return err
		}
		payee, err := decodeString(object["payee"], location+".payee", false)
		if err != nil {
			return err
		}
		if err := validateLocalAddress(relayer, location+".relayer"); err != nil {
			return err
		}
		if err := validateLocalAddress(payee, location+".payee"); err != nil {
			return err
		}
		if relayer == payee {
			return fmt.Errorf("schema ambiguity: %s relayer and payee must differ", location)
		}
		key := channelID + "\x00" + relayer
		if _, duplicate := payeeKeys[key]; duplicate {
			return fmt.Errorf("schema ambiguity: duplicate registered payee key %s/%s", channelID, relayer)
		}
		payeeKeys[key] = struct{}{}
		a.report.FeeIBC.RegisteredPayees = append(a.report.FeeIBC.RegisteredPayees, PayeeRecord{
			ChannelID: channelID,
			Relayer:   relayer,
			Payee:     payee,
		})
	}

	counterpartyPayees, err := decodeArray(
		state["registered_counterparty_payees"],
		"feeibc.registered_counterparty_payees",
	)
	if err != nil {
		return err
	}
	counterpartyKeys := make(map[string]struct{}, len(counterpartyPayees))
	for i, payeeRaw := range counterpartyPayees {
		location := fmt.Sprintf("feeibc.registered_counterparty_payees[%d]", i)
		object, err := decodeObject(
			payeeRaw,
			location,
			[]string{"channel_id", "relayer", "counterparty_payee"},
			[]string{"channel_id", "relayer", "counterparty_payee"},
		)
		if err != nil {
			return err
		}
		channelID, err := decodeString(object["channel_id"], location+".channel_id", false)
		if err != nil {
			return err
		}
		if err := validateIBCIdentifier(channelID, 8, 64, location+".channel_id"); err != nil {
			return err
		}
		relayer, err := decodeString(object["relayer"], location+".relayer", false)
		if err != nil {
			return err
		}
		payee, err := decodeString(object["counterparty_payee"], location+".counterparty_payee", false)
		if err != nil {
			return err
		}
		if strings.TrimSpace(payee) == "" {
			return fmt.Errorf("schema ambiguity: %s.counterparty_payee must not be blank", location)
		}
		if err := validateLocalAddress(relayer, location+".relayer"); err != nil {
			return err
		}
		key := channelID + "\x00" + relayer
		if _, duplicate := counterpartyKeys[key]; duplicate {
			return fmt.Errorf("schema ambiguity: duplicate registered counterparty payee key %s/%s", channelID, relayer)
		}
		counterpartyKeys[key] = struct{}{}
		a.report.FeeIBC.RegisteredCounterpartyPayees = append(
			a.report.FeeIBC.RegisteredCounterpartyPayees,
			CounterpartyPayeeRecord{
				ChannelID:         channelID,
				Relayer:           relayer,
				CounterpartyPayee: payee,
			},
		)
	}

	forwardRelayers, err := decodeArray(state["forward_relayers"], "feeibc.forward_relayers")
	if err != nil {
		return err
	}
	forwardKeys := make(map[string]struct{}, len(forwardRelayers))
	for i, forwardRaw := range forwardRelayers {
		location := fmt.Sprintf("feeibc.forward_relayers[%d]", i)
		object, err := decodeObject(
			forwardRaw,
			location,
			[]string{"address", "packet_id"},
			[]string{"address", "packet_id"},
		)
		if err != nil {
			return err
		}
		address, err := decodeString(object["address"], location+".address", false)
		if err != nil {
			return err
		}
		if err := validateLocalAddress(address, location+".address"); err != nil {
			return err
		}
		packet, err := parsePacketRef(object["packet_id"], location+".packet_id")
		if err != nil {
			return err
		}
		if _, duplicate := forwardKeys[packet.String()]; duplicate {
			return fmt.Errorf("schema ambiguity: duplicate forward relayer packet %s", packet)
		}
		forwardKeys[packet.String()] = struct{}{}
		a.report.FeeIBC.ForwardRelayers = append(a.report.FeeIBC.ForwardRelayers, ForwardRelayerRecord{
			Address: address,
			Packet:  packet,
		})
	}

	return nil
}

func parsePacketFee(raw json.RawMessage, packet PacketRef, path string) (PacketFeeRecord, []Coin, bool, error) {
	object, err := decodeObject(
		raw,
		path,
		[]string{"fee", "refund_address", "relayers"},
		[]string{"fee", "refund_address", "relayers"},
	)
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	refundAddress, err := decodeString(object["refund_address"], path+".refund_address", false)
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	if err := validateLocalAddress(refundAddress, path+".refund_address"); err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	relayers, err := decodeStringArray(object["relayers"], path+".relayers")
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	for i, relayer := range relayers {
		if err := validateLocalAddress(relayer, fmt.Sprintf("%s.relayers[%d]", path, i)); err != nil {
			return PacketFeeRecord{}, nil, false, err
		}
	}
	sort.Strings(relayers)
	fee, err := decodeObject(
		object["fee"],
		path+".fee",
		[]string{"recv_fee", "ack_fee", "timeout_fee"},
		[]string{"recv_fee", "ack_fee", "timeout_fee"},
	)
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	recvFee, err := parseCoins(fee["recv_fee"], path+".fee.recv_fee")
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	ackFee, err := parseCoins(fee["ack_fee"], path+".fee.ack_fee")
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	timeoutFee, err := parseCoins(fee["timeout_fee"], path+".fee.timeout_fee")
	if err != nil {
		return PacketFeeRecord{}, nil, false, err
	}
	recvAndAck, err := addCoinSets(recvFee, ackFee)
	if err != nil {
		return PacketFeeRecord{}, nil, false, fmt.Errorf("schema ambiguity: %s fee total: %w", path, err)
	}
	total := maximumCoins(recvAndAck, timeoutFee)
	return PacketFeeRecord{
		Packet:        packet,
		RefundAddress: refundAddress,
		Relayers:      relayers,
		RecvFee:       recvFee,
		AckFee:        ackFee,
		TimeoutFee:    timeoutFee,
		EscrowAmount:  total,
	}, total, len(relayers) > 0, nil
}

func parseIBC(raw json.RawMessage) (parsedIBC, error) {
	state, err := decodeObject(
		raw,
		"ibc",
		[]string{"client_genesis", "connection_genesis", "channel_genesis"},
		[]string{"client_genesis", "connection_genesis", "channel_genesis"},
	)
	if err != nil {
		return parsedIBC{}, err
	}
	channelGenesis, err := decodeObject(
		state["channel_genesis"],
		"ibc.channel_genesis",
		[]string{
			"channels",
			"acknowledgements",
			"commitments",
			"receipts",
			"send_sequences",
			"recv_sequences",
			"ack_sequences",
			"next_channel_sequence",
			"params",
		},
		[]string{
			"channels",
			"acknowledgements",
			"commitments",
			"receipts",
			"send_sequences",
			"recv_sequences",
			"ack_sequences",
			"next_channel_sequence",
			"params",
		},
	)
	if err != nil {
		return parsedIBC{}, err
	}
	nextChannelSequence, err := decodeUintString(
		channelGenesis["next_channel_sequence"],
		"ibc.channel_genesis.next_channel_sequence",
		false,
	)
	if err != nil {
		return parsedIBC{}, err
	}
	if err := validateChannelParams(channelGenesis["params"]); err != nil {
		return parsedIBC{}, err
	}

	result := parsedIBC{
		channels:            []ChannelRecord{},
		nextChannelSequence: nextChannelSequence,
	}
	channels, err := decodeArray(channelGenesis["channels"], "ibc.channel_genesis.channels")
	if err != nil {
		return parsedIBC{}, err
	}
	channelKeys := make(map[string]struct{}, len(channels))
	var maxChannelSequence uint64
	for i, channelRaw := range channels {
		location := fmt.Sprintf("ibc.channel_genesis.channels[%d]", i)
		channel, err := parseChannel(channelRaw, location)
		if err != nil {
			return parsedIBC{}, err
		}
		key := ChannelRef{PortID: channel.PortID, ChannelID: channel.ChannelID}.String()
		if _, duplicate := channelKeys[key]; duplicate {
			return parsedIBC{}, fmt.Errorf("schema ambiguity: duplicate identified channel %s", key)
		}
		channelKeys[key] = struct{}{}
		channelSequence, err := parseSDKChannelSequence(channel.ChannelID)
		if err != nil {
			return parsedIBC{}, fmt.Errorf("schema ambiguity: %s.channel_id: %w", location, err)
		}
		if channelSequence > maxChannelSequence {
			maxChannelSequence = channelSequence
		}
		result.channels = append(result.channels, channel)
	}
	nextSequence, _ := strconv.ParseUint(nextChannelSequence, 10, 64)
	if len(channels) > 0 && maxChannelSequence >= nextSequence {
		return parsedIBC{}, fmt.Errorf(
			"schema ambiguity: ibc.channel_genesis.next_channel_sequence %d is not greater than maximum exported channel sequence %d",
			nextSequence,
			maxChannelSequence,
		)
	}

	result.acknowledgements, err = parsePacketStates(
		channelGenesis["acknowledgements"],
		"ibc.channel_genesis.acknowledgements",
		false,
	)
	if err != nil {
		return parsedIBC{}, err
	}
	result.commitments, err = parsePacketStates(
		channelGenesis["commitments"],
		"ibc.channel_genesis.commitments",
		false,
	)
	if err != nil {
		return parsedIBC{}, err
	}
	result.receipts, err = parsePacketStates(
		channelGenesis["receipts"],
		"ibc.channel_genesis.receipts",
		true,
	)
	if err != nil {
		return parsedIBC{}, err
	}
	result.sendSequences, err = parsePacketSequences(
		channelGenesis["send_sequences"],
		"ibc.channel_genesis.send_sequences",
	)
	if err != nil {
		return parsedIBC{}, err
	}
	result.recvSequences, err = parsePacketSequences(
		channelGenesis["recv_sequences"],
		"ibc.channel_genesis.recv_sequences",
	)
	if err != nil {
		return parsedIBC{}, err
	}
	result.ackSequences, err = parsePacketSequences(
		channelGenesis["ack_sequences"],
		"ibc.channel_genesis.ack_sequences",
	)
	if err != nil {
		return parsedIBC{}, err
	}
	return result, nil
}

func validateChannelParams(raw json.RawMessage) error {
	params, err := decodeObject(
		raw,
		"ibc.channel_genesis.params",
		[]string{"upgrade_timeout"},
		[]string{"upgrade_timeout"},
	)
	if err != nil {
		return err
	}
	timeout, err := decodeObject(
		params["upgrade_timeout"],
		"ibc.channel_genesis.params.upgrade_timeout",
		[]string{"height", "timestamp"},
		[]string{"height", "timestamp"},
	)
	if err != nil {
		return err
	}
	height, err := decodeObject(
		timeout["height"],
		"ibc.channel_genesis.params.upgrade_timeout.height",
		[]string{"revision_number", "revision_height"},
		[]string{"revision_number", "revision_height"},
	)
	if err != nil {
		return err
	}
	for _, field := range []string{"revision_number", "revision_height"} {
		value, err := decodeUintString(
			height[field],
			"ibc.channel_genesis.params.upgrade_timeout.height."+field,
			false,
		)
		if err != nil {
			return err
		}
		if value != "0" {
			return fmt.Errorf(
				"schema ambiguity: ibc.channel_genesis.params.upgrade_timeout.height must be zero under IBC-Go v8",
			)
		}
	}
	timestamp, err := decodeUintString(
		timeout["timestamp"],
		"ibc.channel_genesis.params.upgrade_timeout.timestamp",
		false,
	)
	if err != nil {
		return err
	}
	if timestamp == "0" {
		return fmt.Errorf(
			"schema ambiguity: ibc.channel_genesis.params.upgrade_timeout.timestamp must be nonzero under IBC-Go v8",
		)
	}
	return nil
}

func parseChannel(raw json.RawMessage, path string) (ChannelRecord, error) {
	object, err := decodeObject(
		raw,
		path,
		[]string{
			"state",
			"ordering",
			"counterparty",
			"connection_hops",
			"version",
			"port_id",
			"channel_id",
			"upgrade_sequence",
		},
		[]string{
			"state",
			"ordering",
			"counterparty",
			"connection_hops",
			"version",
			"port_id",
			"channel_id",
			"upgrade_sequence",
		},
	)
	if err != nil {
		return ChannelRecord{}, err
	}
	state, err := decodeString(object["state"], path+".state", false)
	if err != nil {
		return ChannelRecord{}, err
	}
	switch state {
	case "STATE_UNINITIALIZED_UNSPECIFIED", "STATE_INIT", "STATE_TRYOPEN", "STATE_OPEN",
		"STATE_CLOSED", "STATE_FLUSHING", "STATE_FLUSHCOMPLETE":
	default:
		return ChannelRecord{}, fmt.Errorf("schema ambiguity: %s.state has unknown IBC-Go v8 value %q", path, state)
	}
	ordering, err := decodeString(object["ordering"], path+".ordering", false)
	if err != nil {
		return ChannelRecord{}, err
	}
	switch ordering {
	case "ORDER_NONE_UNSPECIFIED", "ORDER_UNORDERED", "ORDER_ORDERED":
	default:
		return ChannelRecord{}, fmt.Errorf("schema ambiguity: %s.ordering has unknown IBC-Go v8 value %q", path, ordering)
	}
	counterparty, err := parseCounterpartyRef(object["counterparty"], path+".counterparty")
	if err != nil {
		return ChannelRecord{}, err
	}
	connectionHops, err := decodeStringArray(object["connection_hops"], path+".connection_hops")
	if err != nil {
		return ChannelRecord{}, err
	}
	version, err := decodeString(object["version"], path+".version", true)
	if err != nil {
		return ChannelRecord{}, err
	}
	portID, err := decodeString(object["port_id"], path+".port_id", false)
	if err != nil {
		return ChannelRecord{}, err
	}
	channelID, err := decodeString(object["channel_id"], path+".channel_id", false)
	if err != nil {
		return ChannelRecord{}, err
	}
	if err := validateIBCIdentifier(portID, 2, 128, path+".port_id"); err != nil {
		return ChannelRecord{}, err
	}
	if err := validateIBCIdentifier(channelID, 8, 64, path+".channel_id"); err != nil {
		return ChannelRecord{}, err
	}
	if state == "STATE_UNINITIALIZED_UNSPECIFIED" {
		return ChannelRecord{}, fmt.Errorf("schema ambiguity: %s.state would fail IBC-Go v10 Channel.ValidateBasic", path)
	}
	if ordering == "ORDER_NONE_UNSPECIFIED" {
		return ChannelRecord{}, fmt.Errorf("schema ambiguity: %s.ordering would fail IBC-Go v10 Channel.ValidateBasic", path)
	}
	if len(connectionHops) != 1 {
		return ChannelRecord{}, fmt.Errorf(
			"schema ambiguity: %s.connection_hops has %d entries; IBC-Go v10 requires exactly one",
			path,
			len(connectionHops),
		)
	}
	if err := validateIBCIdentifier(connectionHops[0], 10, 64, path+".connection_hops[0]"); err != nil {
		return ChannelRecord{}, err
	}
	upgradeSequence, err := decodeUintString(object["upgrade_sequence"], path+".upgrade_sequence", false)
	if err != nil {
		return ChannelRecord{}, err
	}
	feeVersion, appVersion, err := parseFeeMetadata(version, path+".version")
	if err != nil {
		return ChannelRecord{}, err
	}
	return ChannelRecord{
		PortID:          portID,
		ChannelID:       channelID,
		State:           state,
		Ordering:        ordering,
		Counterparty:    counterparty,
		ConnectionHops:  connectionHops,
		Version:         version,
		FeeVersion:      feeVersion,
		AppVersion:      appVersion,
		UpgradeSequence: upgradeSequence,
	}, nil
}

func parseFeeMetadata(version, path string) (string, string, error) {
	trimmed := strings.TrimSpace(version)
	if !strings.HasPrefix(trimmed, "{") || !json.Valid([]byte(trimmed)) {
		return "", "", nil
	}
	var candidate map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &candidate); err != nil {
		return "", "", nil
	}
	if _, hasFeeVersion := candidate["fee_version"]; !hasFeeVersion {
		return "", "", nil
	}
	if err := rejectDuplicateKeys([]byte(trimmed)); err != nil {
		return "", "", fmt.Errorf("schema ambiguity in embedded ICS-29 metadata at %s: %w", path, err)
	}
	object, err := decodeObject(
		json.RawMessage(trimmed),
		path+"<ics29-metadata>",
		[]string{"fee_version", "app_version"},
		[]string{"fee_version", "app_version"},
	)
	if err != nil {
		return "", "", err
	}
	feeVersion, err := decodeString(object["fee_version"], path+".fee_version", false)
	if err != nil {
		return "", "", err
	}
	appVersion, err := decodeString(object["app_version"], path+".app_version", false)
	if err != nil {
		return "", "", err
	}
	return feeVersion, appVersion, nil
}

func parsePacketStates(raw json.RawMessage, path string, allowEmptyData bool) ([]PacketStateRecord, error) {
	entries, err := decodeArray(raw, path)
	if err != nil {
		return nil, err
	}
	result := make([]PacketStateRecord, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for i, entryRaw := range entries {
		location := fmt.Sprintf("%s[%d]", path, i)
		object, err := decodeObject(
			entryRaw,
			location,
			[]string{"port_id", "channel_id", "sequence", "data"},
			[]string{"port_id", "channel_id", "sequence", "data"},
		)
		if err != nil {
			return nil, err
		}
		packet, err := parsePacketRef(objectFromPacketFields(object), location)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[packet.String()]; duplicate {
			return nil, fmt.Errorf("schema ambiguity: duplicate packet state %s at %s", packet, path)
		}
		seen[packet.String()] = struct{}{}
		data, err := decodeBase64(object["data"], location+".data")
		if err != nil {
			return nil, err
		}
		if len(data) == 0 && !allowEmptyData {
			return nil, fmt.Errorf("schema ambiguity: exported packet state %s has empty data at %s", packet, location)
		}
		digest := sha256.Sum256(data)
		result = append(result, PacketStateRecord{
			Packet:     packet,
			DataBytes:  len(data),
			DataSHA256: hex.EncodeToString(digest[:]),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return packetRefLess(result[i].Packet, result[j].Packet)
	})
	return result, nil
}

func objectFromPacketFields(object map[string]json.RawMessage) json.RawMessage {
	encoded, err := json.Marshal(map[string]json.RawMessage{
		"port_id":    object["port_id"],
		"channel_id": object["channel_id"],
		"sequence":   object["sequence"],
	})
	if err != nil {
		panic(err)
	}
	return encoded
}

func parsePacketSequences(raw json.RawMessage, path string) ([]PacketSequenceRecord, error) {
	entries, err := decodeArray(raw, path)
	if err != nil {
		return nil, err
	}
	result := make([]PacketSequenceRecord, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for i, entryRaw := range entries {
		location := fmt.Sprintf("%s[%d]", path, i)
		object, err := decodeObject(
			entryRaw,
			location,
			[]string{"port_id", "channel_id", "sequence"},
			[]string{"port_id", "channel_id", "sequence"},
		)
		if err != nil {
			return nil, err
		}
		channel := ChannelRef{}
		channel.PortID, err = decodeString(object["port_id"], location+".port_id", false)
		if err != nil {
			return nil, err
		}
		channel.ChannelID, err = decodeString(object["channel_id"], location+".channel_id", false)
		if err != nil {
			return nil, err
		}
		if err := validateIBCIdentifier(channel.PortID, 2, 128, location+".port_id"); err != nil {
			return nil, err
		}
		if err := validateIBCIdentifier(channel.ChannelID, 8, 64, location+".channel_id"); err != nil {
			return nil, err
		}
		sequence, err := decodeUintString(object["sequence"], location+".sequence", true)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[channel.String()]; duplicate {
			return nil, fmt.Errorf("schema ambiguity: duplicate packet sequence channel %s at %s", channel, path)
		}
		seen[channel.String()] = struct{}{}
		result = append(result, PacketSequenceRecord{Channel: channel, Sequence: sequence})
	}
	sort.Slice(result, func(i, j int) bool {
		return channelRefLess(result[i].Channel, result[j].Channel)
	})
	return result, nil
}

func parseChannelRef(raw json.RawMessage, path string) (ChannelRef, error) {
	object, err := decodeObject(
		raw,
		path,
		[]string{"port_id", "channel_id"},
		[]string{"port_id", "channel_id"},
	)
	if err != nil {
		return ChannelRef{}, err
	}
	portID, err := decodeString(object["port_id"], path+".port_id", false)
	if err != nil {
		return ChannelRef{}, err
	}
	channelID, err := decodeString(object["channel_id"], path+".channel_id", false)
	if err != nil {
		return ChannelRef{}, err
	}
	if err := validateIBCIdentifier(portID, 2, 128, path+".port_id"); err != nil {
		return ChannelRef{}, err
	}
	if err := validateIBCIdentifier(channelID, 8, 64, path+".channel_id"); err != nil {
		return ChannelRef{}, err
	}
	return ChannelRef{PortID: portID, ChannelID: channelID}, nil
}

func parseCounterpartyRef(raw json.RawMessage, path string) (ChannelRef, error) {
	object, err := decodeObject(
		raw,
		path,
		[]string{"port_id", "channel_id"},
		[]string{"port_id", "channel_id"},
	)
	if err != nil {
		return ChannelRef{}, err
	}
	portID, err := decodeString(object["port_id"], path+".port_id", false)
	if err != nil {
		return ChannelRef{}, err
	}
	channelID, err := decodeString(object["channel_id"], path+".channel_id", true)
	if err != nil {
		return ChannelRef{}, err
	}
	if err := validateIBCIdentifier(portID, 2, 128, path+".port_id"); err != nil {
		return ChannelRef{}, err
	}
	if channelID != "" {
		if err := validateIBCIdentifier(channelID, 8, 64, path+".channel_id"); err != nil {
			return ChannelRef{}, err
		}
	}
	return ChannelRef{PortID: portID, ChannelID: channelID}, nil
}

func parsePacketRef(raw json.RawMessage, path string) (PacketRef, error) {
	object, err := decodeObject(
		raw,
		path,
		[]string{"port_id", "channel_id", "sequence"},
		[]string{"port_id", "channel_id", "sequence"},
	)
	if err != nil {
		return PacketRef{}, err
	}
	portID, err := decodeString(object["port_id"], path+".port_id", false)
	if err != nil {
		return PacketRef{}, err
	}
	channelID, err := decodeString(object["channel_id"], path+".channel_id", false)
	if err != nil {
		return PacketRef{}, err
	}
	sequence, err := decodeUintString(object["sequence"], path+".sequence", true)
	if err != nil {
		return PacketRef{}, err
	}
	if err := validateIBCIdentifier(portID, 2, 128, path+".port_id"); err != nil {
		return PacketRef{}, err
	}
	if err := validateIBCIdentifier(channelID, 8, 64, path+".channel_id"); err != nil {
		return PacketRef{}, err
	}
	return PacketRef{PortID: portID, ChannelID: channelID, Sequence: sequence}, nil
}

func validateIBCIdentifier(value string, minimum, maximum int, path string) error {
	if strings.TrimSpace(value) == "" ||
		strings.Contains(value, "/") ||
		len(value) < minimum ||
		len(value) > maximum ||
		!ibcIdentifierPattern.MatchString(value) {
		return fmt.Errorf(
			"schema ambiguity: %s is not an ICS-24 identifier of %d-%d characters",
			path,
			minimum,
			maximum,
		)
	}
	return nil
}

func parseSDKChannelSequence(channelID string) (uint64, error) {
	if !channelIDPattern.MatchString(channelID) {
		return 0, fmt.Errorf("channel identifier is not in IBC-Go v8 SDK format channel-{N}")
	}
	sequence, err := strconv.ParseUint(strings.TrimPrefix(channelID, "channel-"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse channel sequence: %w", err)
	}
	if channelID != "channel-"+strconv.FormatUint(sequence, 10) {
		return 0, fmt.Errorf("channel identifier is not canonical channel-{N}")
	}
	return sequence, nil
}

func validateLocalAddress(value, path string) error {
	hrp, payload, err := bech32.DecodeAndConvert(value)
	if err != nil || hrp != "zrn" || len(payload) != 20 {
		return fmt.Errorf("schema ambiguity: %s must be a Zerone zrn Bech32 account address", path)
	}
	canonical, err := bech32.ConvertAndEncode(hrp, payload)
	if err != nil || canonical != value {
		return fmt.Errorf("schema ambiguity: %s must be canonical lowercase Zerone Bech32", path)
	}
	return nil
}

func (a *auditor) auditCrossModuleState() error {
	a.add(
		severityError,
		"OLD_DATABASE_REHEARSAL_REQUIRED",
		"v8 export omits fee lock and channel auxiliary keys; old-DB rehearsal must remove obsolete upgrade/pruning children while preserving recvStartSequence replay protection",
		"",
		"",
	)
	hasExportedFeeState := a.identifiedFeeRecords > 0 ||
		len(a.report.FeeIBC.FeeEnabledChannels) > 0 ||
		len(a.report.FeeIBC.RegisteredPayees) > 0 ||
		len(a.report.FeeIBC.RegisteredCounterpartyPayees) > 0 ||
		len(a.report.FeeIBC.ForwardRelayers) > 0
	if !a.feeAccountExported && (hasExportedFeeState || len(a.report.FeeIBC.ModuleAccountBalances) > 0) {
		a.add(
			severityError,
			"FEEIBC_AUTH_MODULE_ACCOUNT_REQUIRED_BY_STATE",
			"fee records or a nonzero deterministic-address balance exist without an exported feeibc auth ModuleAccount",
			"auth.accounts",
			a.report.FeeIBC.ModuleAccountAddress,
		)
	}
	if len(a.report.FeeIBC.FeeEnabledChannels) > 0 {
		a.add(
			severityError,
			"FEE_ENABLED_CHANNELS_PRESENT",
			fmt.Sprintf("%d ICS-29 fee-enabled channel flags must be reconciled before the fee store is removed", len(a.report.FeeIBC.FeeEnabledChannels)),
			"feeibc.fee_enabled_channels",
			"",
		)
	}
	if a.identifiedFeeRecords > 0 {
		a.add(
			severityError,
			"IDENTIFIED_PACKET_FEES_PRESENT",
			fmt.Sprintf("%d identified packet fee records represent exported escrow obligations", a.identifiedFeeRecords),
			"feeibc.identified_fees",
			"",
		)
	}
	if len(a.report.FeeIBC.RegisteredPayees) > 0 {
		a.add(
			severityWarning,
			"REGISTERED_PAYEES_PRESENT",
			"registered local payee mappings will not exist after ICS-29 removal; archive or explicitly discard them",
			"feeibc.registered_payees",
			"",
		)
	}
	if len(a.report.FeeIBC.RegisteredCounterpartyPayees) > 0 {
		a.add(
			severityWarning,
			"REGISTERED_COUNTERPARTY_PAYEES_PRESENT",
			"registered counterparty payee mappings will not exist after ICS-29 removal; archive or explicitly discard them",
			"feeibc.registered_counterparty_payees",
			"",
		)
	}
	if len(a.report.FeeIBC.ForwardRelayers) > 0 {
		a.add(
			severityError,
			"FORWARD_RELAYER_STATE_PRESENT",
			"forward relayer records are evidence of unresolved asynchronous acknowledgement fee state",
			"feeibc.forward_relayers",
			"",
		)
	}
	if a.relayerAllowlistRecords > 0 {
		a.add(
			severityError,
			"PACKET_FEE_RELAYER_ALLOWLIST_PRESENT",
			fmt.Sprintf("%d packet fees contain relayer allowlists that IBC-Go v8 validation rejects", a.relayerAllowlistRecords),
			"feeibc.identified_fees",
			"",
		)
	}

	obligationAmounts := make(map[string]*big.Int)
	for _, packetFee := range a.report.FeeIBC.PacketFees {
		if err := accumulateCoins(obligationAmounts, packetFee.EscrowAmount); err != nil {
			return fmt.Errorf("schema ambiguity: aggregate feeibc escrow obligations: %w", err)
		}
	}
	obligations := coinMapToSlice(obligationAmounts)
	a.report.FeeIBC.CalculatedEscrowObligations = obligations
	if len(a.report.FeeIBC.ModuleAccountBalances) > 0 {
		a.add(
			severityError,
			"FEEIBC_MODULE_BALANCE_NONZERO",
			"the feeibc ModuleAccount still holds bank balances that require an explicit refund or disposition plan",
			"bank.balances",
			a.report.FeeIBC.ModuleAccountAddress,
		)
	}
	if !equalCoins(a.report.FeeIBC.ModuleAccountBalances, obligations) {
		a.add(
			severityError,
			"FEEIBC_ESCROW_BALANCE_MISMATCH",
			fmt.Sprintf(
				"feeibc bank balance (%s) does not equal calculated PacketFee escrow obligations (%s)",
				formatCoins(a.report.FeeIBC.ModuleAccountBalances),
				formatCoins(obligations),
			),
			"bank.balances",
			a.report.FeeIBC.ModuleAccountAddress,
		)
	}

	channelByKey := make(map[string]ChannelRecord, len(a.report.Channels))
	channelIDs := make(map[string]struct{}, len(a.report.Channels))
	metadataChannels := make(map[string]ChannelRecord)
	flushing := 0
	incompleteHandshakes := 0
	for _, channel := range a.report.Channels {
		ref := ChannelRef{PortID: channel.PortID, ChannelID: channel.ChannelID}
		channelByKey[ref.String()] = channel
		channelIDs[channel.ChannelID] = struct{}{}
		if channel.FeeVersion != "" {
			metadataChannels[ref.String()] = channel
		}
		if channel.UpgradeSequence != "0" {
			a.add(
				severityError,
				"CHANNEL_UPGRADE_SEQUENCE_AMBIGUOUS",
				"nonzero upgrade_sequence cannot distinguish historical upgrades from a current OPEN-state upgrade because v8 export omits upgrade records",
				"ibc.channel_genesis.channels",
				ref.String(),
			)
		}
		switch channel.State {
		case "STATE_FLUSHING", "STATE_FLUSHCOMPLETE":
			flushing++
		case "STATE_INIT", "STATE_TRYOPEN":
			incompleteHandshakes++
		}
	}
	if flushing > 0 {
		a.add(
			severityError,
			"CHANNEL_UPGRADE_STATE_BLOCKS_V10_MIGRATION",
			fmt.Sprintf("%d channels are FLUSHING or FLUSHCOMPLETE; the IBC-Go v10 channel migration refuses these states", flushing),
			"ibc.channel_genesis.channels",
			"",
		)
	}
	if incompleteHandshakes > 0 {
		a.add(
			severityWarning,
			"INCOMPLETE_CHANNEL_HANDSHAKES_PRESENT",
			fmt.Sprintf("%d channels are in INIT or TRYOPEN; freeze and rehearse handshake behavior across the upgrade boundary", incompleteHandshakes),
			"ibc.channel_genesis.channels",
			"",
		)
	}
	enabledKeys := make(map[string]struct{}, len(a.report.FeeIBC.FeeEnabledChannels))
	for _, enabled := range a.report.FeeIBC.FeeEnabledChannels {
		key := enabled.String()
		enabledKeys[key] = struct{}{}
		channel, found := channelByKey[key]
		if !found {
			a.add(
				severityError,
				"FEE_ENABLED_CHANNEL_MISSING",
				"fee-enabled channel flag has no matching exported IBC channel",
				"feeibc.fee_enabled_channels",
				key,
			)
			continue
		}
		if channel.FeeVersion == "" {
			a.add(
				severityError,
				"FEE_ENABLED_CHANNEL_VERSION_AMBIGUOUS",
				"fee-enabled channel version is not recognizable ICS-29 metadata and cannot be safely unwrapped",
				"ibc.channel_genesis.channels",
				key,
			)
		} else if channel.FeeVersion != ics29Version {
			a.add(
				severityError,
				"FEE_ENABLED_CHANNEL_VERSION_UNSUPPORTED",
				fmt.Sprintf("fee-enabled channel advertises %q, expected %q", channel.FeeVersion, ics29Version),
				"ibc.channel_genesis.channels",
				key,
			)
		}
	}
	if len(metadataChannels) > 0 {
		a.add(
			severityError,
			"ICS29_CHANNEL_VERSIONS_PRESENT",
			fmt.Sprintf("%d channel versions still wrap their application version in ICS-29 metadata", len(metadataChannels)),
			"ibc.channel_genesis.channels",
			"",
		)
	}
	for key := range metadataChannels {
		if _, enabled := enabledKeys[key]; !enabled {
			a.add(
				severityError,
				"ICS29_VERSION_WITHOUT_FEE_FLAG",
				"channel has ICS-29 version metadata but no exported fee-enabled flag",
				"ibc.channel_genesis.channels",
				key,
			)
		}
	}

	if len(a.report.ExportedPacketState.Commitments) > 0 {
		a.add(
			severityError,
			"OUTSTANDING_PACKET_COMMITMENTS_PRESENT",
			fmt.Sprintf("%d source-chain packet commitments remain; drain packet lifecycles or explicitly rehearse their acknowledgement/timeout paths across restart", len(a.report.ExportedPacketState.Commitments)),
			"ibc.channel_genesis.commitments",
			"",
		)
	}

	for _, recordSet := range []struct {
		records  []PacketStateRecord
		location string
	}{
		{a.report.ExportedPacketState.Commitments, "ibc.channel_genesis.commitments"},
		{a.report.ExportedPacketState.Acknowledgements, "ibc.channel_genesis.acknowledgements"},
		{a.report.ExportedPacketState.Receipts, "ibc.channel_genesis.receipts"},
	} {
		for _, record := range recordSet.records {
			ref := ChannelRef{PortID: record.Packet.PortID, ChannelID: record.Packet.ChannelID}
			if _, exists := channelByKey[ref.String()]; !exists {
				a.add(
					severityError,
					"PACKET_STATE_CHANNEL_MISSING",
					"exported packet state refers to a channel absent from ibc.channel_genesis.channels",
					recordSet.location,
					record.Packet.String(),
				)
			}
		}
	}
	for _, sequenceSet := range []struct {
		records  []PacketSequenceRecord
		location string
	}{
		{a.report.ExportedPacketState.SendSequences, "ibc.channel_genesis.send_sequences"},
		{a.report.ExportedPacketState.RecvSequences, "ibc.channel_genesis.recv_sequences"},
		{a.report.ExportedPacketState.AckSequences, "ibc.channel_genesis.ack_sequences"},
	} {
		for _, record := range sequenceSet.records {
			if _, exists := channelByKey[record.Channel.String()]; !exists {
				a.add(
					severityError,
					"PACKET_SEQUENCE_CHANNEL_MISSING",
					"exported packet sequence refers to a channel absent from ibc.channel_genesis.channels",
					sequenceSet.location,
					record.Channel.String(),
				)
			}
		}
	}

	feePacketChannels := make(map[string]PacketRef)
	for _, packetFee := range a.report.FeeIBC.PacketFees {
		feePacketChannels[packetFee.Packet.String()] = packetFee.Packet
	}
	for _, packet := range feePacketChannels {
		ref := ChannelRef{PortID: packet.PortID, ChannelID: packet.ChannelID}
		if _, exists := channelByKey[ref.String()]; !exists {
			a.add(
				severityError,
				"PACKET_FEE_CHANNEL_MISSING",
				"exported packet fee refers to a channel absent from ibc.channel_genesis.channels",
				"feeibc.identified_fees",
				packet.String(),
			)
		}
	}
	for _, forward := range a.report.FeeIBC.ForwardRelayers {
		ref := ChannelRef{PortID: forward.Packet.PortID, ChannelID: forward.Packet.ChannelID}
		if _, exists := channelByKey[ref.String()]; !exists {
			a.add(
				severityError,
				"FORWARD_RELAYER_CHANNEL_MISSING",
				"forward relayer state refers to a channel absent from ibc.channel_genesis.channels",
				"feeibc.forward_relayers",
				forward.Packet.String(),
			)
		}
	}
	for _, payee := range a.report.FeeIBC.RegisteredPayees {
		if _, exists := channelIDs[payee.ChannelID]; !exists {
			a.add(
				severityError,
				"REGISTERED_PAYEE_CHANNEL_MISSING",
				"registered payee refers to a channel absent from ibc.channel_genesis.channels",
				"feeibc.registered_payees",
				payee.ChannelID,
			)
		}
	}
	for _, payee := range a.report.FeeIBC.RegisteredCounterpartyPayees {
		if _, exists := channelIDs[payee.ChannelID]; !exists {
			a.add(
				severityError,
				"COUNTERPARTY_PAYEE_CHANNEL_MISSING",
				"registered counterparty payee refers to a channel absent from ibc.channel_genesis.channels",
				"feeibc.registered_counterparty_payees",
				payee.ChannelID,
			)
		}
	}
	return nil
}

func (a *auditor) finish() {
	sort.Slice(a.report.FeeIBC.FeeEnabledChannels, func(i, j int) bool {
		return channelRefLess(a.report.FeeIBC.FeeEnabledChannels[i], a.report.FeeIBC.FeeEnabledChannels[j])
	})
	sort.SliceStable(a.report.FeeIBC.PacketFees, func(i, j int) bool {
		return packetFeeKey(a.report.FeeIBC.PacketFees[i]) < packetFeeKey(a.report.FeeIBC.PacketFees[j])
	})
	sort.Slice(a.report.FeeIBC.RegisteredPayees, func(i, j int) bool {
		left, right := a.report.FeeIBC.RegisteredPayees[i], a.report.FeeIBC.RegisteredPayees[j]
		return left.ChannelID+"\x00"+left.Relayer+"\x00"+left.Payee <
			right.ChannelID+"\x00"+right.Relayer+"\x00"+right.Payee
	})
	sort.Slice(a.report.FeeIBC.RegisteredCounterpartyPayees, func(i, j int) bool {
		left, right := a.report.FeeIBC.RegisteredCounterpartyPayees[i], a.report.FeeIBC.RegisteredCounterpartyPayees[j]
		return left.ChannelID+"\x00"+left.Relayer+"\x00"+left.CounterpartyPayee <
			right.ChannelID+"\x00"+right.Relayer+"\x00"+right.CounterpartyPayee
	})
	sort.Slice(a.report.FeeIBC.ForwardRelayers, func(i, j int) bool {
		left, right := a.report.FeeIBC.ForwardRelayers[i], a.report.FeeIBC.ForwardRelayers[j]
		if left.Packet.String() != right.Packet.String() {
			return packetRefLess(left.Packet, right.Packet)
		}
		return left.Address < right.Address
	})
	sort.Slice(a.report.Channels, func(i, j int) bool {
		left := ChannelRef{PortID: a.report.Channels[i].PortID, ChannelID: a.report.Channels[i].ChannelID}
		right := ChannelRef{PortID: a.report.Channels[j].PortID, ChannelID: a.report.Channels[j].ChannelID}
		return channelRefLess(left, right)
	})
	sort.SliceStable(a.report.Findings, func(i, j int) bool {
		left, right := a.report.Findings[i], a.report.Findings[j]
		return findingKey(left) < findingKey(right)
	})

	stateCounts := make(map[string]int)
	flushing := 0
	for _, channel := range a.report.Channels {
		stateCounts[channel.State]++
		if channel.State == "STATE_FLUSHING" || channel.State == "STATE_FLUSHCOMPLETE" {
			flushing++
		}
	}
	states := make([]string, 0, len(stateCounts))
	for state := range stateCounts {
		states = append(states, state)
	}
	sort.Strings(states)
	a.report.Summary.ChannelStates = make([]StateCount, 0, len(states))
	for _, state := range states {
		a.report.Summary.ChannelStates = append(
			a.report.Summary.ChannelStates,
			StateCount{State: state, Count: stateCounts[state]},
		)
	}
	a.report.Summary.FeeEnabledChannels = len(a.report.FeeIBC.FeeEnabledChannels)
	a.report.Summary.IdentifiedPacketFeeRecords = a.identifiedFeeRecords
	a.report.Summary.PacketFees = len(a.report.FeeIBC.PacketFees)
	a.report.Summary.RegisteredPayees = len(a.report.FeeIBC.RegisteredPayees)
	a.report.Summary.RegisteredCounterpartyPayees = len(a.report.FeeIBC.RegisteredCounterpartyPayees)
	a.report.Summary.ForwardRelayers = len(a.report.FeeIBC.ForwardRelayers)
	a.report.Summary.FeeIBCNonzeroBalanceDenoms = len(a.report.FeeIBC.ModuleAccountBalances)
	a.report.Summary.Channels = len(a.report.Channels)
	a.report.Summary.FlushingChannels = flushing
	a.report.Summary.OutstandingCommitments = len(a.report.ExportedPacketState.Commitments)
	a.report.Summary.ExportedAcknowledgements = len(a.report.ExportedPacketState.Acknowledgements)
	a.report.Summary.ExportedReceipts = len(a.report.ExportedPacketState.Receipts)
	for _, finding := range a.report.Findings {
		switch finding.Severity {
		case severityError:
			a.report.Summary.Errors++
		case severityWarning:
			a.report.Summary.Warnings++
		}
	}
}

func addCoinSets(left, right []Coin) ([]Coin, error) {
	result := coinMap(left)
	if err := accumulateCoins(result, right); err != nil {
		return nil, err
	}
	return coinMapToSlice(result), nil
}

func accumulateCoins(result map[string]*big.Int, coins []Coin) error {
	for _, coin := range coins {
		amount := new(big.Int)
		amount.SetString(coin.Amount, 10)
		if current, ok := result[coin.Denom]; ok {
			amount.Add(amount, current)
		}
		if amount.BitLen() > 256 {
			return fmt.Errorf("denomination %s exceeds the Cosmos SDK 256-bit integer bound", coin.Denom)
		}
		result[coin.Denom] = amount
	}
	return nil
}

func maximumCoins(left, right []Coin) []Coin {
	leftMap := coinMap(left)
	rightMap := coinMap(right)
	for denom, rightAmount := range rightMap {
		leftAmount, ok := leftMap[denom]
		if !ok || rightAmount.Cmp(leftAmount) > 0 {
			leftMap[denom] = new(big.Int).Set(rightAmount)
		}
	}
	return coinMapToSlice(leftMap)
}

func coinMap(coins []Coin) map[string]*big.Int {
	result := make(map[string]*big.Int, len(coins))
	for _, coin := range coins {
		amount := new(big.Int)
		if _, ok := amount.SetString(coin.Amount, 10); !ok {
			panic("validated coin amount became invalid")
		}
		result[coin.Denom] = amount
	}
	return result
}

func coinMapToSlice(values map[string]*big.Int) []Coin {
	denoms := make([]string, 0, len(values))
	for denom, amount := range values {
		if amount.Sign() > 0 {
			denoms = append(denoms, denom)
		}
	}
	sort.Strings(denoms)
	result := make([]Coin, 0, len(denoms))
	for _, denom := range denoms {
		result = append(result, Coin{Denom: denom, Amount: values[denom].String()})
	}
	return result
}

func equalCoins(left, right []Coin) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func channelRefLess(left, right ChannelRef) bool {
	if left.PortID != right.PortID {
		return left.PortID < right.PortID
	}
	return left.ChannelID < right.ChannelID
}

func packetRefLess(left, right PacketRef) bool {
	if left.PortID != right.PortID {
		return left.PortID < right.PortID
	}
	if left.ChannelID != right.ChannelID {
		return left.ChannelID < right.ChannelID
	}
	leftSequence, _ := strconv.ParseUint(left.Sequence, 10, 64)
	rightSequence, _ := strconv.ParseUint(right.Sequence, 10, 64)
	return leftSequence < rightSequence
}

func packetFeeKey(record PacketFeeRecord) string {
	return record.Packet.String() + "\x00" +
		record.RefundAddress + "\x00" +
		strings.Join(record.Relayers, "\x01") + "\x00" +
		coinsKey(record.RecvFee) + "\x00" +
		coinsKey(record.AckFee) + "\x00" +
		coinsKey(record.TimeoutFee)
}

func coinsKey(coins []Coin) string {
	parts := make([]string, len(coins))
	for i, coin := range coins {
		parts[i] = coin.Denom + "=" + coin.Amount
	}
	return strings.Join(parts, ",")
}

func findingKey(finding Finding) string {
	rank := "2"
	if finding.Severity == severityError {
		rank = "1"
	}
	return rank + "\x00" + finding.Code + "\x00" + finding.Subject + "\x00" + finding.Location + "\x00" + finding.Message
}
