//go:build tok_learning

package cross_stack_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	kt "github.com/zerone-chain/zerone/x/knowledge/types"
	ontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
)

var (
	learningRoot         = flag.String("learning-root", "", "trusted source root containing the fixed local Fold-to-Fire worker")
	learningOut          = flag.String("learning-out", "", "explicit fresh evidence directory; default is temporary")
	learningReplay       = flag.String("learning-replay", "", "read-only evidence directory to verify and computationally replay")
	learningExpectedHead = flag.String("learning-expected-head", "", "optional previously pinned lowercase SHA256 run head")
)

const learningDomain = "tok_fold_learning_lab"

// Local adapters only. None of these types extends a public or consensus schema.
// Full Fact JSON remains available identically to every view, not only the graph.
type learningTask struct {
	N int    `json:"n"`
	Q string `json:"q"`
}
type learningFact struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Status     string            `json:"status"`
	Revision   string            `json:"revision"`
	Provenance map[string]string `json:"provenance"`
}
type learningEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}
type learningHistory struct {
	OldID   string `json:"oldId"`
	NewID   string `json:"newId"`
	Outcome string `json:"outcome"`
}
type learningSnapshot struct {
	Tasks           []learningTask    `json:"tasks"`
	Budget          int               `json:"budget"`
	Facts           []learningFact    `json:"facts"`
	Relations       []learningEdge    `json:"relations"`
	History         []learningHistory `json:"history"`
	HistoryComplete bool              `json:"historyComplete"`
}
type learningMaterial struct {
	Tasks       []learningTask `json:"tasks"`
	Polynomials []struct {
		N       int    `json:"n"`
		Content string `json:"content"`
	} `json:"polynomials"`
	Fault struct {
		Content string `json:"content"`
	} `json:"fault"`
}
type learningUsed struct {
	ID            string `json:"id"`
	ContentDigest string `json:"contentDigest"`
	FactDigest    string `json:"factDigest"`
	Revision      string `json:"revision"`
}
type learningOutput struct {
	Task     learningTask    `json:"task"`
	Decision string          `json:"decision"`
	Reason   string          `json:"reason"`
	Value    json.RawMessage `json:"value"`
	Used     []learningUsed  `json:"used"`
}
type learningUse struct {
	Outputs             []learningOutput `json:"outputs"`
	CommonPayloadDigest string           `json:"commonPayloadDigest"`
	Costs               struct {
		Enumerations    int `json:"enumerations"`
		ReferenceChecks int `json:"referenceChecks"`
	} `json:"costs"`
}
type learningCheck struct {
	Metrics map[string]int `json:"metrics"`
	Costs   struct {
		ReferenceChecks int `json:"referenceChecks"`
	} `json:"costs"`
}
type learningEvaluation struct {
	Name     string                     `json:"name"`
	Snapshot learningSnapshot           `json:"snapshot"`
	Uses     map[string]json.RawMessage `json:"uses"`
	Checks   map[string]json.RawMessage `json:"checks"`
}
type learningRoundRecord struct {
	Branch string                `json:"branch"`
	Claim  *kt.Claim             `json:"claim"`
	Round  *kt.VerificationRound `json:"round"`
	Fact   *kt.Fact              `json:"fact"`
	Events sdk.Events            `json:"events"`
}
type learningBundle struct {
	ActualHeight    string        `json:"actual_height"`
	RequestedHeight string        `json:"requested_height"`
	Bundle          *kt.ToKBundle `json:"bundle"`
	FullSHA256      string        `json:"full_payload_and_metadata_sha256"`
}

func learningJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	obj, err := tokLearningReceiptObjectFor(v)
	require.NoError(t, err)
	return obj.data
}
func learningSHA(t *testing.T, v any) string {
	t.Helper()
	obj, err := tokLearningReceiptObjectFor(v)
	require.NoError(t, err)
	return obj.digest
}
func learningDecode(t *testing.T, raw []byte, dst any) {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	require.NoError(t, d.Decode(dst))
	require.Equal(t, io.EOF, d.Decode(new(any)))
}
func learningClone(t *testing.T, s learningSnapshot) learningSnapshot {
	t.Helper()
	var out learningSnapshot
	learningDecode(t, learningJSON(t, s), &out)
	return out
}

// A writer limit stops collection rather than buffering an unbounded child.
// CommandContext also kills the direct Node process after ten seconds. Requests
// never choose a path, module, command, environment variable, or executable.
type learningLimitedBuffer struct {
	buffer bytes.Buffer // named: do not expose Buffer.ReadFrom's unbounded fast path
	limit  int
}

func (b *learningLimitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		return 0, fmt.Errorf("local worker output limit")
	}
	return b.buffer.Write(p)
}

func TestToKLearningWorkerBounds(t *testing.T) {
	b := &learningLimitedBuffer{limit: 8}
	n, err := io.Copy(b, strings.NewReader("12345678"))
	require.NoError(t, err)
	require.Equal(t, int64(8), n)
	n, err = io.Copy(b, strings.NewReader("9"))
	require.Error(t, err)
	require.Zero(t, n)
	require.Equal(t, "12345678", b.buffer.String())
}

type learningWorker struct{ root, node string }

