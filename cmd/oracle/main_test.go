package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The axiom-based static evaluator was removed; the oracle is LLM-only now.
// These tests run the server with no LLM configured (CombinedEvaluator{}),
// so /evaluate returns "uncertain" — the honest verdict when the oracle has
// no evaluator and no assumed axiom bedrock to check against.

func TestOracleServer_Health(t *testing.T) {
	eval := &CombinedEvaluator{}
	srv := newOracleServer(eval, "llm")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ok", body["status"])
	require.Equal(t, "llm", body["tier"])
}

func TestOracleServer_Evaluate(t *testing.T) {
	eval := &CombinedEvaluator{}
	srv := newOracleServer(eval, "llm")

	payload := `{"claim":"Electromagnetic waves propagate at the speed of light in vacuum","domain":"physics","claim_type":"fact"}`
	req := httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp EvaluateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	// With no LLM configured, the oracle honestly returns uncertain.
	require.Equal(t, "uncertain", resp.Verdict)
	require.NotEmpty(t, resp.Reasoning)
}

func TestOracleServer_EvaluateBadRequest(t *testing.T) {
	eval := &CombinedEvaluator{}
	srv := newOracleServer(eval, "llm")

	req := httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOracleServer_Prefetch(t *testing.T) {
	eval := &CombinedEvaluator{}
	srv := newOracleServer(eval, "llm")

	payload := `{"claim":"Water boils at 100 degrees Celsius at sea level","domain":"physics","claim_type":"fact"}`
	req := httptest.NewRequest(http.MethodPost, "/prefetch", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "prefetching", body["status"])
}