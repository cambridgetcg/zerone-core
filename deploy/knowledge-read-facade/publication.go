package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

const maxPublications = 32
const geometrySchema = "zerone.knowledge-geometry-snapshot/v0"
const snapshotSchema = "zerone.knowledge-public-snapshot/v1"
const queryPath = "/zerone/knowledge/v1/facts?pagination.limit=100"
const topologyScope = "TOK_ROOT/v1 over projected node IDs and all returned canonical relation/inference edges; not a ToK selector root; excludes content, method, strength, time, history and source metadata"
const payloadScope = "SHA-256 of exact embedded UTF-8 compact JSON payload bytes, including projected metadata; not a chain proof"

// The existing dashboard _knowledge.ts projection is the input contract. No
// embedded legacy adjacency, inference engine, observer, or live scan is added.
type projectionSource struct {
	ChainID         string `json:"chainId"`
	BlockHeight     string `json:"blockHeight"`
	StatusHeight    string `json:"statusHeight"`
	CatchingUp      bool   `json:"catchingUp"`
	QueryPath       string `json:"queryPath"`
	QueryTracked    bool   `json:"queryTracked"`
	Writes          bool   `json:"writes"`
	Completeness    string `json:"completeness"`
	UpstreamRecords int    `json:"upstreamRecords"`
	ReturnedRecords int    `json:"returnedRecords"`
	Truncated       bool   `json:"truncated"`
}
type projectedFact struct {
	ID                string `json:"id"`
	Content           string `json:"content"`
	Domain            string `json:"domain"`
	Category          string `json:"category"`
	Status            string `json:"status"`
	ClaimType         string `json:"claimType"`
	Confidence        int    `json:"confidence"`
	VerifiedAtBlock   string `json:"verifiedAtBlock"`
	LastVerifiedBlock string `json:"lastVerifiedBlock"`
	Energy            int    `json:"energy"`
	EnergyCap         int    `json:"energyCap"`
	FitnessScore      int    `json:"fitnessScore"`
	MethodID          string `json:"methodId"`
}
type projectedRelation struct {
	SourceFactID         string `json:"sourceFactId"`
	TargetFactID         string `json:"targetFactId"`
	Relation             string `json:"relation"`
	Inference            string `json:"inference"`
	InferenceStrengthBps int    `json:"inferenceStrengthBps"`
	CreatedAtBlock       string `json:"createdAtBlock"`
	MethodID             string `json:"methodId"`
}
type projection struct {
	Schema    string              `json:"schema"`
	Source    projectionSource    `json:"source"`
	Facts     []projectedFact     `json:"facts"`
	Relations []projectedRelation `json:"relations"`
}

// Metadata is supplied by an operator after obtaining the canonical actual-query
// projection. A digest binds that assertion to bytes; it does not prove origin,
// release deployment, chain execution, custody, or authority.
type publicationMetadata struct {
	Schema           string `json:"schema"`
	ChainID          string `json:"chainId"`
	BlockHeight      string `json:"blockHeight"`
	StatusHeight     string `json:"statusHeight"`
	BlockTime        string `json:"blockTime"`
	ObservedAt       string `json:"observedAt"`
	RESTOrigin       string `json:"restOrigin"`
	RPCOrigin        string `json:"rpcOrigin"`
	SourceCommit     string `json:"sourceCommit"`
	ProjectionSHA256 string `json:"projectionSha256"`
	Provenance       string `json:"provenance"`
}

const metadataSchema = "zerone.knowledge-publication-metadata/v1"
const provenanceScope = "operator-asserted actual-query projection; not chain, release, custody or authority proof"