func (w learningWorker) call(t *testing.T, request any) json.RawMessage {
	t.Helper()
	input := learningJSON(t, request)
	require.LessOrEqual(t, len(input), 262144)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, w.node, filepath.Join(w.root, "tools/fold-to-fire/learning-loop.mjs"))
	cmd.Dir = w.root
	// In particular, do not inherit NODE_OPTIONS, preload hooks or credentials.
	cmd.Env = []string{"LANG=C", "TZ=UTC"}
	cmd.Stdin = bytes.NewReader(input)
	stdout := &learningLimitedBuffer{limit: 1 << 20}
	stderr := &learningLimitedBuffer{limit: 4096}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	require.NoError(t, cmd.Run(), "worker: %s %s", stdout.buffer.String(), stderr.buffer.String())
	require.Empty(t, stderr.buffer.String())
	return learningJSON(t, json.RawMessage(bytes.TrimSpace(stdout.buffer.Bytes())))
}
func (w learningWorker) use(t *testing.T, s learningSnapshot) map[string]json.RawMessage {
	t.Helper()
	uses := map[string]json.RawMessage{}
	for _, view := range []string{"node", "graph", "rows"} {
		var req map[string]any
		learningDecode(t, learningJSON(t, s), &req)
		req["op"], req["view"] = "consume", view
		uses[view] = w.call(t, req)
		var use learningUse
		learningDecode(t, uses[view], &use)
		require.Zero(t, use.Costs.ReferenceChecks, "use must precede and not invoke the cross-check")
	}
	var node, graph, rows learningUse
	learningDecode(t, uses["node"], &node)
	learningDecode(t, uses["graph"], &graph)
	learningDecode(t, uses["rows"], &rows)
	require.Equal(t, graph.Outputs, rows.Outputs, "equal-information relational control")
	require.Equal(t, node.CommonPayloadDigest, graph.CommonPayloadDigest)
	require.Equal(t, graph.CommonPayloadDigest, rows.CommonPayloadDigest)
	return uses
}
func (w learningWorker) check(t *testing.T, s learningSnapshot, uses map[string]json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	checks := map[string]json.RawMessage{}
	for _, view := range []string{"node", "graph", "rows"} {
		var use learningUse
		learningDecode(t, uses[view], &use)
		checks[view] = w.call(t, map[string]any{"op": "check", "tasks": s.Tasks, "outputs": use.Outputs})
	}
	return checks
}
func (w learningWorker) evaluate(t *testing.T, name string, s learningSnapshot) learningEvaluation {
	t.Helper()
	uses := w.use(t, s)
	return learningEvaluation{name, s, uses, w.check(t, s, uses)}
}
func learningMetrics(t *testing.T, checks map[string]json.RawMessage, view string) map[string]int {
	t.Helper()
	var check learningCheck
	learningDecode(t, checks[view], &check)
	return check.Metrics
}

// Follow full_loop_test.go, not the synthetic SetFact/CompleteRound shortcuts in
// the selector tests. Only quorum/deadlines are fixture overrides. Fee, balance,
// confidence-scaled challenge stake, cooldown, and verdict rules remain active.
type learningLab struct {
	h      *TestHarness
	ms     kt.MsgServer
	actors []sdk.AccAddress
}

