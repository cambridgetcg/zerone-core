package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cc "github.com/zerone-chain/zerone/x/capture_challenge/types"
	cd "github.com/zerone-chain/zerone/x/capture_defense/types"
	emergency "github.com/zerone-chain/zerone/x/emergency/types"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	ibcrl "github.com/zerone-chain/zerone/x/ibcratelimit/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	ontology "github.com/zerone-chain/zerone/x/ontology/types"
	qualification "github.com/zerone-chain/zerone/x/qualification/types"
	staking "github.com/zerone-chain/zerone/x/staking/types"
	bridge "github.com/zerone-chain/zerone/x/substrate_bridge/types"
	"google.golang.org/protobuf/proto"
)

const consensusRepairsMaxRecords = 100_000
const consensusRepairsMaxBytes = 64 << 20

var consensusRepairsCensusNames = []string{"seat_runoffs", "reformulations", "verification_histories", "qualification_escrow", "challenge_backing", "alignment_inputs", "halt_reasons", "adapter_minima", "ibc_refunds"}

var consensusRepairsCensusStores = map[string][]string{
	"seat_runoffs": {"zerone_gov"}, "reformulations": {"knowledge"},
	"verification_histories": {"capture_defense"},
	"qualification_escrow":   {"qualification", "bank"},
	"challenge_backing":      {"capture_challenge", "bank"},
	"alignment_inputs":       {"vesting_rewards", "bank", "zerone_staking", "zerone_ontology"},
	"halt_reasons":           {"emergency", "zerone_gov"},
	"adapter_minima":         {"substrate_bridge"}, "ibc_refunds": {"zerone_ibcrl"},
}

