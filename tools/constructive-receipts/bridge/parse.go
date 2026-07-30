package bridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	maxRequestBytes = 64 << 10
	maxTreeBytes    = 256 << 10
)

var boundedID = regexp.MustCompile(`^[a-z0-9][a-z0-9._:@/-]{0,127}$`)

// ParseRequest strictly decodes and validates a receipt request.
func ParseRequest(data []byte) (Request, error) {
	var request Request
	if err := decodeStrict(data, maxRequestBytes, false, &request); err != nil {
		return Request{}, fmt.Errorf("request: %w", err)
	}
	root, err := requireObjectFields(data, "$", "schema", "target", "poca", "source")
	if err != nil {
		return Request{}, fmt.Errorf("request: %w", err)
	}
	if _, err := requireObjectFields(
		root["target"],
		"$.target",
		"tree_schema",
		"tree_policy_version",
		"tree_policy_digest",
		"tree_document_digest",
		"node_id",
		"node_digest",
	); err != nil {
		return Request{}, fmt.Errorf("request: %w", err)
	}
	if _, err := requireObjectFields(
		root["poca"],
		"$.poca",
		"profile_id",
		"profile_version",
		"profile_digest",
		"evidence_bundle_digest",
		"subject_digest",
	); err != nil {
		return Request{}, fmt.Errorf("request: %w", err)
	}
	if _, err := requireObjectFields(
		root["source"],
		"$.source",
		"source_system",
		"record_id",
		"revision",
	); err != nil {
		return Request{}, fmt.Errorf("request: %w", err)
	}
	if err := validateRequest(request); err != nil {
		return Request{}, fmt.Errorf("request: %w", err)
	}
	return request, nil
}

func validateRequest(request Request) error {
	if request.Schema != RequestSchema {
		return fmt.Errorf("schema must be %q", RequestSchema)
	}
	for _, field := range []struct {
		path  string
		value string
	}{
		{"target.tree_policy_digest", request.Target.TreePolicyDigest},
		{"target.tree_document_digest", request.Target.TreeDocumentDigest},
		{"target.node_digest", request.Target.NodeDigest},
		{"poca.profile_digest", request.PoCA.ProfileDigest},
		{"poca.evidence_bundle_digest", request.PoCA.EvidenceBundleDigest},
		{"poca.subject_digest", request.PoCA.SubjectDigest},
	} {
		if err := validateDigest(field.path, field.value); err != nil {
			return err
		}
	}
	for _, field := range []struct {
		path  string
		value string
	}{
		{"poca.profile_id", request.PoCA.ProfileID},
		{"poca.profile_version", request.PoCA.ProfileVersion},
		{"source.source_system", request.Source.SourceSystem},
		{"source.record_id", request.Source.RecordID},
		{"source.revision", request.Source.Revision},
	} {
		if !boundedID.MatchString(field.value) {
			return fmt.Errorf("%s must match %s", field.path, boundedID.String())
		}
	}
	if request.Target.TreeSchema != TreeSchema {
		return fmt.Errorf("target.tree_schema must be %q", TreeSchema)
	}
	if request.Target.TreePolicyVersion != TreePolicyVersion {
		return fmt.Errorf("target.tree_policy_version must be %q", TreePolicyVersion)
	}
	if request.Target.NodeID != TargetNodeID {
		return fmt.Errorf("target.node_id must be %q", TargetNodeID)
	}
	return nil
}

type treeDocument struct {
	Schema          string            `json:"schema"`
	Authoritative   bool              `json:"authoritative"`
	NetworkObserved bool              `json:"networkObserved"`
	RewardBearing   bool              `json:"rewardBearing"`
	SnapshotDate    string            `json:"snapshotDate"`
	PolicyVersion   string            `json:"policyVersion"`
	ReleaseBoundary releaseBoundary   `json:"releaseBoundary"`
	Policy          json.RawMessage   `json:"policy"`
	Roots           []string          `json:"roots"`
	Nodes           []json.RawMessage `json:"nodes"`
}