func newLearningLab(t *testing.T) *learningLab {
	h := NewTestHarness(t)
	h.Ctx = h.Ctx.WithBlockHeight(1000).WithBlockTime(time.Unix(1700000000, 0).UTC()).WithEventManager(sdk.NewEventManager())
	params, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	params.MinVerifiers = 1
	params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 1
	require.NoError(t, h.KnowledgeKeeper.SetParams(h.Ctx, params))
	require.Equal(t, uint32(2), h.KnowledgeKeeper.GetEffectiveMinVerifiers(h.Ctx, learningDomain))
	require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &kt.Domain{Name: learningDomain, Status: kt.DomainStatus_DOMAIN_STATUS_ACTIVE}))
	h.App.ZeroneOntologyKeeper.SetDomain(h.Ctx, &ontologytypes.Domain{Name: learningDomain, Status: "active", Stratum: uint32(ontologytypes.StratumEmpirical), Depth: 1})
	lab := &learningLab{h: h, ms: knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)}
	for _, id := range []string{"tok-lab-supplier-0001", "tok-lab-verifier-0001", "tok-lab-verifier-0002", "tok-lab-challenger-01", "tok-lab-corrector-001"} {
		addr := sdk.AccAddress([]byte(id))
		lab.actors = append(lab.actors, addr)
		require.NoError(t, h.FundAccount(addr, sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewInt(5_000_000_000)))))
	}
	return lab
}
func (l *learningLab) height(height uint64) {
	l.h.Ctx = l.h.Ctx.WithBlockHeight(int64(height)).WithBlockTime(time.Unix(1700000000+int64(height)-1000, 0).UTC())
}
func (l *learningLab) balances() map[string]string {
	b := map[string]string{}
	for _, a := range l.actors {
		b[a.String()] = l.h.GetBalance(a, "uzrn").Amount.String()
	}
	return b
}
func (l *learningLab) finish(t *testing.T, roundID, vote, branch string, eventStart int) learningRoundRecord {
	t.Helper()
	h := l.h
	round, ok := h.KnowledgeKeeper.GetVerificationRound(h.Ctx, roundID)
	require.True(t, ok)
	require.Equal(t, kt.VerificationPhase_VERIFICATION_PHASE_COMMIT, round.Phase)
	for i, a := range l.actors[1:3] {
		salt := []byte(fmt.Sprintf("tok-learning-fixed-salt-%d", i))
		_, err := l.ms.SubmitCommitment(h.Ctx, &kt.MsgSubmitCommitment{Verifier: a.String(), RoundId: roundID, CommitHash: kt.ComputeCommitmentHash(roundID, vote, 0, salt)})
		require.NoError(t, err)
	}
	l.height(round.CommitDeadline + 1)
	require.NoError(t, h.KnowledgeKeeper.AdvanceRoundPhases(h.Ctx))
	round, ok = h.KnowledgeKeeper.GetVerificationRound(h.Ctx, roundID)
	require.True(t, ok)
	require.Equal(t, kt.VerificationPhase_VERIFICATION_PHASE_REVEAL, round.Phase)
	for i, a := range l.actors[1:3] {
		_, err := l.ms.SubmitReveal(h.Ctx, &kt.MsgSubmitReveal{Verifier: a.String(), RoundId: roundID, Vote: vote, Confidence: 0, Salt: []byte(fmt.Sprintf("tok-learning-fixed-salt-%d", i))})
		require.NoError(t, err)
	}
	l.height(round.RevealDeadline + 1)
	require.NoError(t, h.KnowledgeKeeper.AdvanceRoundPhases(h.Ctx))
	round, ok = h.KnowledgeKeeper.GetVerificationRound(h.Ctx, roundID)
	require.True(t, ok)
	require.Equal(t, kt.VerificationPhase_VERIFICATION_PHASE_COMPLETE, round.Phase)
	require.Len(t, round.Commits, 2)
	require.Len(t, round.Reveals, 2)
	claim, ok := h.KnowledgeKeeper.GetClaim(h.Ctx, round.ClaimId)
	require.True(t, ok)
	var fact *kt.Fact
	if vote == "accept" {
		require.Equal(t, kt.Verdict_VERDICT_ACCEPT, round.Verdict)
		require.Equal(t, kt.ClaimStatus_CLAIM_STATUS_ACCEPTED, claim.Status)
		fact, ok = h.KnowledgeKeeper.GetFact(h.Ctx, knowledgekeeper.GenerateFactID(claim.Id, uint64(h.Ctx.BlockHeight())))
		require.True(t, ok)
		require.Equal(t, claim.FactContent, fact.Content)
	} else {
		require.Equal(t, kt.Verdict_VERDICT_REJECT, round.Verdict)
		require.Equal(t, kt.ClaimStatus_CLAIM_STATUS_REJECTED, claim.Status)
		_, ok = h.KnowledgeKeeper.GetFact(h.Ctx, knowledgekeeper.GenerateFactID(claim.Id, uint64(h.Ctx.BlockHeight())))
		require.False(t, ok, "rejected replacement must not materialize a fact")
	}
	return learningRoundRecord{branch, claim, round, fact, h.Ctx.EventManager().Events()[eventStart:]}
}
func (l *learningLab) submit(t *testing.T, actor int, content string, relations []*kt.ClaimRelation, vote, branch string) learningRoundRecord {
	t.Helper()
	h := l.h
	// Wait out the real pacing-scaled cooldown; do not disable it.
	next := h.KnowledgeKeeper.GetLastClaimHeight(h.Ctx, l.actors[actor].String()) + h.KnowledgeKeeper.GetEffectiveCooldown(h.Ctx, learningDomain)
	if next > uint64(h.Ctx.BlockHeight()) {
		l.height(next)
	}
	start := len(h.Ctx.EventManager().Events())
	resp, err := l.ms.SubmitClaim(h.Ctx, &kt.MsgSubmitClaim{Submitter: l.actors[actor].String(), FactContent: content, Domain: learningDomain, Category: "empirical", Stake: h.KnowledgeKeeper.GetEffectiveMinReviewFee(h.Ctx), Relations: relations})
	require.NoError(t, err)
	claim, ok := h.KnowledgeKeeper.GetClaim(h.Ctx, resp.ClaimId)
	require.True(t, ok)
	return l.finish(t, claim.VerificationRoundId, vote, branch, start)
}
func learningRelation(target string, kind kt.RelationType) []*kt.ClaimRelation {
	return []*kt.ClaimRelation{{TargetFactId: target, Relation: kind, Inference: kt.InferenceType_INFERENCE_TYPE_DEDUCTIVE, InferenceStrengthBps: 1_000_000}}
}
func learningProject(t *testing.T, fact *kt.Fact, height uint64) learningFact {
	t.Helper()
	return learningFact{fact.Id, fact.Content, strings.TrimPrefix(fact.Status.String(), "FACT_STATUS_"), strconv.FormatUint(height, 10), map[string]string{"chainFactJSON": string(learningJSON(t, fact)), "origin": "handler-produced-fact"}}
}
func (l *learningLab) bundle(t *testing.T, id string, cascade bool) learningBundle {
	t.Helper()
	sel := &kt.ToKSelector{Variant: &kt.ToKSelector_RootedSubtree{RootedSubtree: &kt.RootedSubtreeSelector{RootFactId: id, MaxDepth: 1}}}
	if cascade {
		sel = &kt.ToKSelector{Variant: &kt.ToKSelector_CascadeReplay{CascadeReplay: &kt.CascadeReplaySelector{DisprovenFactId: id, MaxDepth: 1, IncludeStatusHistory: true, IncludeSupersessions: true}}}
	}
	b, err := l.h.KnowledgeKeeper.AssembleToKBundle(l.h.Ctx, sel, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(l.h.Ctx.BlockHeight()), b.SnapshotBlock)
	require.LessOrEqual(t, len(b.Nodes), 4)
	if cascade {
		require.Equal(t, "v2", b.Provenance.TokRootVersion)
		require.Equal(t, knowledgekeeper.ComputeToKSnapshotRootV2(b.IncludedNodeIds, b.IncludedEdges, b.CascadeEvents, b.Vindications, b.StatusHistory), b.SnapshotRoot)
	} else {
		require.Equal(t, "v1", b.Provenance.TokRootVersion)
		require.Equal(t, knowledgekeeper.ComputeToKSnapshotRoot(b.IncludedNodeIds, b.IncludedEdges), b.SnapshotRoot)
	}
	return learningBundle{strconv.FormatUint(b.SnapshotBlock, 10), "0", b, learningSHA(t, b)}
}
func learningSnapshotFromBundles(t *testing.T, tasks []learningTask, bundles []learningBundle) learningSnapshot {
	t.Helper()
	s := learningSnapshot{Tasks: tasks, Budget: 9, Facts: []learningFact{}, Relations: []learningEdge{}, History: []learningHistory{}, HistoryComplete: true}
	for _, exported := range bundles {
		b := exported.Bundle
		for _, f := range b.Nodes {
			s.Facts = append(s.Facts, learningProject(t, f, b.SnapshotBlock))
		}
		for _, edge := range b.IncludedEdges {
			if edge.Relation == kt.RelationType_RELATION_TYPE_REQUIRES.String() {
				s.Relations = append(s.Relations, learningEdge{edge.FromFactId, edge.ToFactId, "REQUIRES"})
			}
		}
	}
	sort.Slice(s.Facts, func(i, j int) bool { return s.Facts[i].ID < s.Facts[j].ID })
	sort.Slice(s.Relations, func(i, j int) bool { return s.Relations[i].Source < s.Relations[j].Source })
	return s
}
func learningN(t *testing.T, s learningSnapshot, n int) learningSnapshot {
	t.Helper()
	s = learningClone(t, s)
	tasks := []learningTask{}
	facts := []learningFact{}
	ids := map[string]bool{}
	for _, task := range s.Tasks {
		if task.N == n {
			tasks = append(tasks, task)
		}
	}
	for _, fact := range s.Facts {
		var p struct {
			N int `json:"n"`
		}
		learningDecode(t, []byte(fact.Content), &p)
		if p.N == n {
			facts = append(facts, fact)
			ids[fact.ID] = true
		}
	}
	edges := []learningEdge{}
	history := []learningHistory{}
	for _, e := range s.Relations {
		if ids[e.Source] && ids[e.Target] {
			edges = append(edges, e)
		}
	}
	for _, e := range s.History {
		if ids[e.OldID] || ids[e.NewID] {
			history = append(history, e)
		}
	}
	s.Tasks, s.Facts, s.Relations, s.History = tasks, facts, edges, history
	return s
}

