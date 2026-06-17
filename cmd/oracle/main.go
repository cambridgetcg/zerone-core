package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

// newOracleServer creates an HTTP mux with /health, /evaluate, and /prefetch routes.
func newOracleServer(eval *CombinedEvaluator, tier string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"tier":   tier,
		})
	})

	mux.HandleFunc("POST /evaluate", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
		var req EvaluateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		resp, err := eval.Evaluate(req)
		if err != nil {
			// On eval error, return uncertain response instead of 500.
			resp = &EvaluateResponse{
				Verdict:    "uncertain",
				Confidence: 0.5,
				Reasoning:  "evaluation error: " + err.Error(),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("POST /prefetch", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
		var req EvaluateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		go func() {
			_, _ = eval.Evaluate(req)
		}()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "prefetching",
		})
	})

	return mux
}

func main() {
	port := flag.Int("port", 8081, "HTTP server port")
	llmAPIKey := flag.String("llm-api-key", "", "Anthropic API key (required)")
	llmAPIURL := flag.String("llm-api-url", defaultAnthropicURL, "Anthropic API base URL")
	llmModel := flag.String("llm-model", "claude-sonnet-4-5-20250514", "LLM model name")
	llmMaxTokens := flag.Int("llm-max-tokens", 500, "LLM max tokens")
	llmTimeout := flag.Duration("llm-timeout", 2*time.Second, "LLM request timeout")
	flag.Parse()

	// The axiom-based static evaluator was removed (truth is, not proven;
	// axioms are assumptions — the oracle no longer checks claims against a
	// bedrock of assumed-true facts). The oracle now evaluates via an LLM,
	// so an API key is required.
	if *llmAPIKey == "" {
		log.Fatal("--llm-api-key is required (the axiom-based static evaluator was removed; the oracle evaluates via an LLM)")
	}

	llmEval := NewLLMEvaluator(*llmAPIURL, *llmAPIKey, *llmModel, *llmMaxTokens, *llmTimeout)
	combined := &CombinedEvaluator{LLM: llmEval}
	log.Printf("llm evaluator configured: model=%s url=%s", *llmModel, *llmAPIURL)

	srv := newOracleServer(combined, "llm")
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("zerone-oracle starting on %s (tier=llm)", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}