type releaseBoundary struct {
	AddsConsensusBehavior         bool `json:"addsConsensusBehavior"`
	ActivatesRewards              bool `json:"activatesRewards"`
	MovesFunds                    bool `json:"movesFunds"`
	GrantsQualification           bool `json:"grantsQualification"`
	AuthorizesSecurityTesting     bool `json:"authorizesSecurityTesting"`
	AssertsProtocolSecurity       bool `json:"assertsProtocolSecurity"`
	PerformsNetworkRequests       bool `json:"performsNetworkRequests"`
	PublishesConfidentialEvidence bool `json:"publishesConfidentialEvidence"`
}

type treeNode struct {
	ID                    string          `json:"id"`
	Title                 string          `json:"title"`
	Stage                 string          `json:"stage"`
	Domain                string          `json:"domain"`
	Summary               string          `json:"summary"`
	Prerequisites         []string        `json:"prerequisites"`
	AttainmentEvidence    string          `json:"attainmentEvidence"`
	RewardEligibility     string          `json:"rewardEligibility"`
	DefaultDisclosureLane string          `json:"defaultDisclosureLane"`
	ArtifactRequirements  []string        `json:"artifactRequirements"`
	RevalidationTriggers  []string        `json:"revalidationTriggers"`
	Standards             []treeStandard  `json:"standards"`
	RepositoryReferences  []string        `json:"repositoryReferences"`
	Acceptance            json.RawMessage `json:"acceptance"`
}

type treeStandard struct {
	CanonicalID        string `json:"canonicalId"`
	Authority          string `json:"authority"`
	Title              string `json:"title"`
	Revision           string `json:"revision"`
	AuthorityStatus    string `json:"authorityStatus"`
	NormalizedMaturity string `json:"normalizedMaturity"`
	Specification      string `json:"specification"`
	StatusCheckedAt    string `json:"statusCheckedAt"`
	ReviewAfter        string `json:"reviewAfter"`
}

type parsedTree struct {
	DocumentDigest string
	PolicyDigest   string
	NodeDigest     string
	Node           treeNode
}

func parseTree(data []byte) (parsedTree, error) {
	var document treeDocument
	if err := decodeStrict(data, maxTreeBytes, true, &document); err != nil {
		return parsedTree{}, fmt.Errorf("tree: %w", err)
	}
	root, err := requireObjectFields(
		data,
		"$",
		"schema",
		"authoritative",
		"networkObserved",
		"rewardBearing",
		"snapshotDate",
		"policyVersion",
		"releaseBoundary",
		"policy",
		"roots",
		"nodes",
	)
	if err != nil {
		return parsedTree{}, fmt.Errorf("tree: %w", err)
	}
	if _, err := requireObjectFields(
		root["releaseBoundary"],
		"$.releaseBoundary",
		"addsConsensusBehavior",
		"activatesRewards",
		"movesFunds",
		"grantsQualification",
		"authorizesSecurityTesting",
		"assertsProtocolSecurity",
		"performsNetworkRequests",
		"publishesConfidentialEvidence",
	); err != nil {
		return parsedTree{}, fmt.Errorf("tree: %w", err)
	}
	if document.Schema != TreeSchema {
		return parsedTree{}, fmt.Errorf("tree schema must be %q", TreeSchema)
	}
	if document.PolicyVersion != TreePolicyVersion {
		return parsedTree{}, fmt.Errorf("tree policyVersion must be %q", TreePolicyVersion)
	}
	if document.Authoritative || document.NetworkObserved || document.RewardBearing {
		return parsedTree{}, errors.New("tree authority, network, and reward boundary flags must remain false")
	}
	if document.ReleaseBoundary.AddsConsensusBehavior ||
		document.ReleaseBoundary.ActivatesRewards ||
		document.ReleaseBoundary.MovesFunds ||
		document.ReleaseBoundary.GrantsQualification ||
		document.ReleaseBoundary.AuthorizesSecurityTesting ||
		document.ReleaseBoundary.AssertsProtocolSecurity ||
		document.ReleaseBoundary.PerformsNetworkRequests ||
		document.ReleaseBoundary.PublishesConfidentialEvidence {
		return parsedTree{}, errors.New("every tree release boundary must remain false")
	}

	documentDigest := digestBytes(data)
	if documentDigest != ReviewedTreeDigest {
		return parsedTree{}, fmt.Errorf(
			"tree document digest mismatch: expected %s, got %s",
			ReviewedTreeDigest,
			documentDigest,
		)
	}
	policyDigest, err := digestCanonicalRaw(document.Policy)
	if err != nil {
		return parsedTree{}, fmt.Errorf("tree policy: %w", err)
	}
	if policyDigest != ReviewedTreePolicyDigest {
		return parsedTree{}, fmt.Errorf(
			"tree policy digest mismatch: expected %s, got %s",
			ReviewedTreePolicyDigest,
			policyDigest,
		)
	}

	var target treeNode
	var targetRaw json.RawMessage
	targetCount := 0
	for index, raw := range document.Nodes {
		var header struct {
			ID string `json:"id"`
		}
		// The enclosing strict scan has already rejected duplicate keys, and the
		// exact document digest binds every non-target node. Decode only the id
		// here; the pinned target receives a full strict decode below.
		if err := json.Unmarshal(raw, &header); err != nil {
			return parsedTree{}, fmt.Errorf("tree nodes[%d]: %w", index, err)
		}
		if header.ID != TargetNodeID {
			continue
		}
		targetCount++
		if err := decodeStrict(raw, maxTreeBytes, true, &target); err != nil {
			return parsedTree{}, fmt.Errorf("tree target node: %w", err)
		}
		targetRaw = raw
	}
	if targetCount != 1 {
		return parsedTree{}, fmt.Errorf("tree must contain target node %q exactly once", TargetNodeID)
	}
	if target.Stage != TargetNodeStage ||
		target.AttainmentEvidence != TargetNodeEvidence ||
		target.RewardEligibility != TargetRewardEligibility {
		return parsedTree{}, fmt.Errorf(
			"tree target node contract drift: stage=%q attainment=%q reward=%q",
			target.Stage,
			target.AttainmentEvidence,
			target.RewardEligibility,
		)
	}
	nodeDigest, err := digestCanonicalRaw(targetRaw)
	if err != nil {
		return parsedTree{}, fmt.Errorf("tree target node: %w", err)
	}
	if nodeDigest != ReviewedTargetNodeDigest {
		return parsedTree{}, fmt.Errorf(
			"tree target node digest mismatch: expected %s, got %s",
			ReviewedTargetNodeDigest,
			nodeDigest,
		)
	}
	return parsedTree{
		DocumentDigest: documentDigest,
		PolicyDigest:   policyDigest,
		NodeDigest:     nodeDigest,
		Node:           target,
	}, nil
}