type snapshotFreshness struct {
	Kind         string `json:"kind"`
	StatusHeight string `json:"statusHeight"`
	CatchingUp   bool   `json:"catchingUp"`
	LiveStatus   string `json:"liveStatus"`
}
type snapshotCoverage struct {
	Kind               string `json:"kind"`
	WholeGraph         string `json:"wholeGraph"`
	CanonicalRelations string `json:"canonicalRelations"`
	History            string `json:"history"`
	Truncated          bool   `json:"truncated"`
	Nodes              int    `json:"nodes"`
	Edges              int    `json:"edges"`
}
type publicSnapshot struct {
	Schema             string              `json:"schema"`
	ChainID            string              `json:"chainId"`
	BlockHeight        string              `json:"blockHeight"`
	ObservedAt         string              `json:"observedAt"`
	Freshness          snapshotFreshness   `json:"freshness"`
	Coverage           snapshotCoverage    `json:"coverage"`
	TopologyRoot       string              `json:"topologyRoot"`
	TopologyRootScope  string              `json:"topologyRootScope"`
	PayloadDigest      string              `json:"payloadDigest"`
	PayloadDigestScope string              `json:"payloadDigestScope"`
	Publication        publicationMetadata `json:"publication"`
	Payload            json.RawMessage     `json:"payload"`
}
type publicationHead struct {
	Schema      string `json:"schema"`
	SnapshotID  string `json:"snapshotId"`
	ChainID     string `json:"chainId"`
	BlockHeight string `json:"blockHeight"`
	ObservedAt  string `json:"observedAt"`
	Coverage    string `json:"coverage"`
}