func learningBindings(t *testing.T, w learningWorker) (any, any, any) {
	t.Helper()
	tasks := []learningTask{}
	for _, n := range []int{3, 5, 7} {
		for _, q := range []string{"1", "3/2", "2"} {
			tasks = append(tasks, learningTask{n, q})
		}
	}
	policy := map[string]any{
		"profile": "tok-fold-learning-lab/0", "scope": "exact finite square-lattice model; no physical or biological generalization",
		"budget": 9, "budget_unit": "one production enumeration per task; reference checks accounted separately",
		"views": []string{"node", "graph", "rows"}, "unresolved": "ABSTAIN", "graph_rows": "equal information; semantic equality required, advantage not required",
		"history_complete_scope": "bounded scenario only; not global history completeness", "at_block_height": "0",
		"root_semantics": "v1 topology only; cascade v2 unchanged; full payload plus metadata committed separately",
		"effects":        tokLearningZeroEffects(), "shared_control": true,
		"limits": []string{"local receipts do not repair consensus RateFact or query counters", "scripted actors are not independent witnesses", "no PoCA or branchflow eligibility inference", "test-only uzrn balances and internal bookkeeping change", "challenge EvidenceIds are retained locally, not persisted by production ChallengeFact"},
	}
	// Source identities are relative paths plus content hashes, never local paths,
	// wall clocks, temporary homes, genesis random keys, or build-output names.
	paths := []string{"go.mod", "go.sum", "tests/cross_stack/harness_test.go", "tests/cross_stack/tok_learning_loop_test.go", "tests/cross_stack/tok_learning_receipts_test.go", "tests/cross_stack/tok_learning_receipts_open_unix_test.go", "tests/cross_stack/tok_learning_receipts_open_other_test.go", "tools/fold-to-fire/enumerate.mjs", "tools/fold-to-fire/reference.mjs", "tools/fold-to-fire/learning-loop.mjs", "tools/witness-v0/protocol/canonical.go"}
	// Bind the direct handler/selector method, not a claim of binary attestation.
	for _, name := range []string{"msg_server.go", "rounds.go", "phases.go", "confidence.go", "vindication.go", "pacing.go", "fire_activity.go", "tok_bundle.go", "tok_selector.go", "tok_cascade.go", "tok_serialise.go"} {
		paths = append(paths, "x/knowledge/keeper/"+name)
	}
	sort.Strings(paths)
	sources := []map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(w.root, filepath.FromSlash(path)))
		require.NoError(t, err)
		sum := sha256.Sum256(data)
		sources = append(sources, map[string]string{"path": path, "sha256": hex.EncodeToString(sum[:])})
	}
	method := map[string]any{"base_commit": "89553a0132dafa1ba9670f53b8a3195b33a6e720", "sources": sources,
		"source_scope":     "listed method files; base commit names the reviewed export, not a live binary attestation",
		"fixture":          map[string]any{"initial_height": "1000", "epoch_seconds": "1700000000", "seconds_per_height": "1", "min_verifiers": "1", "effective_verifiers": "2", "commit_blocks": "2", "reveal_blocks": "2", "aggregation_blocks": "1", "fund_each_uzrn": "5000000000", "domain": learningDomain, "actors": []string{"tok-lab-supplier-0001", "tok-lab-verifier-0001", "tok-lab-verifier-0002", "tok-lab-challenger-01", "tok-lab-corrector-001"}},
		"execution":        "real handlers and AdvanceRoundPhases; direct calls bypass signed transactions/ante; no server or full block-loop claim",
		"rejected_control": "discarded SDK cache branch with identical replacement content, handlers and two reject votes; not another live history",
		"cross_check":      "separate local post-scored DFS and rational Horner implementation; shared process operator"}
	return tasks, policy, method
}