type ConsensusRepairsFinding struct {
	Module  string `json:"module"`
	Key     string `json:"key_hex"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}
type ConsensusRepairsCensus struct {
	Height   int64                     `json:"height"`
	Digest   string                    `json:"digest"`
	Digests  map[string]string         `json:"census_digests"`
	Records  int                       `json:"records"`
	Bytes    int                       `json:"bytes"`
	Findings []ConsensusRepairsFinding `json:"findings"`
}

// Snapshot every mounted persistent store, not skip-on-error keeper iterators.
// The global bound deliberately refuses oversized copies; it never truncates.
// The same image supports the runtime exact-diff postcondition.
func (app *ZeroneApp) consensusRepairsSnapshot(ctx sdk.Context) (rows map[string]map[string][]byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			rows = nil
			err = fmt.Errorf("consensus repairs store read fault: %v", r)
		}
	}()
	rows = make(map[string]map[string][]byte, len(app.keys))
	names := make([]string, 0, len(app.keys))
	for name := range app.keys {
		names = append(names, name)
	}
	sort.Strings(names)
	count, total := 0, 0
	for _, name := range names {
		entries := make(map[string][]byte)
		rows[name] = entries
		err := func() (err error) {
			it := ctx.KVStore(app.keys[name]).Iterator(nil, nil)
			defer func() {
				if e := it.Close(); err == nil {
					err = e
				}
			}()
			for ; it.Valid(); it.Next() {
				count++
				total += len(it.Key()) + len(it.Value())
				if count > consensusRepairsMaxRecords || total > consensusRepairsMaxBytes {
					return fmt.Errorf("consensus repairs census resource ceiling exceeded (100000 records/64 MiB)")
				}
				entries[string(it.Key())] = bytes.Clone(it.Value())
			}
			if e := it.Error(); e != nil && (it.Valid() || (e.Error() != "invalid cacheMergeIterator" && e.Error() != "invalid memIterator")) {
				return e
			}
			return nil
		}()
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", name, err)
		}
	}
	return rows, nil
}

func consensusRepairsAmount(raw string) (*big.Int, error) {
	// Empty is a legacy spelling of zero in optional amount fields.
	if raw == "" {
		return new(big.Int), nil
	}
	for _, c := range raw {
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("invalid nonnegative amount %q", raw)
		}
	}
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	return value, nil
}

// CensusConsensusRepairsV1 is read-only and usable on a copied database before
// H-1. It describes source state, not scheduling authority or proof of payment.
// "held" means retained without redistribution; reasons distinguish actual
// execution holds from retained-history/prospective-input discontinuities.
func (app *ZeroneApp) CensusConsensusRepairsV1(ctx sdk.Context) (report ConsensusRepairsCensus, err error) {
	report.Height = ctx.BlockHeight()
	report.Findings = []ConsensusRepairsFinding{}
	if ctx.BlockHeight() < 0 {
		return report, fmt.Errorf("consensus repairs census requires a nonnegative height")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("consensus repairs census read fault: %v", r)
		}
	}()
	rows, err := app.consensusRepairsSnapshot(ctx)
	if err != nil {
		return report, err
	}
	names := make([]string, 0, len(rows))
	for name := range rows {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	storeDigests := make(map[string]string, len(rows))
	var length [8]byte
	for _, name := range names {
		storeHash := sha256.New()
		keys := make([]string, 0, len(rows[name]))
		for key := range rows[name] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := rows[name][key]
			report.Records++
			report.Bytes += len(key) + len(value)
			for _, field := range [][]byte{[]byte(name), []byte(key), value} {
				binary.BigEndian.PutUint64(length[:], uint64(len(field)))
				hash.Write(length[:])
				hash.Write(field)
				storeHash.Write(length[:])
				storeHash.Write(field)
			}
		}
		storeDigests[name] = hex.EncodeToString(storeHash.Sum(nil))
	}
	report.Digest = hex.EncodeToString(hash.Sum(nil))
	report.Digests = make(map[string]string, len(consensusRepairsCensusNames))
	for _, name := range consensusRepairsCensusNames {
		parts := []string{name}
		for _, store := range consensusRepairsCensusStores[name] {
			parts = append(parts, store, storeDigests[store])
		}
		encoded, _ := json.Marshal(parts)
		digest := sha256.Sum256(encoded)
		report.Digests[name] = hex.EncodeToString(digest[:])
	}
	add := func(module, key, outcome, reason string) {
		report.Findings = append(report.Findings, ConsensusRepairsFinding{module, hex.EncodeToString([]byte(key)), outcome, reason})
	}
	scan := func(module string, prefix byte, inspect func(string, []byte) error) {
		keys := make([]string, 0)
		for key := range rows[module] {
			if len(key) > 0 && key[0] == prefix {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			if e := inspect(key, rows[module][key]); e != nil {
				add(module, key, "blocker", e.Error())
			}
		}
	}
	identity := func(key string, prefix byte, id string) error {
		if id == "" || key != string([]byte{prefix})+id {
			return fmt.Errorf("record key/identity mismatch")
		}
		return nil
	}
	// Raw presence is checked as well as each official reader: present-empty is
	// never equivalent to absent, even on a backend that returns nil for Get.
	markers := app.ConsensusRepairsV1Markers()
	for i, loc := range consensusRepairsMarkerLocations() {
		if value, present := rows[loc.store][string(loc.key)]; present {
			reason := "marker already enabled"
			if !bytes.Equal(value, []byte{1}) {
				reason = "malformed activation marker (including present-empty)"
			}
			add(loc.store, string(loc.key), "blocker", reason)
		}
		on, e := markers[i].Enabled(ctx)
		if e != nil {
			add(loc.store, string(loc.key), "blocker", e.Error())
		} else if on { /* raw presence already reported */
		}
	}

	// #117: retain original IDs and check historical parent/child relationships.
	seats := make(map[uint64]*gov.SeatElectionProposal)
	var maxID uint64
	scan("zerone_gov", 0x10, func(key string, value []byte) error {
		var p gov.SeatElectionProposal
		if e := json.Unmarshal(value, &p); e != nil {
			return e
		}
		if p.ProposalId == 0 || key != string(gov.SeatElectionKey(p.ProposalId)) {
			return fmt.Errorf("seat key/id mismatch")
		}
		seats[p.ProposalId] = &p
		if p.ProposalId > maxID {
			maxID = p.ProposalId
		}
		return nil
	})
	next := uint64(1)
	if value, present := rows["zerone_gov"]["\x12"]; present {
		if len(value) != 8 {
			add("zerone_gov", "\x12", "blocker", "malformed next seat ID")
		} else {
			next = binary.BigEndian.Uint64(value)
		}
	}
	if next <= maxID || next > math.MaxUint64-2 {
		add("zerone_gov", "\x12", "blocker", "seat counter is behind stored IDs or lacks runoff allocation headroom")
	}
	children := make(map[string]int)
	parentsReferenced := make(map[uint64]bool)
	ids := make([]uint64, 0, len(seats))
	for id := range seats {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		p := seats[id]
		key := string(gov.SeatElectionKey(id))
		if !p.IsRunoff {
			if len(p.RunoffParentIds) != 0 {
				add("zerone_gov", key, "blocker", "non-runoff has parent links")
			}
			continue
		}
		if len(p.RunoffParentIds) < 2 {
			add("zerone_gov", key, "blocker", "runoff lacks parents")
			continue
		}
		ordered := append([]uint64(nil), p.RunoffParentIds...)
		sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
		valid := true
		for i, parentID := range ordered {
			parent := seats[parentID]
			if parentID >= id || parent == nil || parent.SeatIndex != p.SeatIndex || parent.Stage != "runoff" || (i > 0 && ordered[i-1] == parentID) {
				valid = false
			} else {
				parentsReferenced[parentID] = true
			}
		}
		if !valid {
			add("zerone_gov", key, "blocker", "inconsistent runoff links")
		}
		group, _ := json.Marshal(ordered)
		children[string(group)]++
	}
	groups := make([]string, 0, len(children))
	for group := range children {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	for _, group := range groups {
		if children[group] != 2 {
			add("zerone_gov", "\x10", "blocker", "runoff parent group does not have exactly two children: "+group)
		}
	}
	for _, id := range ids {
		if seats[id].Stage == "runoff" && !parentsReferenced[id] {
			add("zerone_gov", string(gov.SeatElectionKey(id)), "blocker", "orphaned runoff parent")
		}
	}

	// #118: bounded equivalent of CensusReformulationHolds, including index completeness.
	params := knowledge.DefaultParams()
	if value, ok := rows["knowledge"][string(knowledge.ParamsKey)]; ok {
		if e := proto.Unmarshal(value, &params); e != nil {
			add("knowledge", string(knowledge.ParamsKey), "blocker", e.Error())
		}
	}
	scan("knowledge", 0x62, func(key string, value []byte) error {
		var p knowledge.AugmentationBounty
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x62, p.Id); e != nil {
			return e
		}
		if strings.ContainsRune(p.Id, 0) {
			return fmt.Errorf("NUL bounty ID")
		}
		return nil
	})
	scan("knowledge", 0x63, func(key string, value []byte) error {
		var p knowledge.Augmentation
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x63, p.Id); e != nil {
			return e
		}
		if strings.ContainsRune(p.Id, 0) || strings.ContainsRune(p.BountyId, 0) {
			return fmt.Errorf("NUL augmentation or bounty ID")
		}
		if p.BountyId != "" && !bytes.Equal(rows["knowledge"][string(knowledge.AugmentationByBountyKey(p.BountyId, p.Id))], []byte{1}) {
			return fmt.Errorf("missing/malformed augmentation bounty index")
		}
		if reason := knowledgekeeper.ReformulationHoldReason(&p, &params); reason != "" {
			add("knowledge", key, "held", reason+"; pooled bounty escrow retained, no inferred entitlement")
		}
		return nil
	})
	scan("knowledge", 0x65, func(key string, value []byte) error {
		parts := strings.Split(key[1:], "\x00")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || !bytes.Equal(value, []byte{1}) {
			return fmt.Errorf("malformed bounty index")
		}
		var p knowledge.Augmentation
		if e := proto.Unmarshal(rows["knowledge"][string(knowledge.AugmentationKey(parts[1]))], &p); e != nil {
			return e
		}
		if p.Id != parts[1] || p.BountyId != parts[0] {
			return fmt.Errorf("dangling/foreign bounty index")
		}
		return nil
	})
	scan("capture_defense", 0x05, func(key string, value []byte) error {
		var p cd.VerificationHistoryEntry
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if p.Domain == "" || p.RoundId == "" {
			return fmt.Errorf("empty verification history identity")
		}
		if e := identity(key, 0x05, p.Domain+"/"+p.RoundId); e != nil {
			return e
		}
		if len(p.SubmitBlocks) >= 2 {
			add("capture_defense", key, "held", fmt.Sprintf("history retained; %d samples evaluated with integer variance prospectively", len(p.SubmitBlocks)))
		}
		return nil
	})
	qparams := qualification.DefaultParams()
	if value, ok := rows["qualification"][string(qualification.ParamsKey)]; ok {
		if e := proto.Unmarshal(value, qparams); e != nil {
			add("qualification", string(qualification.ParamsKey), "blocker", e.Error())
		}
	}
	expired := new(big.Int)
	scan("qualification", 0x01, func(key string, value []byte) error {
		var p qualification.DomainQualification
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if p.Validator == "" || p.Domain == "" {
			return fmt.Errorf("empty qualification identity")
		}
		if e := identity(key, 0x01, p.Validator+"/"+p.Domain); e != nil {
			return e
		}
		amount, e := consensusRepairsAmount(p.StakedAmount)
		if e != nil {
			return e
		}
		if p.Pathway == qualification.QualificationPathway_QUALIFICATION_PATHWAY_STAKE && qparams.StakeLockPeriod > math.MaxUint64-p.GrantedAt {
			return fmt.Errorf("StakeLockPeriod overflow")
		}
		if p.Status == qualification.QualificationStatus_QUALIFICATION_STATUS_EXPIRED && p.StakedAmount != "" {
			expired.Add(expired, amount)
			add("qualification", key, "held", "EXPIRED retained amount="+amount.String()+"; refund held, prior payment unknown; Yu's hold decision required if positive")
		}
		return nil
	})
	qbalance := app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress("qualification"), "uzrn").Amount.String()
	add("qualification", "", "held", "escrow balance="+qbalance+"; EXPIRED retained amounts="+expired.String()+"; neither proves unpaid entitlement")

	required := new(big.Int)
	scan("capture_challenge", 0x02, func(key string, value []byte) error {
		var p cc.DomainBountyPool
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x02, p.Domain); e != nil {
			return e
		}
		amount, e := consensusRepairsAmount(p.Balance)
		if e != nil {
			return e
		}
		required.Add(required, amount)
		return nil
	})
	scan("capture_challenge", 0x01, func(key string, value []byte) error {
		var p cc.CaptureChallenge
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x01, p.Id); e != nil {
			return e
		}
		amount, e := consensusRepairsAmount(p.Stake)
		if e != nil {
			return e
		}
		switch p.Status {
		case cc.ChallengeStatus_CHALLENGE_STATUS_OPEN, cc.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE, cc.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW:
			if p.Resolution != nil {
				return fmt.Errorf("nonterminal challenge has resolution")
			}
			required.Add(required, amount)
		case cc.ChallengeStatus_CHALLENGE_STATUS_RESOLVED:
			if p.Resolution == nil {
				return fmt.Errorf("resolved challenge lacks resolution")
			}
			if p.Resolution.Outcome == cc.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED {
				return fmt.Errorf("legacy RESOLVED+REJECTED challenge requires reconciliation")
			}
		case cc.ChallengeStatus_CHALLENGE_STATUS_EXPIRED:
		default:
			return fmt.Errorf("unknown challenge status")
		}
		return nil
	})
	balance := app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress("capture_challenge"), "uzrn").Amount.BigInt()
	if balance.Cmp(required) < 0 {
		add("capture_challenge", "", "blocker", "pool deficit: balance="+balance.String()+" required="+required.String())
	} else {
		add("capture_challenge", "", "held", "backing retained: balance="+balance.String()+" required="+required.String())
	}

	// #122/#124: preserve raw inputs and surface the denominator/count discontinuity.
	registry, active, domains, activeDomains := 0, 0, 0, 0
	stake := new(big.Int)
	scan("zerone_staking", 0x01, func(key string, value []byte) error {
		var p staking.Validator
		if e := json.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x01, p.OperatorAddress); e != nil {
			return e
		}
		amount, e := consensusRepairsAmount(p.TotalStake)
		if e != nil {
			return e
		}
		registry++
		if p.IsActive {
			active++
			stake.Add(stake, amount)
		}
		return nil
	})
	scan("zerone_ontology", 0x02, func(key string, value []byte) error {
		var p ontology.Domain
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x02, p.Name); e != nil {
			return e
		}
		domains++
		if p.Status == "active" {
			activeDomains++
		} else if p.Status != "proposed" && p.Status != "deprecated" {
			return fmt.Errorf("unknown domain status")
		}
		return nil
	})
	mint, e := consensusRepairsAmount(string(rows["vesting_rewards"]["\x0b"]))
	if e != nil {
		add("vesting_rewards", "\x0b", "blocker", e.Error())
		mint = new(big.Int)
	}
	supply := app.BankKeeper.GetSupply(ctx, "uzrn").Amount.BigInt()
	add("alignment", "", "held", fmt.Sprintf("prospective inputs only; bank_supply=%s mint_ledger=%s registry=%d active=%d active_stake=%s domains=%d active_domains=%d", supply, mint, registry, active, stake, domains, activeDomains))
	if registry == 0 {
		add("alignment", "", "blocker", "Q1 not-observed/neutral pacing is absent from reviewed stack; empty registry must not silently activate critical pacing")
	}

	var phase gov.ResearchFundGovernanceState
	if value, present := rows["zerone_gov"]["\x0d"]; present {
		if err := json.Unmarshal(value, &phase); err != nil {
			add("zerone_gov", "\x0d", "blocker", "malformed research-fund phase state")
		}
	}
	if phase.PhaseStartedAtBlock > uint64(ctx.BlockHeight()) {
		add("zerone_gov", "\x0d", "blocker", "research-fund phase begins in the future")
	}
	legacyHalts, scopedHalts := 0, 0
	unknownHalt := false
	scan("emergency", 0x02, func(key string, value []byte) error {
		var p emergency.EmergencyCeremony
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x02, p.Id); e != nil {
			return e
		}
		if p.Type == string(emergency.CeremonyHalt) && p.Phase == string(emergency.PhaseFinalized) {
			legacyHalts++
			var halt emergency.EmergencyHaltProposal
			if e := proto.Unmarshal(p.ProposalData, &halt); e != nil {
				return e
			}
			if halt.Id != p.Id {
				return fmt.Errorf("halt proposal/ceremony ID mismatch")
			}
			reason := "out-of-window finalized halt retained; not current gate evidence"
			if p.StartBlock >= phase.PhaseStartedAtBlock && p.StartBlock <= uint64(ctx.BlockHeight()) {
				if halt.Reason == emergency.HaltReasonResearchFund {
					scopedHalts++
					reason = "scoped research_fund halt retained"
				} else {
					unknownHalt = true
					reason = "unknown halt reason retained; conservative forward gate, not a justified rollback"
				}
			}
			add("emergency", key, "held", fmt.Sprintf("%s; start_block=%d", reason, p.StartBlock))
		}
		return nil
	})
	scan("zerone_gov", 0x14, func(key string, value []byte) error {
		var p gov.PhaseTransitionProposal
		if e := json.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x14, p.LipID); e != nil {
			return e
		}
		if p.IsRollback && p.Stage == "pending_activation" {
			return fmt.Errorf("pending legacy rollback: admission evidence not recorded; independent reconciliation required")
		}
		return nil
	})
	forwardCount := scopedHalts
	if unknownHalt {
		forwardCount++
	}
	add("emergency", "", "held", fmt.Sprintf("halt history retained: legacy_count=%d scoped_valid=%d unknown=%t conservative_forward_count=%d since_block=%d; pending rollback admission is not reconstructible", legacyHalts, scopedHalts, unknownHalt, forwardCount, phase.PhaseStartedAtBlock))
	scan("substrate_bridge", 0x84, func(key string, value []byte) error {
		var p bridge.AdapterRegistration
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x84, p.AdapterId); e != nil {
			return e
		}
		if p.RequiredQualificationDomain != "" {
			switch p.MinQualificationStatus {
			case bridge.QualificationStatus_QUALIFICATION_STATUS_UNSPECIFIED, bridge.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY, bridge.QualificationStatus_QUALIFICATION_STATUS_ACTIVE:
			default:
				add("substrate_bridge", key, "held", "unsupported qualification minimum retained; new work refused")
			}
		}
		return nil
	})
	scan("zerone_ibcrl", 0x01, func(key string, value []byte) error {
		var p ibcrl.RateLimit
		if e := json.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x01, p.ChannelId+"\x00"+p.Denom); e != nil {
			return e
		}
		if e := ibcrl.ValidateRateLimit(&p); e != nil {
			return e
		}
		if strings.ContainsRune(p.ChannelId, 0) || strings.ContainsRune(p.Denom, 0) {
			return fmt.Errorf("ambiguous IBC rule identity")
		}
		for _, amount := range []string{p.CurrentSend, p.CurrentRecv} {
			if _, err := consensusRepairsAmount(amount); err != nil {
				return err
			}
		}
		if p.WindowBlocks > math.MaxUint64-p.WindowStart || p.WindowBlocks > math.MaxUint64-uint64(ctx.BlockHeight()) {
			return fmt.Errorf("overflow-prone IBC window")
		}
		return nil
	})
	scan("zerone_ibcrl", 0x02, func(key string, value []byte) error {
		var p struct {
			ChannelID      string  `json:"channel_id"`
			Sequence       uint64  `json:"sequence"`
			Denom          string  `json:"denom"`
			Amount         string  `json:"amount"`
			WindowStart    *uint64 `json:"window_start"`
			RuleGeneration *uint64 `json:"rule_generation"`
		}
		if e := json.Unmarshal(value, &p); e != nil {
			return e
		}
		expected := append([]byte{0x02}, []byte(p.ChannelID)...)
		expected = append(expected, 0)
		expected = binary.BigEndian.AppendUint64(expected, p.Sequence)
		if p.ChannelID == "" || p.Denom == "" || key != string(expected) {
			return fmt.Errorf("packet flow key/identity mismatch")
		}
		if _, e := consensusRepairsAmount(p.Amount); e != nil {
			return e
		}
		if p.WindowStart != nil || p.RuleGeneration != nil {
			return fmt.Errorf("pre-carrier flow unexpectedly has charge identity")
		}
		add("ibcratelimit", key, "held", "legacy packet flow retained; cannot debit any window on refund; capacity retained until reset")
		return nil
	})
	scan("zerone_ibcrl", 0x03, func(key string, value []byte) error {
		parts := strings.Split(key[1:], "\x00")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || len(value) != 8 {
			return fmt.Errorf("corrupt IBC generation")
		}
		return fmt.Errorf("reserved IBC generation prefix 0x03 occupied before activation")
	})
	// Reserve the ten absent marker rows plus receipt, including a conservative
	// metadata envelope, so a near-ceiling source refuses at H-1, not only in
	// the post-write diff. No truncation or increase of the bound at H.
	encodedReport, encodeErr := json.Marshal(report)
	if encodeErr != nil {
		return report, encodeErr
	}
	if report.Records+11 > consensusRepairsMaxRecords || report.Bytes+len(encodedReport)+4096 > consensusRepairsMaxBytes {
		add("carrier", "", "blocker", "insufficient activation receipt/marker headroom within census resource ceiling")
	}
	sort.SliceStable(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		return a.Reason < b.Reason
	})
	for _, finding := range report.Findings {
		if finding.Outcome == "blocker" {
			return report, fmt.Errorf("consensus repairs census blocker: %s/%s: %s", finding.Module, finding.Key, finding.Reason)
		}
	}
	return report, nil
}
