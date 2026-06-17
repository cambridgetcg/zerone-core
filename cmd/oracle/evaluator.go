package main

// EvaluateRequest is the input to an Evaluator.
type EvaluateRequest struct {
	Claim     string `json:"claim"`
	Domain    string `json:"domain"`
	ClaimType string `json:"claim_type"`
}

// EvaluateResponse is the output from an Evaluator.
type EvaluateResponse struct {
	Verdict    string  `json:"verdict"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// Evaluator is the interface every evaluation strategy must implement.
type Evaluator interface {
	Evaluate(req EvaluateRequest) (*EvaluateResponse, error)
	Name() string
}

// CombinedEvaluator orchestrates LLM evaluation.
//
// The axiom-based StaticEvaluator was removed: truth is, not proven, and
// axioms are assumptions — the oracle no longer checks claims against a
// bedrock of assumed-true facts. The oracle now evaluates via an LLM, or
// returns uncertain when no evaluator is configured. There is no offline
// axiom evaluation.
type CombinedEvaluator struct {
	LLM *LLMEvaluator
}

// Evaluate delegates to the LLM evaluator when one is configured; otherwise
// it returns uncertain. The oracle does not fake evaluation against assumed
// axioms — without an evaluator it honestly says "uncertain."
func (c *CombinedEvaluator) Evaluate(req EvaluateRequest) (*EvaluateResponse, error) {
	if c.LLM != nil {
		return c.LLM.Evaluate(req)
	}
	return &EvaluateResponse{
		Verdict:    "uncertain",
		Confidence: 0.5,
		Reasoning:  "no evaluator configured (the axiom-based static evaluator was removed; an LLM evaluator is required for non-uncertain verdicts)",
	}, nil
}

// Name returns the evaluator name.
func (c *CombinedEvaluator) Name() string {
	return "combined"
}