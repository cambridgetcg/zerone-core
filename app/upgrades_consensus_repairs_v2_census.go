package app

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	alignment "github.com/zerone-chain/zerone/x/alignment/types"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	staking "github.com/zerone-chain/zerone/x/staking/types"
	"google.golang.org/protobuf/proto"
)

var consensusRepairsV2CensusNames = []string{"q1_health", "q2_rounds", "q5_calibration", "pr37_support", "fx7_accounting", "fx8_retained"}

func consensusRepairsV2Reads() (map[string][]consensusRepairsRead, error) {
	balance, err := collections.EncodeKeyWithPrefix(banktypes.BalancesPrefix.Bytes(), collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey), collections.Join(authtypes.NewModuleAddress(knowledge.TrainingFundModuleName), "uzrn"))
	if err != nil {
		return nil, err
	}
	return map[string][]consensusRepairsRead{
		"q1_health":      {{"alignment", []byte{1}, true}, {"alignment", []byte{2}, true}, {"zerone_staking", []byte{1}, false}},
		"q2_rounds":      {{"knowledge", []byte{2}, false}, {"knowledge", []byte{3}, false}, {"knowledge", []byte{6}, true}, {"knowledge", []byte{0x21}, false}},
		"q5_calibration": {{"knowledge", []byte{0x5a}, false}, {"knowledge", []byte{1}, false}},
		"pr37_support":   {{"knowledge", []byte{1}, false}, {"knowledge", []byte{2}, false}},
		"fx7_accounting": {{"knowledge", []byte{0x62}, false}, {"knowledge", []byte{0x6b}, false}, {"knowledge", []byte{0x35}, false}, {"knowledge", []byte{0x36}, false}, {"bank", balance, true}},
		"fx8_retained":   {{"zerone_gov", []byte{1}, false}, {"zerone_gov", []byte{6}, false}, {"zerone_gov", []byte{8}, true}, {"zerone_gov", []byte{0x0e}, false}, {"zerone_gov", []byte{0x0f}, false}, {"zerone_gov", []byte{0x14}, false}},
	}, nil
}

// CensusConsensusRepairsV2 is for a stopped source-state copy, before repaired
// export. It retains only declared inputs under independent 100k/64MiB budgets.
// Full-state hashing is streamed and is not bounded by the selected-row limits.
// Held means unchanged history/input, not entitlement or permission to redistribute.
func (app *ZeroneApp) CensusConsensusRepairsV2(ctx sdk.Context) (ConsensusRepairsCensus, error) {
	report, _, err := app.censusConsensusRepairsV2(ctx)
	return report, err
}

func consensusRepairsStrictJSON(raw []byte, value any) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("null record")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing record JSON: %v", err)
	}
	return nil
}

func consensusRepairsCanonicalProto(raw []byte, value proto.Message) error {
	if err := proto.Unmarshal(raw, value); err != nil {
		return err
	}
	canonical, err := (proto.MarshalOptions{Deterministic: true}).Marshal(value)
	if err != nil {
		return err
	}
	if len(value.ProtoReflect().GetUnknown()) != 0 || !bytes.Equal(raw, canonical) {
		return fmt.Errorf("noncanonical or unknown protobuf fields")
	}
	return nil
}