func decodeStrict(data []byte, maximum int, allowNull bool, destination any) error {
	if len(data) == 0 {
		return errors.New("document is empty")
	}
	if len(data) > maximum {
		return fmt.Errorf("document exceeds %d-byte limit", maximum)
	}
	if err := inspectJSON(data, allowNull); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	return nil
}

func inspectJSON(data []byte, allowNull bool) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func(string) error
	walk = func(path string) error {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON token at %s: %w", path, err)
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			if token == nil && !allowNull {
				return fmt.Errorf("JSON null is not allowed at %s", path)
			}
			return nil
		}
		switch delimiter {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return fmt.Errorf("decode JSON object key at %s: %w", path, err)
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("non-string JSON object key at %s", path)
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON object key %q at %s", key, path)
				}
				seen[key] = struct{}{}
				if err := walk(path + "." + key); err != nil {
					return err
				}
			}
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("close JSON object at %s: %w", path, err)
			}
		case '[':
			index := 0
			for decoder.More() {
				if err := walk(fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
				index++
			}
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("close JSON array at %s: %w", path, err)
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q at %s", delimiter, path)
		}
		return nil
	}
	if err := walk("$"); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing JSON token: %w", err)
	}
	return nil
}

func requireObjectFields(raw json.RawMessage, path string, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", path, err)
	}
	if object == nil {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	for _, field := range fields {
		value, ok := object[field]
		if !ok {
			return nil, fmt.Errorf("%s is missing required field %q", path, field)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("%s.%s must not be null", path, field)
		}
	}
	return object, nil
}

func digestCanonicalRaw(raw json.RawMessage) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode canonical JSON: %w", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode canonical JSON: %w", err)
	}
	return digestBytes(encoded), nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validateDigest(path, value string) error {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("%s must be sha256:<64 lowercase hex>", path)
	}
	for _, character := range value[len("sha256:"):] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return fmt.Errorf("%s must be sha256:<64 lowercase hex>", path)
		}
	}
	return nil
}
