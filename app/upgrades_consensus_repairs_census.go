package app

import (
	"bytes"
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
	alignment "github.com/zerone-chain/zerone/x/alignment/types"
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

var consensusRepairsCensusNames = []string{"seat_runoffs", "reformulations", "verification_histories", "qualification_escrow", "challenge_backing", "alignment_inputs", "halt_reasons", "adapter_minima", "ibc_refunds"}

type ConsensusRepairsFinding struct {
	Census  string `json:"census,omitempty"`
	Module  string `json:"module"`
	Key     string `json:"key_hex"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}
type ConsensusRepairsRollback struct {
	LIPID           string `json:"lip_id"`
	ActivationBlock uint64 `json:"activation_block"`
	DueNow          bool   `json:"due_now"`
	Outcome         string `json:"outcome_on_current_state"`
	Reason          string `json:"reason"`
}
type ConsensusRepairsCensus struct {
	Height           int64                                      `json:"height"`
	Digest           string                                     `json:"digest"`
	Digests          map[string]string                          `json:"census_digests"`
	Records          int                                        `json:"records"`
	Bytes            int                                        `json:"bytes"`
	Findings         []ConsensusRepairsFinding                  `json:"findings"`
	Stores           map[string]ConsensusRepairsSize            `json:"stores"`
	Prefixes         map[string]map[string]ConsensusRepairsSize `json:"prefixes"`
	Censuses         map[string]ConsensusRepairsSize            `json:"census_sizes"`
	Reads            map[string][]ConsensusRepairsReadSize      `json:"census_reads"`
	DomainStatuses   map[string]int                             `json:"domain_statuses"`
	PendingRollbacks []ConsensusRepairsRollback                 `json:"pending_rollbacks"`
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
	// Never release this cache: even an accidentally writing future helper
	// cannot commit state or events, including on direct offline/startup calls.
	ctx, _ = ctx.CacheContext()
	rows, err := app.consensusRepairsCensusRows(ctx, &report)
	if err != nil {
		return report, err
	}

	censusName := "" // Marker faults are outside the nine named censuses.
	add := func(module, key, outcome, reason string) {
		report.Findings = append(report.Findings, ConsensusRepairsFinding{censusName, module, hex.EncodeToString([]byte(key)), outcome, reason})
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
		store := ctx.KVStore(app.keys[loc.store])
		if store.Has(loc.key) {
			value := store.Get(loc.key)
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
	censusName = "seat_runoffs"
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
	censusName = "reformulations"
	params := knowledge.DefaultParams()
	if value, ok := rows["knowledge"][string(knowledge.ParamsKey)]; ok {
		if e := proto.Unmarshal(value, &params); e != nil {
			add("knowledge", string(knowledge.ParamsKey), "blocker", e.Error())
		}
	}
	nulBounties, nulAugmentations := 0, 0
	scan("knowledge", 0x62, func(key string, value []byte) error {
		var p knowledge.AugmentationBounty
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x62, p.Id); e != nil {
			return e
		}
		if strings.ContainsRune(p.Id, 0) {
			nulBounties++
			add("knowledge", key, "held", "NUL bounty ID; zc-reformulation-id-delimiter hold, no rewrite")
		}
		return nil
	})
	// Match indexes against decoded primary identities, not ambiguous NUL
	// splitting. Held delimiter collisions must not mask dangling indexes.
	indexOwners := make(map[string]int)
	scan("knowledge", 0x63, func(key string, value []byte) error {
		var p knowledge.Augmentation
		if e := proto.Unmarshal(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x63, p.Id); e != nil {
			return e
		}
		if strings.ContainsRune(p.Id, 0) || strings.ContainsRune(p.BountyId, 0) {
			nulAugmentations++
			add("knowledge", key, "held", "NUL augmentation or bounty ID; zc-reformulation-id-delimiter hold, no rewrite")
		}
		if p.BountyId != "" {
			indexKey := string(knowledge.AugmentationByBountyKey(p.BountyId, p.Id))
			indexOwners[indexKey]++
			if !bytes.Equal(rows["knowledge"][indexKey], []byte{1}) {
				return fmt.Errorf("missing/malformed augmentation bounty index")
			}
		}
		if reason := knowledgekeeper.ReformulationHoldReason(&p, &params); reason != "" {
			add("knowledge", key, "held", reason+"; pooled bounty escrow retained, no inferred entitlement")
		}
		return nil
	})
	scan("knowledge", 0x65, func(key string, value []byte) error {
		if !bytes.Equal(value, []byte{1}) {
			return fmt.Errorf("malformed bounty index")
		}
		if indexOwners[key] == 0 {
			return fmt.Errorf("dangling/foreign bounty index")
		}
		if strings.Count(key[1:], "\x00") > 1 {
			add("knowledge", key, "held", fmt.Sprintf("NUL-delimited bounty index retained; primary_records=%d", indexOwners[key]))
		}
		return nil
	})
	add("knowledge", "", "held", fmt.Sprintf("NUL IDs retained: bounties=%d augmentations_or_bounty_references=%d", nulBounties, nulAugmentations))
	censusName = "verification_histories"
	histories := 0
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
			histories++
		}
		return nil
	})
	add("capture_defense", "", "held", fmt.Sprintf("histories retained; integer variance prospective; histories_with_multiple_samples=%d digest=%s", histories, report.Digests["verification_histories"]))
	censusName = "qualification_escrow"
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
		if p.Status == qualification.QualificationStatus_QUALIFICATION_STATUS_EXPIRED && amount.Sign() > 0 {
			expired.Add(expired, amount)
			add("qualification", key, "held", "EXPIRED retained amount="+amount.String()+"; refund held, prior payment unknown; Yu's hold decision required if positive")
		}
		return nil
	})
	qbalance := app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress("qualification"), "uzrn").Amount.String()
	add("qualification", "", "held", "escrow balance="+qbalance+"; EXPIRED retained amounts="+expired.String()+"; neither proves unpaid entitlement")

	censusName = "challenge_backing"
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
	censusName = "alignment_inputs"
	registry, active, domains, activeDomains := 0, 0, 0, 0
	report.DomainStatuses = make(map[string]int)
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
		report.DomainStatuses[p.Status]++
		if p.Status == "active" {
			activeDomains++
		}
		return nil
	})
	mint, e := consensusRepairsAmount(string(rows["vesting_rewards"]["\x0b"]))
	if e != nil {
		add("vesting_rewards", "\x0b", "blocker", e.Error())
		mint = new(big.Int)
	}
	supply := app.BankKeeper.GetSupply(ctx, "uzrn").Amount.BigInt()
	add("alignment", "", "held", fmt.Sprintf("prospective inputs only; bank_supply=%s mint_ledger=%s registry=%d active=%d active_stake=%s domains=%d active_domains=%d legacy_staking_ratio_bps=%d fixed_staking_ratio_bps=%d", supply, mint, registry, active, stake, domains, activeDomains, consensusRepairsRatio(stake, mint), consensusRepairsRatio(stake, supply)))
	statuses := make([]string, 0, len(report.DomainStatuses))
	for status := range report.DomainStatuses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		if status != "active" {
			add("zerone_ontology", "", "held", fmt.Sprintf("inactive domain status=%q count=%d", status, report.DomainStatuses[status]))
		}
	}
	if stake.Sign() == 0 && mint.Sign() == 0 && supply.Sign() > 0 {
		add("alignment", "", "blocker", "Q1 not-observed/neutral pacing is absent: zero active stake and mint ledger with positive bank supply changes neutral to zero")
	}

	censusName = "halt_reasons"
	phase := *gov.DefaultResearchFundGovernanceState()
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
			reason := "out-of-window finalized halt retained; not current gate evidence"
			if p.StartBlock >= phase.PhaseStartedAtBlock && p.StartBlock <= uint64(ctx.BlockHeight()) {
				// Match CountResearchFundHalts: old/future payloads are not evidence.
				var halt emergency.EmergencyHaltProposal
				if e := proto.Unmarshal(p.ProposalData, &halt); e != nil {
					return e
				}
				if halt.Id != p.Id {
					return fmt.Errorf("halt proposal/ceremony ID mismatch")
				}
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
	// Ascending ID order leaves the same trailing streak as the production
	// reverse walk; unlike the keeper iterator, malformed records cannot vanish.
	expiredStreak := 0
	scan("zerone_gov", 0x06, func(key string, value []byte) error {
		var p gov.ResearchSpendProposal
		if e := json.Unmarshal(value, &p); e != nil {
			return e
		}
		if key != string(gov.ResearchSpendKey(p.ProposalId)) {
			return fmt.Errorf("research proposal key/id mismatch")
		}
		if p.Stage == string(gov.ResearchStageExpired) {
			expiredStreak++
		} else {
			expiredStreak = 0
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
		if p.TargetPhase < gov.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR || p.TargetPhase > gov.ResearchFundPhase_RESEARCH_FUND_PHASE_FULL_GOVERNANCE {
			return fmt.Errorf("malformed phase transition target")
		}
		switch p.Stage {
		case gov.PhaseTransitionStageDiscussion, gov.PhaseTransitionStageVoting, gov.PhaseTransitionStagePassed, gov.PhaseTransitionStageFailed, gov.PhaseTransitionStagePending, gov.PhaseTransitionStageActivated, gov.PhaseTransitionStageCancelled:
		default:
			return fmt.Errorf("malformed phase transition stage")
		}
		var lip gov.LIP
		if e := json.Unmarshal(rows["zerone_gov"][string(gov.LIPKey(p.LipID))], &lip); e != nil {
			return fmt.Errorf("missing/malformed phase transition LIP: %w", e)
		}
		if lip.Id != p.LipID || !gov.IsPhaseTransitionCategory(lip.Category) || p.IsRollback != (lip.Category == gov.CategoryPhaseRollback) {
			return fmt.Errorf("phase transition metadata/LIP mismatch")
		}
		if p.IsRollback && p.Stage == gov.PhaseTransitionStagePending {
			// This is the activation predicate, NOT ValidatePhaseRollback:
			// admission rejects the pending proposal itself. Do not move height
			// forward or simulate other BeginBlock writes to predict the future.
			outcome, reason := "execute", "current scoped halt or gridlock justifies rollback"
			if phase.CurrentPhase <= gov.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR || p.TargetPhase != phase.CurrentPhase-1 {
				outcome, reason = "cancel", "rollback target no longer valid"
			} else if expiredStreak < gov.RollbackGridlockThreshold && scopedHalts == 0 {
				outcome, reason = "cancel", gov.ErrNoRollbackJustification.Error()
				if unknownHalt {
					reason = emergency.ErrHaltEvidence.Error()
				}
			}
			report.PendingRollbacks = append(report.PendingRollbacks, ConsensusRepairsRollback{p.LipID, p.ActivationBlock, p.ActivationBlock != 0 && p.ActivationBlock <= uint64(ctx.BlockHeight()), outcome, reason})
		}
		return nil
	})
	forwardCount := scopedHalts
	if unknownHalt {
		forwardCount++
	}
	add("emergency", "", "held", fmt.Sprintf("halt history retained: legacy_count=%d scoped_valid=%d unknown=%t conservative_forward_count=%d since_block=%d; pending rollback admission is not reconstructible", legacyHalts, scopedHalts, unknownHalt, forwardCount, phase.PhaseStartedAtBlock))
	censusName = "adapter_minima"
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
	censusName = "ibc_refunds"
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
	flows := 0
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
		flows++
		return nil
	})
	add("ibcratelimit", "", "held", fmt.Sprintf("legacy packet flow retained; count=%d digest=%s; cannot debit any window on refund; capacity retained until reset", flows, report.Digests["ibc_refunds"]))
	scan("zerone_ibcrl", 0x03, func(key string, value []byte) error {
		parts := strings.Split(key[1:], "\x00")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || len(value) != 8 {
			return fmt.Errorf("corrupt IBC generation")
		}
		return fmt.Errorf("reserved IBC generation prefix 0x03 occupied before activation")
	})

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

// Match alignment/keeper.ratioBPS, including denominator-zero precedence and
// its Uint64-before-cap conversion. This is a reading, not a new sensor policy.
func consensusRepairsRatio(stake, denominator *big.Int) uint64 {
	if denominator.Sign() == 0 {
		return alignment.NeutralBPS
	}
	value := new(big.Int).Mul(stake, new(big.Int).SetUint64(alignment.BPS))
	value.Div(value, denominator)
	return min(value.Uint64(), alignment.BPS)
}