func learningRun(t *testing.T, w learningWorker) *tokLearningReceipts {
	t.Helper()
	task, policy, method := learningBindings(t, w)
	run, err := newTokLearningReceipts(task, policy, method)
	require.NoError(t, err)
	l := newLearningLab(t)
	h := l.h
	initialBalances := l.balances()
	params, err := h.KnowledgeKeeper.GetParams(h.Ctx)
	require.NoError(t, err)
	materialRaw := w.call(t, map[string]any{"op": "fixture"})
	var material learningMaterial
	learningDecode(t, materialRaw, &material)
	require.Equal(t, learningJSON(t, task), learningJSON(t, material.Tasks))
	initial := []learningRoundRecord{}
	cacheCalls := []any{}
	roots := map[int]string{}
	cachedIDs := []string{}
	for _, polynomial := range material.Polynomials {
		content := polynomial.Content
		if polynomial.N == 5 {
			content = material.Fault.Content
		}
		record := l.submit(t, 0, content, nil, "accept", "main")
		initial = append(initial, record)
		roots[polynomial.N] = record.Fact.Id
		tasks := []learningTask{}
		for _, task := range material.Tasks {
			if task.N == polynomial.N {
				tasks = append(tasks, task)
			}
		}
		req := map[string]any{"op": "cached", "parent": learningProject(t, record.Fact, uint64(h.Ctx.BlockHeight())), "tasks": tasks}
		response := w.call(t, req)
		cacheCalls = append(cacheCalls, map[string]any{"request": req, "response": response})
		var cached struct {
			Cached []struct {
				Content string `json:"content"`
			} `json:"cached"`
		}
		learningDecode(t, response, &cached)
		for _, item := range cached.Cached {
			r := l.submit(t, 0, item.Content, learningRelation(record.Fact.Id, kt.RelationType_RELATION_TYPE_REQUIRES), "accept", "main")
			initial = append(initial, r)
			if polynomial.N == 5 {
				cachedIDs = append(cachedIDs, r.Fact.Id)
			}
		}
	}
	preBundles := []learningBundle{}
	for _, n := range []int{3, 5, 7} {
		b := l.bundle(t, roots[n], false)
		require.Len(t, b.Bundle.Nodes, 4)
		require.Len(t, b.Bundle.IncludedEdges, 3)
		preBundles = append(preBundles, b)
	}
	before := learningSnapshotFromBundles(t, material.Tasks, preBundles)
	appendStage := func(stage string, payload any) string {
		digest, e := run.Append(stage, payload, map[string]any{"actual_context_height": strconv.FormatInt(h.Ctx.BlockHeight(), 10), "branch": "main", "effects": tokLearningZeroEffects()})
		require.NoError(t, e)
		return digest
	}
	evidenceCheckpoint := appendStage("evidence", map[string]any{"material": materialRaw, "cache_calls": cacheCalls, "rounds": initial, "bundles": preBundles, "fixture_params": params, "initial_test_balances_uzrn": initialBalances})
	preUses := w.use(t, before)
	useCheckpoint := appendStage("use", map[string]any{"snapshot": before, "uses": preUses, "reference_checks_so_far": "0"})
	// No fixture label participates in consume. Only the retained, independently
	// computed *local* discrepancy below licenses this test's challenge action.
	preChecks := w.check(t, before, preUses)
	require.Equal(t, 2, learningMetrics(t, preChecks, "graph")["incorrectReuse"])
	require.Equal(t, 7, learningMetrics(t, preChecks, "graph")["exactAnswerAgreement"])
	discrepancy := map[string]any{"use_checkpoint": useCheckpoint, "snapshot_sha256": learningSHA(t, before), "outputs": preUses["graph"], "check": preChecks["graph"]}
	discrepancySHA := learningSHA(t, discrepancy)
	old, ok := h.KnowledgeKeeper.GetFact(h.Ctx, roots[5])
	require.True(t, ok)
	challenge := &kt.MsgChallengeFact{Challenger: l.actors[3].String(), FactId: old.Id, Reason: "Local exact activity discrepancy sha256=" + discrepancySHA, EvidenceIds: []string{"sha256:" + discrepancySHA}, Stake: knowledgekeeper.EffectiveMinChallengeStake(params, old.Confidence).String()}
	start := len(h.Ctx.EventManager().Events())
	challenged, err := l.ms.ChallengeFact(h.Ctx, challenge)
	require.NoError(t, err)
	challengeRecord := l.finish(t, challenged.RoundId, "accept", "main", start)
	require.Empty(t, challengeRecord.Claim.References, "production drops EvidenceIds; do not change citation semantics")
	require.Contains(t, challengeRecord.Claim.FactContent, discrepancySHA)
	old, ok = h.KnowledgeKeeper.GetFact(h.Ctx, roots[5])
	require.True(t, ok)
	require.Equal(t, kt.FactStatus_FACT_STATUS_DISPROVEN, old.Status)
	for _, id := range cachedIDs {
		f, found := h.KnowledgeKeeper.GetFact(h.Ctx, id)
		require.True(t, found)
		require.Equal(t, kt.FactStatus_FACT_STATUS_CONTESTED, f.Status)
	}
	feedbackCheckpoint := appendStage("feedback", map[string]any{"checks": preChecks, "discrepancy": discrepancy, "discrepancy_sha256": discrepancySHA, "submitted_challenge": challenge, "challenge_reason_sha256": learningSHA(t, challenge.Reason), "challenge_round": challengeRecord, "evidence_id_limit": "MsgChallengeFact.EvidenceIds not stored in Claim; retained here only"})
	cascade := l.bundle(t, old.Id, true)
	require.Len(t, cascade.Bundle.CascadeEvents, 3)
	require.NotEmpty(t, cascade.Bundle.StatusHistory)
	for _, event := range cascade.Bundle.CascadeEvents {
		require.Equal(t, challengeRecord.Claim.Id, event.ChallengeClaimId)
		require.Contains(t, cachedIDs, event.DescendantFactId)
	}
	currentBundles := []learningBundle{l.bundle(t, roots[3], false), l.bundle(t, roots[5], false), l.bundle(t, roots[7], false)}
	postChallenge := learningSnapshotFromBundles(t, material.Tasks, currentBundles)
	correctContent := ""
	for _, p := range material.Polynomials {
		if p.N == 5 {
			correctContent = p.Content
		}
	}
	// Actual rejected replacement in a disposable store branch. Never SetFact,
	// fake accepted result, restore the contested caches, or bypass a keeper rule.
	parentContext := h.Ctx
	branchContext, _ := h.Ctx.CacheContext()
	h.Ctx = branchContext
	rejected := l.submit(t, 4, correctContent, learningRelation(old.Id, kt.RelationType_RELATION_TYPE_SUPERSEDES), "reject", "discarded-negative-control")
	rejectedSnapshot := learningN(t, postChallenge, 5)
	candidate := learningFact{"claim:" + rejected.Claim.Id, rejected.Claim.FactContent, "REJECTED", strconv.FormatInt(h.Ctx.BlockHeight(), 10), map[string]string{"origin": "unmaterialized-rejected-claim; not a Fact", "claimJSON": string(learningJSON(t, rejected.Claim))}}
	rejectedSnapshot.Facts = append(rejectedSnapshot.Facts, candidate)
	rejectedSnapshot.History = append(rejectedSnapshot.History, learningHistory{old.Id, candidate.ID, "REJECTED"})
	negative := w.evaluate(t, "handler-rejected-replacement", rejectedSnapshot)
	for _, view := range []string{"node", "graph", "rows"} {
		require.Equal(t, 3, learningMetrics(t, negative.Checks, view)["abstentions"])
	}
	branchBalances := l.balances()
	h.Ctx = parentContext // deliberately do not call the cache's write closure
	_, found := h.KnowledgeKeeper.GetClaim(h.Ctx, rejected.Claim.Id)
	require.False(t, found, "negative branch must not leak into main")
	corrected := l.submit(t, 4, correctContent, learningRelation(old.Id, kt.RelationType_RELATION_TYPE_SUPERSEDES), "accept", "main")
	relations, err := h.KnowledgeKeeper.GetFactRelations(h.Ctx, corrected.Fact.Id)
	require.NoError(t, err)
	require.Len(t, relations, 1)
	require.Equal(t, kt.RelationType_RELATION_TYPE_SUPERSEDES, relations[0].Relation)
	require.Equal(t, old.Id, relations[0].TargetFactId)
	correctedBundle := l.bundle(t, corrected.Fact.Id, false)
	require.Len(t, correctedBundle.Bundle.Nodes, 1)
	require.Empty(t, correctedBundle.Bundle.IncludedEdges, "v1 support selector must not be made to include SUPERSEDES")
	cascadeAfter := l.bundle(t, old.Id, true)
	require.Contains(t, cascadeAfter.Bundle.SupersessionChain, corrected.Fact.Id)
	// Cascade selectors do not promise corrected content: export it separately.
	require.NotContains(t, cascadeAfter.Bundle.IncludedNodeIds, corrected.Fact.Id)
	afterBundles := []learningBundle{l.bundle(t, roots[3], false), l.bundle(t, roots[5], false), l.bundle(t, roots[7], false), correctedBundle}
	after := learningSnapshotFromBundles(t, material.Tasks, afterBundles)
	after.Relations = append(after.Relations, learningEdge{corrected.Fact.Id, old.Id, "SUPERSEDES"})
	after.History = append(after.History, learningHistory{old.Id, corrected.Fact.Id, "ACCEPTED"})
	for _, id := range cachedIDs {
		f, exists := h.KnowledgeKeeper.GetFact(h.Ctx, id)
		require.True(t, exists)
		require.Equal(t, kt.FactStatus_FACT_STATUS_CONTESTED, f.Status)
	}
	correctionCheckpoint := appendStage("correction", map[string]any{"cascade_before": cascade, "cascade_after": cascadeAfter, "corrected_content": correctedBundle, "bundles": afterBundles, "snapshot": after, "replacement": corrected, "replacement_relations": relations, "negative": negative, "rejected_round": rejected, "negative_branch_test_balances_uzrn": branchBalances, "history_projection": "derived from accepted replacement Claim, real SUPERSEDES relation and cascade records; bounded scenario, not global completeness"})
	postUses := w.use(t, after)
	postChecks := w.check(t, after, postUses)
	for _, view := range []string{"node", "graph", "rows"} {
		require.Equal(t, 9, learningMetrics(t, postChecks, view)["exactAnswerAgreement"])
		require.Zero(t, learningMetrics(t, postChecks, view)["incorrectReuse"])
	}
	var postGraph learningUse
	learningDecode(t, postUses["graph"], &postGraph)
	for _, output := range postGraph.Outputs {
		if output.Task.N == 5 {
			require.Len(t, output.Used, 1)
			require.Equal(t, corrected.Fact.Id, output.Used[0].ID)
		}
	}
	controls := learningControls(t, w, before, after)
	comparison := learningComparison(t, preChecks, postChecks)
	appendStage("rerun", map[string]any{"uses": postUses, "checks": postChecks, "controls": controls, "comparison": comparison})
	suppliedIDs := []string{}
	for _, record := range initial {
		suppliedIDs = append(suppliedIDs, record.Fact.Id)
	}
	sort.Strings(suppliedIDs)
	attribution := map[string]any{
		"effects": tokLearningZeroEffects(), "allocation_status": "NOT_EVALUATED", "shared_scripted_control": true,
		"supplied_evidence":    map[string]any{"actor": l.actors[0].String(), "checkpoint": evidenceCheckpoint, "fact_ids": suppliedIDs},
		"consumed_evidence":    map[string]any{"worker_source": "tools/fold-to-fire/learning-loop.mjs", "checkpoint": useCheckpoint, "before": learningConsumed(t, preUses["graph"]), "after": learningConsumed(t, postUses["graph"])},
		"detected_discrepancy": map[string]any{"actor": l.actors[3].String(), "checker_source": "tools/fold-to-fire/reference.mjs", "checkpoint": feedbackCheckpoint, "evidence_sha256": discrepancySHA},
		"supplied_correction":  map[string]any{"actor": l.actors[4].String(), "checkpoint": correctionCheckpoint, "claim_id": corrected.Claim.Id, "fact_id": corrected.Fact.Id},
		"test_balances_uzrn":   map[string]any{"initial": initialBalances, "final": l.balances()},
		"boundary":             "descriptive observed artifact roles, not contribution shares, identity/authenticity, independent witnesses, qualification, reward eligibility or consensus use accounting; memory-app fees/rewards and internal bookkeeping may change"}
	appendStage("attribution", attribution)
	head, err := run.Head()
	require.NoError(t, err)
	t.Logf("learning head=%s before=%s after=%s comparison=%s", head, learningJSON(t, learningMetrics(t, preChecks, "graph")), learningJSON(t, learningMetrics(t, postChecks, "graph")), learningJSON(t, comparison))
	return run
}