func (app *ZeroneApp) censusConsensusRepairsV2(ctx sdk.Context) (report ConsensusRepairsCensus, recalibrated []*knowledge.AgentCalibration, err error) {
	report.Height = ctx.BlockHeight()
	report.Findings = []ConsensusRepairsFinding{}
	if report.Height < 0 {
		return report, nil, fmt.Errorf("consensus repairs v2 census requires nonnegative height")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("consensus repairs v2 census read fault: %v", r)
			recalibrated = nil
		}
	}()
	// Discard even accidentally writing future helpers, including their events.
	ctx, _ = ctx.CacheContext()
	reads, err := consensusRepairsV2Reads()
	if err != nil {
		return report, nil, err
	}
	rows, err := app.consensusRepairsCensusRowsFor(ctx, &report, consensusRepairsV2CensusNames, reads)
	if err != nil {
		return report, nil, err
	}
	name := ""
	add := func(store, key, outcome, reason string) {
		report.Findings = append(report.Findings, ConsensusRepairsFinding{name, store, hex.EncodeToString([]byte(key)), outcome, reason})
	}
	scan := func(store string, prefix byte, inspect func(string, []byte) error) {
		var keys []string
		for key := range rows[store] {
			if len(key) > 0 && key[0] == prefix {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			if e := inspect(key, rows[store][key]); e != nil {
				add(store, key, "blocker", e.Error())
			}
		}
	}
	identity := func(key string, prefix byte, id string) error {
		if id == "" || key != string([]byte{prefix})+id {
			return fmt.Errorf("record key/identity mismatch")
		}
		return nil
	}
	for i, loc := range consensusRepairsV2MarkerLocations() {
		store := ctx.KVStore(app.keys[loc.store])
		if store.Has(loc.key) {
			add(loc.store, string(loc.key), "blocker", "v2 marker already present (enabled or malformed)")
		}
		if _, e := app.ConsensusRepairsV2Markers()[i].Enabled(ctx); e != nil {
			add(loc.store, string(loc.key), "blocker", e.Error())
		}
	}
	name = "q1_health"
	registered, active := 0, 0
	scan("zerone_staking", 1, func(key string, value []byte) error {
		registered++ // Occupied malformed keys are NOT evidence of no registry.
		var v staking.Validator
		if e := consensusRepairsStrictJSON(value, &v); e != nil {
			return e
		}
		if e := identity(key, 1, v.OperatorAddress); e != nil {
			return e
		}
		if _, e := consensusRepairsAmount(v.TotalStake); e != nil {
			return e
		}
		if v.IsActive {
			active++
		}
		return nil
	})
	for _, entry := range []struct {
		key    string
		target any
	}{{"\x01", new(alignment.Params)}, {"\x02", new(alignment.AlignmentState)}} {
		if value, ok := rows["alignment"][entry.key]; ok {
			if e := consensusRepairsStrictJSON(value, entry.target); e != nil {
				add("alignment", entry.key, "blocker", e.Error())
			}
		}
	}
	add("alignment", "", "held", fmt.Sprintf("registered=%d active=%d; empty registry selects not_observed at next scheduled observation; stored categories and observation history retained", registered, active))

	name = "q2_rounds"
	claims := map[string]*knowledge.Claim{}
	scan("knowledge", 2, func(key string, value []byte) error {
		var c knowledge.Claim
		if e := proto.Unmarshal(value, &c); e != nil {
			return e
		}
		if e := identity(key, 2, c.Id); e != nil {
			return e
		}
		claims[c.Id] = &c
		return nil
	})
	if raw, ok := rows["knowledge"]["\x06"]; ok {
		if e := proto.Unmarshal(raw, new(knowledge.Params)); e != nil {
			add("knowledge", "\x06", "blocker", e.Error())
		}
	}
	rounds := map[string]*knowledge.VerificationRound{}
	scan("knowledge", 3, func(key string, value []byte) error {
		var r knowledge.VerificationRound
		if e := proto.Unmarshal(value, &r); e != nil {
			return e
		}
		if e := identity(key, 3, r.Id); e != nil {
			return e
		}
		if e := knowledge.ValidateVerificationRoundRecord(&r, true); e != nil {
			return e
		}
		rounds[r.Id] = &r
		revealed := map[string]bool{}
		for _, reveal := range r.Reveals {
			if reveal != nil {
				revealed[reveal.Verifier] = true
			}
		}
		missed := 0
		for _, commit := range r.Commits {
			if commit != nil && !revealed[commit.Verifier] {
				missed++
			}
		}
		add("knowledge", key, "held", fmt.Sprintf("policy=%d phase=%s verdict=%s unmatched_commits=%d parent_present=%t; retained history, no replayed slash/reputation or inferred debt", r.ReviewPolicyVersion, r.Phase, r.Verdict, missed, claims[r.ClaimId] != nil))
		return nil
	})
	scan("knowledge", 0x21, func(key string, value []byte) error {
		if !bytes.Equal(value, []byte{1}) || rounds[key[1:]] == nil {
			return fmt.Errorf("malformed/dangling active-round index")
		}
		return nil
	})

	name = "q5_calibration"
	scan("knowledge", 0x5a, func(key string, value []byte) error {
		var c knowledge.AgentCalibration
		if e := consensusRepairsCanonicalProto(value, &c); e != nil {
			return e
		}
		if e := identity(key, 0x5a, c.Address); e != nil {
			return e
		}
		old := c.CalibrationScoreBps
		c.CalibrationScoreBps = knowledgekeeper.ComputeSmoothedAgentCalibrationScore(&c)
		add("knowledge", key, "held", fmt.Sprintf("stored_score=%d smoothed_score=%d submissions=%d genuine_zero=%t; score-only recompute, counters retained", old, c.CalibrationScoreBps, c.TotalSubmissions, c.TotalSubmissions > 0 && c.CalibrationScoreBps == 0))
		if old != c.CalibrationScoreBps {
			recalibrated = append(recalibrated, &c)
		}
		return nil
	})
	facts := map[string]*knowledge.Fact{}
	legacyZero, presentZero := 0, 0
	scan("knowledge", 1, func(key string, value []byte) error {
		var f knowledge.Fact
		if e := proto.Unmarshal(value, &f); e != nil {
			return e
		}
		if e := identity(key, 1, f.Id); e != nil {
			return e
		}
		facts[f.Id] = &f
		if f.SubmitterCalibrationSnapshotBps == 0 {
			if f.HasCalibrationSnapshot {
				presentZero++
			} else {
				legacyZero++
			}
		}
		return nil
	})
	add("knowledge", "", "held", fmt.Sprintf("retained fact snapshots: legacy_or_unknown_zero=%d present_zero=%d; no historical resnapshot", legacyZero, presentZero))
	name = "pr37_support"
	zeroFacts := 0
	for _, f := range facts {
		if f.Confidence == 0 {
			zeroFacts++
		}
	}
	add("knowledge", "", "held", fmt.Sprintf("zero_confidence_facts=%d; no historical confidence/floor rewrite; zero alone proves neither missing nor resolved support", zeroFacts))
	// Iterate in primary-key order; diagnose citation inputs without predicting
	// future round verdicts, domain-dependent growth, or entitlement.
	scan("knowledge", 2, func(key string, _ []byte) error {
		c := claims[key[1:]]
		if c == nil {
			return fmt.Errorf("claim unavailable after raw decode")
		}
		edges := map[string]uint64{}
		for _, ref := range c.References {
			if ref != "" {
				edges[ref] = 1_000_000
			}
		}
		contradictions := 0
		for _, rel := range c.Relations {
			if rel == nil {
				return fmt.Errorf("nil claim relation")
			}
			if rel.Relation == knowledge.RelationType_RELATION_TYPE_CONTRADICTS {
				contradictions++
				continue
			}
			if rel.TargetFactId == "" {
				continue
			}
			strength := rel.InferenceStrengthBps
			if strength == 0 || strength > 1_000_000 {
				strength = 1_000_000
			}
			if strength > edges[rel.TargetFactId] {
				edges[rel.TargetFactId] = strength
			}
		}
		missing, zero := 0, 0
		for id, strength := range edges {
			f := facts[id]
			if f == nil {
				missing++
				continue
			}
			eff := f.Confidence
			if f.DependencyConfidenceFloor > 0 && f.DependencyConfidenceFloor < eff {
				eff = f.DependencyConfidenceFloor
			}
			v := new(big.Int).Mul(new(big.Int).SetUint64(eff), new(big.Int).SetUint64(strength))
			v.Quo(v, big.NewInt(1_000_000))
			if v.Sign() == 0 {
				zero++
			}
		}
		if missing+zero+contradictions > 0 {
			add("knowledge", key, "held", fmt.Sprintf("resolved_zero_support=%d missing_support=%d contradiction_edges=%d; prospective only", zero, missing, contradictions))
		}
		return nil
	})

	name = "fx7_accounting"
	kg := &knowledge.GenesisState{AccountingState: &knowledge.KnowledgeGenesisAccounting{}}
	scan("knowledge", 0x62, func(key string, value []byte) error {
		var b knowledge.AugmentationBounty
		if e := consensusRepairsCanonicalProto(value, &b); e != nil {
			return e
		}
		if e := identity(key, 0x62, b.Id); e != nil {
			return e
		}
		kg.AugmentationBounties = append(kg.AugmentationBounties, &b)
		return nil
	})
	balance := app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(knowledge.TrainingFundModuleName), "uzrn").Amount.BigInt()
	kg.TrainingFundAllocation = balance.String()
	nonEscrow := new(big.Int).Set(balance)
	scan("knowledge", 0x6b, func(key string, value []byte) error {
		amount, e := consensusRepairsAmount(string(value))
		if e != nil {
			return e
		}
		if amount.String() != string(value) {
			return fmt.Errorf("noncanonical accounting escrow amount")
		}
		nonEscrow.Sub(nonEscrow, amount)
		kg.AccountingState.AugmentationEscrows = append(kg.AccountingState.AugmentationEscrows, &knowledge.AugmentationEscrowBalance{BountyId: key[1:], RemainingUzrn: string(value)})
		return nil
	})
	scan("knowledge", 0x35, func(key string, value []byte) error {
		if len(value) != 8 {
			return fmt.Errorf("bootstrap address counter must have exactly eight bytes")
		}
		kg.AccountingState.BootstrapAddressCounts = append(kg.AccountingState.BootstrapAddressCounts, &knowledge.BootstrapAddressCount{Address: key[1:], Count: binary.BigEndian.Uint64(value)})
		return nil
	})
	scan("knowledge", 0x36, func(key string, value []byte) error {
		if len(key) != 9 || len(value) != 8 {
			return fmt.Errorf("bootstrap epoch counter must have exactly eight key/value bytes")
		}
		kg.AccountingState.BootstrapEpochCounts = append(kg.AccountingState.BootstrapEpochCounts, &knowledge.BootstrapEpochCount{Epoch: binary.BigEndian.Uint64([]byte(key)[1:]), Count: binary.BigEndian.Uint64(value)})
		return nil
	})
	kg.AccountingState.TrainingFundNonEscrowBalance = nonEscrow.String()
	if e := knowledge.ValidateGenesisAccounting(kg); e != nil {
		add("knowledge", "", "blocker", e.Error())
	}
	add("knowledge", "", "held", "FX7 inventory retained; no reconstructed escrow, counter normalization or proof against historical export loss")

	name = "fx8_retained"
	gg := &gov.GenesisState{}
	scan("zerone_gov", 1, func(key string, value []byte) error {
		var lip gov.LIP
		if e := consensusRepairsStrictJSON(value, &lip); e != nil {
			return e
		}
		if e := identity(key, 1, lip.Id); e != nil {
			return e
		}
		gg.Lips = append(gg.Lips, &lip)
		return nil
	})
	scan("zerone_gov", 6, func(key string, value []byte) error {
		var p gov.ResearchSpendProposal
		if e := consensusRepairsStrictJSON(value, &p); e != nil {
			return e
		}
		if key != string(gov.ResearchSpendKey(p.ProposalId)) {
			return fmt.Errorf("research proposal key/identity mismatch")
		}
		gg.ResearchSpendProposals = append(gg.ResearchSpendProposals, &p)
		return nil
	})
	if value, ok := rows["zerone_gov"]["\x08"]; ok {
		if len(value) != 8 {
			add("zerone_gov", "\x08", "blocker", "research counter must have exactly eight bytes")
		} else {
			next := binary.BigEndian.Uint64(value)
			gg.NextResearchSpendNumber = &next
		}
	}
	scan("zerone_gov", 0x0f, func(key string, value []byte) error {
		if len(key) <= 10 || key[9] != 0 {
			return fmt.Errorf("malformed research community vote key")
		}
		gg.ResearchCommunityVotes = append(gg.ResearchCommunityVotes, &gov.GenesisResearchCommunityVote{ProposalId: binary.BigEndian.Uint64([]byte(key)[1:9]), Voter: key[10:], Vote: string(value)})
		return nil
	})
	scan("zerone_gov", 0x0e, func(key string, value []byte) error {
		if len(key) <= 1 || len(value) != 8 {
			return fmt.Errorf("malformed distinct voter key/height")
		}
		gg.DistinctVoters = append(gg.DistinctVoters, &gov.GenesisDistinctVoter{Voter: key[1:], FirstVotedAtBlock: binary.BigEndian.Uint64(value)})
		return nil
	})
	scan("zerone_gov", 0x14, func(key string, value []byte) error {
		var p gov.PhaseTransitionProposal
		if e := consensusRepairsStrictJSON(value, &p); e != nil {
			return e
		}
		if e := identity(key, 0x14, p.LipID); e != nil {
			return e
		}
		gg.PhaseTransitions = append(gg.PhaseTransitions, &gov.GenesisPhaseTransition{LipId: p.LipID, TargetPhase: p.TargetPhase, ConditionsSnapshot: p.ConditionsSnapshot, Stage: p.Stage, ActivationBlock: p.ActivationBlock, IsRollback: p.IsRollback, CancelReason: p.CancelReason})
		return nil
	})
	if e := gg.ValidateRetainedRecords(); e != nil {
		add("zerone_gov", "", "blocker", e.Error())
	}
	add("zerone_gov", "", "held", "FX8 retained proposals, votes, counters and phase history; no replay of current eligibility or governance execution")
	sort.SliceStable(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		if a.Census != b.Census {
			return a.Census < b.Census
		}
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		return a.Reason < b.Reason
	})
	for _, f := range report.Findings {
		if f.Outcome == "blocker" {
			return report, nil, fmt.Errorf("consensus repairs v2 census blocker: %s/%s/%s: %s", f.Census, f.Module, f.Key, f.Reason)
		}
	}
	return report, recalibrated, nil
}