// Exact-key reflection prevents Go's struct field case folding at ALL typed
// publication/config boundaries, and requires explicit fields rather than zero
// defaults. RawMessage is checked separately against its own typed contract.
func decodeExact(body []byte, out any, maximum int) error {
	if err := validateJSON(body, maximum); err != nil {
		return err
	}
	var check func(json.RawMessage, reflect.Type) error
	check = func(raw json.RawMessage, t reflect.Type) error {
		if t == reflect.TypeOf(json.RawMessage{}) {
			return nil
		}
		if bytes.Equal(raw, []byte("null")) {
			return errors.New("null is not an explicit field value")
		}
		switch t.Kind() {
		case reflect.Struct:
			var obj map[string]json.RawMessage
			if json.Unmarshal(raw, &obj) != nil || obj == nil {
				return errors.New("object required")
			}
			if len(obj) != t.NumField() {
				return errors.New("missing or unknown field")
			}
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				name := field.Tag.Get("json")
				value, ok := obj[name]
				if !ok {
					return fmt.Errorf("exact field %s required", name)
				}
				if err := check(value, field.Type); err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}
			}
		case reflect.Slice:
			var values []json.RawMessage
			if json.Unmarshal(raw, &values) != nil || values == nil {
				return errors.New("array required")
			}
			for _, v := range values {
				if err := check(v, t.Elem()); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := check(body, reflect.TypeOf(out).Elem()); err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}
func compactJSON(value any) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}
func boundedText(s string, max int, empty bool) bool {
	if len(s) > max || (!empty && strings.TrimSpace(s) == "") {
		return false
	}
	for _, r := range s {
		if (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') || r == 0x61c || r == 0x200e || r == 0x200f || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
			return false
		}
	}
	return true
}
func member(s, prefix, values string) bool {
	for _, v := range strings.Fields(values) {
		if s == prefix+v {
			return true
		}
	}
	return false
}
func metric(n int) bool      { return n >= 0 && n <= 1000000 }
func methodID(s string) bool { return s == "" || factIDPattern.MatchString(s) }
func heightLE(a, b string) bool {
	if !validHeight(a, false) || !validHeight(b, true) {
		return false
	}
	return len(a) < len(b) || (len(a) == len(b) && a <= b)
}
func validateProjection(raw []byte) (projection, error) {
	var p projection
	if err := decodeExact(raw, &p, snapshotBytes); err != nil {
		return p, err
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) != nil || !bytes.Equal(compact.Bytes(), raw) {
		return p, errors.New("projection must be exact compact query projection JSON")
	}
	s := p.Source
	if p.Schema != geometrySchema || s.ChainID != "zerone-1" || !nearHeights(s.BlockHeight, s.StatusHeight) || s.QueryPath != queryPath || s.QueryTracked || s.Writes || s.Completeness != "NOT_CLAIMED" || len(p.Facts) > 128 || len(p.Relations) > 512 || s.ReturnedRecords != len(p.Facts) || s.UpstreamRecords < s.ReturnedRecords || s.UpstreamRecords > 128 || (!s.Truncated && s.UpstreamRecords != s.ReturnedRecords) {
		return p, errors.New("invalid projection source or bounds")
	}
	ids := map[string]bool{}
	previous := ""
	for _, f := range p.Facts {
		if !factIDPattern.MatchString(f.ID) || f.ID <= previous || !boundedText(f.Content, 16384, false) || !boundedText(f.Domain, 128, true) || !boundedText(f.Category, 128, true) || !methodID(f.MethodID) || !metric(f.Confidence) || !metric(f.Energy) || !metric(f.EnergyCap) || !metric(f.FitnessScore) || !member(f.Status, "FACT_STATUS_", "UNSPECIFIED PENDING PROVISIONAL VERIFIED ACTIVE CONTESTED CHALLENGED SUPERSEDED EXPIRED DISPROVEN REVOKED AT_RISK PRUNED") || !member(f.ClaimType, "CLAIM_TYPE_", "UNSPECIFIED ASSERTION RELATION DEFINITION CONSTRAINT NEGATION OBSERVATION COMPUTATIONAL CONJECTURE") || !heightLE(f.VerifiedAtBlock, s.BlockHeight) || !heightLE(f.LastVerifiedBlock, s.BlockHeight) || !heightLE(f.VerifiedAtBlock, s.StatusHeight) || !heightLE(f.LastVerifiedBlock, s.StatusHeight) || (!heightLE(f.VerifiedAtBlock, f.LastVerifiedBlock) && !(f.VerifiedAtBlock == "0" && f.LastVerifiedBlock == "0")) {
			return p, errors.New("invalid projected fact")
		}
		ids[f.ID] = true
		previous = f.ID
	}
	pairs := map[string]bool{}
	previous = ""
	for _, e := range p.Relations {
		pair := e.SourceFactID + "\x00" + e.TargetFactID
		identity := strings.Join([]string{e.SourceFactID, e.TargetFactID, e.Relation, e.Inference, fmt.Sprint(e.InferenceStrengthBps), e.CreatedAtBlock, e.MethodID}, "\x00")
		if !factIDPattern.MatchString(e.SourceFactID) || !factIDPattern.MatchString(e.TargetFactID) || (!ids[e.SourceFactID] && !ids[e.TargetFactID]) || pairs[pair] || identity <= previous || !member(e.Relation, "RELATION_TYPE_", "UNSPECIFIED SUPPORTS CONTRADICTS REQUIRES REFINES GENERALIZES SUPERSEDES CITES REFORMULATES") || !member(e.Inference, "INFERENCE_TYPE_", "UNSPECIFIED DEDUCTIVE INDUCTIVE ABDUCTIVE EMPIRICAL ANALOGICAL CITATION") || !metric(e.InferenceStrengthBps) || !methodID(e.MethodID) || !heightLE(e.CreatedAtBlock, s.BlockHeight) || !heightLE(e.CreatedAtBlock, s.StatusHeight) {
			return p, errors.New("invalid canonical projected relation")
		}
		pairs[pair] = true
		previous = identity
	}
	return p, nil
}
func projectionRoot(p projection) string {
	field := func(b *bytes.Buffer, s string) {
		_ = binary.Write(b, binary.BigEndian, uint64(len(s)))
		b.WriteString(s)
	}
	var nodes, edges, root bytes.Buffer
	field(&nodes, "TOK_NODES")
	field(&edges, "TOK_EDGES")
	field(&root, "TOK_ROOT")
	ids := make([]string, 0, len(p.Facts))
	for _, f := range p.Facts {
		ids = append(ids, f.ID)
	}
	sort.Strings(ids)
	for _, id := range ids {
		field(&nodes, id)
	}
	tuples := make([][4]string, 0, len(p.Relations))
	for _, e := range p.Relations {
		tuples = append(tuples, [4]string{e.SourceFactID, e.TargetFactID, e.Relation, e.Inference})
	}
	sort.Slice(tuples, func(i, j int) bool {
		for k := 0; k < 4; k++ {
			if tuples[i][k] != tuples[j][k] {
				return tuples[i][k] < tuples[j][k]
			}
		}
		return false
	})
	for _, e := range tuples {
		for _, v := range e {
			field(&edges, v)
		}
	}
	// The final root includes raw SHA-256 bytes, not their hexadecimal spelling.
	for _, b := range [][]byte{nodes.Bytes(), edges.Bytes()} {
		h := sha256Bytes(b)
		root.Write(h)
	}
	return hashBytes(root.Bytes())
}
func sha256Bytes(b []byte) []byte { h := sha256.Sum256(b); return h[:] }
func validateMetadata(m publicationMetadata, p projection, raw []byte) error {
	observed, err := time.Parse(time.RFC3339Nano, m.ObservedAt)
	if err != nil {
		return err
	}
	block, err := time.Parse(time.RFC3339Nano, m.BlockTime)
	if err != nil {
		return err
	}
	if m.Schema != metadataSchema || m.ChainID != p.Source.ChainID || m.BlockHeight != p.Source.BlockHeight || m.StatusHeight != p.Source.StatusHeight || p.Source.CatchingUp || m.ProjectionSHA256 != hashBytes(raw) || m.Provenance != provenanceScope || len(m.SourceCommit) != 40 || !isLowerHex(m.SourceCommit) || observed.Sub(block) > freshnessWindow || block.Sub(observed) > futureSkew {
		return errors.New("publication metadata mismatch or stale observation")
	}
	if _, err := fixedOrigin(m.RESTOrigin); err != nil {
		return err
	}
	if _, err := fixedOrigin(m.RPCOrigin); err != nil {
		return err
	}
	return nil
}
func isLowerHex(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return s != ""
}
func buildSnapshot(raw []byte, m publicationMetadata, now time.Time) ([]byte, error) {
	p, err := validateProjection(raw)
	if err != nil {
		return nil, err
	}
	if err := validateMetadata(m, p, raw); err != nil {
		return nil, err
	}
	observed, _ := time.Parse(time.RFC3339Nano, m.ObservedAt)
	if now.Sub(observed) > freshnessWindow || observed.Sub(now) > futureSkew {
		return nil, errors.New("publisher requires a fresh observation")
	}
	out := publicSnapshot{snapshotSchema, p.Source.ChainID, p.Source.BlockHeight, m.ObservedAt, snapshotFreshness{"observation-only", p.Source.StatusHeight, p.Source.CatchingUp, "UNKNOWN"}, snapshotCoverage{"bounded-page", "NOT_CLAIMED", "source-asserted", "NOT_INCLUDED", p.Source.Truncated, len(p.Facts), len(p.Relations)}, projectionRoot(p), topologyScope, hashBytes(raw), payloadScope, m, raw}
	body, err := compactJSON(out)
	if err != nil {
		return nil, err
	}
	if _, err := validateSnapshot(body, p.Source.ChainID); err != nil {
		return nil, err
	}
	return body, nil
}
func validateSnapshot(body []byte, chain string) (publicSnapshot, error) {
	var s publicSnapshot
	if err := decodeExact(body, &s, snapshotBytes); err != nil {
		return s, err
	}
	p, err := validateProjection(s.Payload)
	if err != nil {
		return s, err
	}
	if err := validateMetadata(s.Publication, p, s.Payload); err != nil {
		return s, err
	}
	if s.Schema != snapshotSchema || s.ChainID != chain || s.ChainID != p.Source.ChainID || s.BlockHeight != p.Source.BlockHeight || s.ObservedAt != s.Publication.ObservedAt || s.TopologyRootScope != topologyScope || s.TopologyRoot != projectionRoot(p) || s.PayloadDigestScope != payloadScope || s.PayloadDigest != hashBytes(s.Payload) || s.Freshness != (snapshotFreshness{"observation-only", p.Source.StatusHeight, p.Source.CatchingUp, "UNKNOWN"}) || s.Coverage != (snapshotCoverage{"bounded-page", "NOT_CLAIMED", "source-asserted", "NOT_INCLUDED", p.Source.Truncated, len(p.Facts), len(p.Relations)}) {
		return s, errors.New("snapshot identity, root, digest or metadata mismatch")
	}
	return s, nil
}

func readFileBounded(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(maximum) {
		return nil, errors.New("bounded regular file required")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readBounded(f, maximum)
}
func readBounded(r io.Reader, maximum int) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maximum {
		return nil, errors.New("file exceeds byte limit")
	}
	return b, nil
}