func learningConsumed(t *testing.T, raw json.RawMessage) []learningUsed {
	t.Helper()
	var use learningUse
	learningDecode(t, raw, &use)
	byID := map[string]learningUsed{}
	for _, out := range use.Outputs {
		for _, ref := range out.Used {
			byID[ref.ID] = ref
		}
	}
	refs := []learningUsed{}
	for _, ref := range byID {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	return refs
}
func learningComparison(t *testing.T, before, after map[string]json.RawMessage) any {
	t.Helper()
	diff := func(a, b map[string]int) map[string]string {
		out := map[string]string{}
		for key, value := range a {
			out[key] = strconv.Itoa(value - b[key])
		}
		return out
	}
	change := map[string]any{}
	advantage := map[string]any{}
	for _, view := range []string{"node", "graph", "rows"} {
		change[view] = diff(learningMetrics(t, after, view), learningMetrics(t, before, view))
	}
	advantage["before"] = diff(learningMetrics(t, before, "graph"), learningMetrics(t, before, "node"))
	advantage["after"] = diff(learningMetrics(t, after, "graph"), learningMetrics(t, after, "node"))
	referenceChecks := 0
	for _, phase := range []map[string]json.RawMessage{before, after} {
		for _, raw := range phase {
			var check learningCheck
			learningDecode(t, raw, &check)
			referenceChecks += check.Costs.ReferenceChecks
		}
	}
	return map[string]any{"after_minus_before": change, "graph_minus_node": advantage, "graph_rows_agree": true, "main_reference_checks": strconv.Itoa(referenceChecks), "metric_deltas": "signed exact decimal strings; zero and negative values retained"}
}
func learningControls(t *testing.T, w learningWorker, before, after learningSnapshot) []learningEvaluation {
	t.Helper()
	controls := []learningEvaluation{}
	add := func(name string, s learningSnapshot, abstentions int) learningEvaluation {
		e := w.evaluate(t, name, s)
		require.Equal(t, abstentions, learningMetrics(t, e.Checks, "graph")["abstentions"])
		controls = append(controls, e)
		return e
	}
	b, a := learningN(t, before, 5), learningN(t, after, 5)
	q1b, q1a := learningClone(t, b), learningClone(t, a)
	q1b.Tasks = q1b.Tasks[:1]
	q1a.Tasks = q1a.Tasks[:1]
	x := add("q1-before-no-numerical-benefit", q1b, 0)
	y := add("q1-after-no-numerical-benefit", q1a, 0)
	var ux, uy learningUse
	learningDecode(t, x.Uses["graph"], &ux)
	learningDecode(t, y.Uses["graph"], &uy)
	require.Equal(t, ux.Outputs[0].Value, uy.Outputs[0].Value)
	unchanged := add("no-change-replay", a, 0)
	require.Equal(t, unchanged.Uses, w.use(t, a))
	// An actual n=5 correction is disconnected from the n=3 request; no new
	// fabricated chain nodes or authoritative history are needed for this control.
	disconnected := learningClone(t, after)
	disconnected.Tasks = learningN(t, before, 3).Tasks
	disconnectedUse := add("disconnected-correction", disconnected, 0)
	isolatedUses := w.use(t, learningN(t, after, 3))
	for _, view := range []string{"node", "graph", "rows"} {
		var withCorrection, withoutCorrection learningUse
		learningDecode(t, disconnectedUse.Uses[view], &withCorrection)
		learningDecode(t, isolatedUses[view], &withoutCorrection)
		require.Equal(t, withoutCorrection.Outputs, withCorrection.Outputs)
	}
	missing := learningClone(t, b)
	missing.Relations = []learningEdge{}
	e := add("missing-relationships", missing, 3)
	require.Equal(t, 0, learningMetrics(t, e.Checks, "node")["abstentions"])
	wrong := learningClone(t, b)
	for i := range wrong.Relations {
		wrong.Relations[i].Source, wrong.Relations[i].Target = wrong.Relations[i].Target, wrong.Relations[i].Source
	}
	add("wrong-direction-relationships", wrong, 3)
	incomplete := learningClone(t, a)
	incomplete.HistoryComplete = false
	add("incomplete-history", incomplete, 3)
	unresolved := learningClone(t, a)
	unresolved.History[0].Outcome = "UNRESOLVED"
	add("unresolved-correction", unresolved, 3)
	contradictory := learningClone(t, b)
	for _, f := range a.Facts {
		if f.ID == a.History[0].NewID {
			contradictory.Facts = append(contradictory.Facts, f)
		}
	}
	contradictory.Tasks = contradictory.Tasks[1:]
	add("contradictory-accepted-content", contradictory, 2)
	outside := learningClone(t, a)
	outside.Tasks = []learningTask{{9, "1"}, {5, "7/3"}}
	e = add("out-of-scope", outside, 2)
	for _, raw := range e.Uses {
		var use learningUse
		learningDecode(t, raw, &use)
		require.Zero(t, use.Costs.Enumerations)
	}
	empty := learningSnapshot{Tasks: b.Tasks, Budget: 1, Facts: []learningFact{}, Relations: []learningEdge{}, History: []learningHistory{}, HistoryComplete: true}
	e = add("bounded-recomputation", empty, 2)
	require.Equal(t, 1, learningMetrics(t, e.Checks, "node")["recomputations"])
	return controls
}

// Replay first verifies the closed object chain and locally trusted method
// hashes, then executes only fixed worker operations on retained snapshots.
// Finally reconstruct the real-handler scenario in a NEW memory app and compare
// every canonical stage (including actual rounds, bundles, negative branch and
// attribution). Evidence storage is read-only throughout; the scratch app is not
// a public chain or a source of independent testimony.
func learningVerifyReplay(t *testing.T, w learningWorker, run *tokLearningReceipts) {
	t.Helper()
	task, policy, method := learningBindings(t, w)
	boundTask, boundPolicy, boundMethod := run.Bindings()
	require.Equal(t, learningJSON(t, task), boundTask, "task binding")
	require.Equal(t, learningJSON(t, policy), boundPolicy, "policy binding")
	require.Equal(t, learningJSON(t, method), boundMethod, "trusted source/method binding")
	useRaw, _, err := run.Stage("use")
	require.NoError(t, err)
	var use struct {
		Snapshot learningSnapshot           `json:"snapshot"`
		Uses     map[string]json.RawMessage `json:"uses"`
	}
	learningDecode(t, useRaw, &use)
	require.Equal(t, use.Uses, w.use(t, use.Snapshot))
	feedbackRaw, _, err := run.Stage("feedback")
	require.NoError(t, err)
	var feedback struct {
		Checks map[string]json.RawMessage `json:"checks"`
	}
	learningDecode(t, feedbackRaw, &feedback)
	require.Equal(t, feedback.Checks, w.check(t, use.Snapshot, use.Uses))
	correctionRaw, _, err := run.Stage("correction")
	require.NoError(t, err)
	var correction struct {
		Snapshot learningSnapshot   `json:"snapshot"`
		Negative learningEvaluation `json:"negative"`
	}
	learningDecode(t, correctionRaw, &correction)
	require.Equal(t, correction.Negative, w.evaluate(t, correction.Negative.Name, correction.Negative.Snapshot))
	rerunRaw, _, err := run.Stage("rerun")
	require.NoError(t, err)
	var rerun struct {
		Uses     map[string]json.RawMessage `json:"uses"`
		Checks   map[string]json.RawMessage `json:"checks"`
		Controls []learningEvaluation       `json:"controls"`
	}
	learningDecode(t, rerunRaw, &rerun)
	require.Equal(t, rerun.Uses, w.use(t, correction.Snapshot))
	require.Equal(t, rerun.Checks, w.check(t, correction.Snapshot, rerun.Uses))
	for _, control := range rerun.Controls {
		require.Equal(t, control, w.evaluate(t, control.Name, control.Snapshot))
	}
	rebuilt := learningRun(t, w)
	for _, stage := range tokLearningReceiptStages() {
		payload, metadata, e := run.Stage(stage)
		require.NoError(t, e)
		expectedPayload, expectedMetadata, e := rebuilt.Stage(stage)
		require.NoError(t, e)
		require.Equal(t, string(expectedPayload), string(payload), "reconstructed %s payload", stage)
		require.Equal(t, string(expectedMetadata), string(metadata), "reconstructed %s metadata", stage)
	}
	expectedHead, err := rebuilt.Head()
	require.NoError(t, err)
	head, err := run.Head()
	require.NoError(t, err)
	require.Equal(t, expectedHead, head)
}

func TestToK_FoldToFireLearningLoop(t *testing.T) {
	require.False(t, *learningOut != "" && *learningReplay != "", "run and replay are mutually exclusive")
	require.False(t, *learningExpectedHead != "" && *learningReplay == "", "expected head is replay-only")
	root := *learningRoot
	if root == "" {
		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		root = filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	}
	root, err := filepath.Abs(root)
	require.NoError(t, err)
	// Node resolves the entry module's real path; match its exact CLI guard.
	root, err = filepath.EvalSymlinks(root)
	require.NoError(t, err)
	node, err := exec.LookPath("node")
	require.NoError(t, err, "Node required by opt-in tok_learning test; never auto-installed")
	node, err = filepath.Abs(node)
	require.NoError(t, err)
	w := learningWorker{root, node}
	if *learningReplay != "" {
		run, err := loadTokLearningReceipts(*learningReplay, *learningExpectedHead)
		require.NoError(t, err)
		learningVerifyReplay(t, w, run)
		head, err := run.Head()
		require.NoError(t, err)
		t.Logf("REPLAY verified head=%s; retained computation and fresh memory-app reconstruction; evidence read-only", head)
		return
	}
	run := learningRun(t, w)
	destination := *learningOut
	if destination == "" {
		destination = filepath.Join(t.TempDir(), "evidence")
	}
	head, err := run.Write(destination)
	require.NoError(t, err)
	loaded, err := loadTokLearningReceipts(destination, head)
	require.NoError(t, err)
	loadedHead, err := loaded.Head()
	require.NoError(t, err)
	require.Equal(t, head, loadedHead)
	t.Logf("RUN evidence=%s head=%s; zero live effects, allocation NOT_EVALUATED", destination, head)
}
