package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// OracleEvaluation represents the evaluation result from the oracle sidecar.
type OracleEvaluation struct {
	Verdict    string  `json:"verdict"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// OracleClient defines the interface for evaluating claims via an oracle.
type OracleClient interface {
	Evaluate(claim, domain, claimType string) (*OracleEvaluation, error)
}

// HTTPOracleClient implements OracleClient by calling an HTTP oracle sidecar.
type HTTPOracleClient struct {
	endpoint      string
	timeout       time.Duration
	minConfidence float64
	client        *http.Client
}

// NewHTTPOracleClient creates a new HTTPOracleClient with the given endpoint,
// timeout, and minimum confidence threshold.
func NewHTTPOracleClient(endpoint string, timeout time.Duration, minConfidence float64) *HTTPOracleClient {
	return &HTTPOracleClient{
		endpoint:      endpoint,
		timeout:       timeout,
		minConfidence: minConfidence,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Evaluate sends a claim to the oracle sidecar for evaluation.
//
// The method POSTs a JSON payload to {endpoint}/evaluate and decodes the
// response into an OracleEvaluation. If the oracle's confidence falls below
// the configured minConfidence threshold and the verdict is not already
// "uncertain", the verdict is overridden to "uncertain" with a note appended
// to the reasoning.
func (c *HTTPOracleClient) Evaluate(claim, domain, claimType string) (*OracleEvaluation, error) {
	reqBody := map[string]string{
		"claim":      claim,
		"domain":     domain,
		"claim_type": claimType,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.endpoint+"/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("oracle: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oracle: non-200 status: %d", resp.StatusCode)
	}

	var result OracleEvaluation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("oracle: failed to decode response: %w", err)
	}

	// Confidence threshold: override verdict to "uncertain" if confidence is
	// below the minimum and the verdict is not already "uncertain".
	if result.Confidence < c.minConfidence && result.Verdict != "uncertain" {
		result.Verdict = "uncertain"
		result.Reasoning = fmt.Sprintf("%s (below minimum confidence threshold %.2f)", result.Reasoning, c.minConfidence)
	}

	return &result, nil
}

// EvaluateWithOracle returns ok=false when there is no usable verdict: no
// client, an error, an "uncertain" verdict (including one downgraded for low
// confidence), or any verdict other than "accept" or "reject". The caller then
// casts no vote for that claim.
func EvaluateWithOracle(client OracleClient, claim, domain, claimType string) (verdict string, confidence uint64, ok bool) {
	if client == nil {
		return "", 0, false
	}

	result, err := client.Evaluate(claim, domain, claimType)
	if err != nil || result == nil {
		return "", 0, false
	}

	// Only "accept" and "reject" are usable ballots. "uncertain", "malformed"
	// and any other word mean no vote, never a default accept.
	switch result.Verdict {
	case "accept", "reject":
	default:
		return "", 0, false
	}

	// Map confidence: float64 [0,1] -> uint64 [0,1_000_000], capped at
	// 1_000_000. Clamp before converting, because Go leaves the conversion of
	// a negative, NaN or out-of-range float64 to uint64 implementation-defined.
	c := result.Confidence
	if math.IsNaN(c) || c < 0 {
		return "", 0, false
	}
	if c > 1 {
		c = 1
	}

	return result.Verdict, uint64(c * 1_000_000), true
}
