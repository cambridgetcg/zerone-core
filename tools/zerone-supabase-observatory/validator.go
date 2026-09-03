package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxManifestBytes = 64 << 10
	maxSchemaBytes   = 128 << 10
	maxFixtureBytes  = 128 << 10

	staticTreeRepository       = "https://github.com/cambridgetcg/zerone-core"
	staticTreeRevision         = "264f3c383f408729f4d0c27d332cd454c9eb4400"
	staticTreePath             = "dashboard/public/standards/constructive-intelligence-tree.v1.json"
	staticTreeRawPayloadSHA256 = "sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf"
	knowledgeGeometryChainID   = "zerone-1"
)

var (
	digestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	hex32Pattern   = regexp.MustCompile(`^hex:[0-9a-f]{64}$`)
	heightPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	chainIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

	expectedSourcePins = []sourcePin{
		{"zerone.research-commons-spec", "ZERONE", "CURRENT_TRACKED", "https://github.com/cambridgetcg/zerone-core", "264f3c383f408729f4d0c27d332cd454c9eb4400", "docs/specs/research-commons-rc-0.1.md", "sha256:cff28edbbc7bbad80a101555257046173c19ac05b80b14a9d93317f220dccce7", "CURRENT", "LOCAL_BYTES"},
		{"zerone.research-commons-manifest", "ZERONE", "CURRENT_TRACKED", "https://github.com/cambridgetcg/zerone-core", "264f3c383f408729f4d0c27d332cd454c9eb4400", "dashboard/public/standards/research-commons.v0.1.json", "sha256:94f020f1d37faac48300d14071ec995245aedbdc7f08fc19cafd3797450cdb8c", "CURRENT", "LOCAL_BYTES"},
		{"zerone.tok-substrate-doctrine", "ZERONE", "CURRENT_TRACKED", "https://github.com/cambridgetcg/zerone-core", "264f3c383f408729f4d0c27d332cd454c9eb4400", "docs/TOK_SUBSTRATE.md", "sha256:4fec6e3a410d5736f61cd43f4d9c421380b93f649c2f0d026a5f4e68a6534328", "CURRENT", "LOCAL_BYTES"},
		{"zerone.tok-bundle-implementation", "ZERONE", "CURRENT_TRACKED", "https://github.com/cambridgetcg/zerone-core", "264f3c383f408729f4d0c27d332cd454c9eb4400", "x/knowledge/keeper/tok_bundle.go", "sha256:e2d786d7287a2194ff858d47de84e3b140c9922b7ef37dcfadaec74be675f248", "CURRENT", "LOCAL_BYTES"},
		{"zerone.static-tree", "ZERONE", "CURRENT_TRACKED", "https://github.com/cambridgetcg/zerone-core", "264f3c383f408729f4d0c27d332cd454c9eb4400", "dashboard/public/standards/constructive-intelligence-tree.v1.json", "sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf", "CURRENT", "LOCAL_BYTES"},
		{"zerone.knowledge-geometry-projection", "ZERONE", "CURRENT_TRACKED", "https://github.com/cambridgetcg/zerone-core", "264f3c383f408729f4d0c27d332cd454c9eb4400", "dashboard/functions/api/_knowledge.ts", "sha256:c828c0cd0cdca353d6fd6005d3e3e113cda964123d4a7722d060ad5ed9baed9e", "CURRENT", "LOCAL_BYTES"},
		{"agenttool.research-commons-static-interop", "AGENTTOOL", "CURRENT_TRACKED", "https://github.com/cambridgetcg/agenttool", "796a753ab8624ad11af621ef4572544ea3b8f463", "packages/research-commons/interop/research-commons-zerone-v0.1.json", "sha256:8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a", "CURRENT", "EXTERNAL_PIN_LITERAL"},
		{"agenttool.supabase-stack-baseline", "AGENTTOOL", "CURRENT_TRACKED", "https://github.com/cambridgetcg/agenttool", "796a753ab8624ad11af621ef4572544ea3b8f463", "docs/STACK.md", "sha256:207b66a20bf8dd34ac8637d9783858a7ae897afeb531285bbe5078f3bd890d43", "CURRENT", "EXTERNAL_PIN_LITERAL"},
	}
	effectFields = []string{
		"network", "storage", "database_read", "database_write", "agenttool_api_write",
		"hosted_route", "economic", "governance", "consensus", "identity", "permission",
		"authority_transfer", "karma", "nen", "score", "chain_read", "chain_write",
		"knowledge_admission", "scientific_adjudication", "wallet", "escrow", "payout",
		"reward", "zrn", "integration_ready",
	}
)

func verifyRepository(repositoryRoot string) (verificationReport, error) {
	root, err := os.OpenRoot(repositoryRoot)
	if err != nil {
		return verificationReport{}, fmt.Errorf("open repository root: %w", err)
	}
	defer root.Close()
	manifestBytes, err := readRootFile(root, manifestRelativePath, maxManifestBytes)
	if err != nil {
		return verificationReport{}, fmt.Errorf("read manifest: %w", err)
	}
	if err := validateManifestSeal(manifestBytes); err != nil {
		return verificationReport{}, err
	}
	var contract manifest
	nullable := []string{"$.pending_bindings[0].revision", "$.pending_bindings[0].path", "$.pending_bindings[0].raw_sha256"}
	if err := decodeClosed(manifestBytes, &contract, nullable...); err != nil {
		return verificationReport{}, fmt.Errorf("validate manifest JSON: %w", err)
	}
	if err := validateManifestShape(manifestBytes); err != nil {
		return verificationReport{}, fmt.Errorf("validate manifest shape: %w", err)
	}
	if err := validateManifest(contract); err != nil {
		return verificationReport{}, err
	}
	localPins, externalPins := 0, 0
	for _, pin := range contract.SourcePins {
		switch pin.Verification {
		case "LOCAL_BYTES":
			data, err := readRootFile(root, pin.Path, maxDocumentBytes)
			if err != nil {
				return verificationReport{}, fmt.Errorf("source pin %s unavailable: %w", pin.ID, err)
			}
			if actual := rawSHA256(data); actual != pin.RawSHA256 {
				return verificationReport{}, fmt.Errorf("source pin %s raw SHA-256 = %s, want %s", pin.ID, actual, pin.RawSHA256)
			}
			localPins++
		case "EXTERNAL_PIN_LITERAL":
			externalPins++
		default:
			return verificationReport{}, fmt.Errorf("source pin %s has unknown verification mode %q", pin.ID, pin.Verification)
		}
	}
	schemaBytes, err := readRootFile(root, contract.Schema.Path, maxSchemaBytes)
	if err != nil {
		return verificationReport{}, fmt.Errorf("read schema: %w", err)
	}
	if actual := rawSHA256(schemaBytes); actual != contract.Schema.RawSHA256 {
		return verificationReport{}, fmt.Errorf("schema raw SHA-256 = %s, want %s", actual, contract.Schema.RawSHA256)
	}
	if err := validateSchemaSurface(schemaBytes); err != nil {
		return verificationReport{}, fmt.Errorf("schema surface: %w", err)
	}
	report := verificationReport{
		Format: verificationFormat, Protocol: protocolID, Decision: "COHERENT_SOURCE_ONLY",
		ManifestRawSHA256: rawSHA256(manifestBytes), VerifiedLocalSourcePins: localPins,
		PinnedExternalSources: externalPins, PendingBindings: len(contract.PendingBindings),
		CandidateEffects: contract.CandidateEffects, ObservedValidatorEffects: contract.ValidatorExecution,
	}
	for _, fixture := range contract.Fixtures {
		data, err := readRootFile(root, fixture.Path, maxFixtureBytes)
		if err != nil {
			return verificationReport{}, fmt.Errorf("fixture %s unavailable: %w", fixture.ID, err)
		}
		if actual := rawSHA256(data); actual != fixture.RawSHA256 {
			return verificationReport{}, fmt.Errorf("fixture %s raw SHA-256 = %s, want %s", fixture.ID, actual, fixture.RawSHA256)
		}
		validation, validationErr := validateJournal(data)
		observedDecision := "ACCEPT"
		if validationErr != nil {
			observedDecision = "REJECT"
		}
		if observedDecision != fixture.ExpectedDecision {
			return verificationReport{}, fmt.Errorf("fixture %s decision = %s, want %s: %v", fixture.ID, observedDecision, fixture.ExpectedDecision, validationErr)
		}
		if observedDecision == "ACCEPT" && (validation.ObservationCount != fixture.ExpectedObservationCount || len(validation.ConflictGroups) != fixture.ExpectedConflictGroups) {
			return verificationReport{}, fmt.Errorf("fixture %s accepted with observations/conflicts %d/%d, want %d/%d", fixture.ID, validation.ObservationCount, len(validation.ConflictGroups), fixture.ExpectedObservationCount, fixture.ExpectedConflictGroups)
		}
		report.Fixtures = append(report.Fixtures, fixtureVerification{fixture.ID, fixture.ExpectedDecision, observedDecision, validation.ObservationCount, len(validation.ConflictGroups)})
	}
	return report, nil
}

func validateManifestSeal(data []byte) error {
	if actual := rawSHA256(data); actual != expectedManifestRawSHA256 {
		return fmt.Errorf("manifest raw SHA-256 = %s, want exact sealed bytes %s", actual, expectedManifestRawSHA256)
	}
	return nil
}

func validateManifest(contract manifest) error {
	if contract.Format != manifestFormat || contract.Protocol != protocolID || contract.Status != "SOURCE_ONLY_OFFLINE_NO_EFFECT" || contract.AsOf != "2026-08-21T09:42:59Z" {
		return errors.New("manifest identity or observation cutoff drifted")
	}
	if len(contract.SourcePins) != len(expectedSourcePins) {
		return fmt.Errorf("source pin count = %d, want %d", len(contract.SourcePins), len(expectedSourcePins))
	}
	for index, expected := range expectedSourcePins {
		if contract.SourcePins[index] != expected {
			return fmt.Errorf("source pin %d drifted: %#v", index, contract.SourcePins[index])
		}
	}
	if len(contract.PendingBindings) != 1 {
		return fmt.Errorf("pending binding count = %d, want 1", len(contract.PendingBindings))
	}
	pending := contract.PendingBindings[0]
	if pending.ID != "agenttool.supabase-research-shadow-hosting-profile" || pending.Owner != "AGENTTOOL" || pending.State != "PENDING_UNTRACKED" || pending.Repository != "https://github.com/cambridgetcg/agenttool" || pending.Revision != nil || pending.Path != nil || pending.RawSHA256 != nil || pending.AuthorityTransfer || pending.IntegrationReady {
		return fmt.Errorf("future AgentTool hosting binding must remain pending/null and non-authorizing: %#v", pending)
	}
	if contract.Schema.Path != "tools/zerone-supabase-observatory/protocol/observation-journal.v0.1.schema.json" || !digestPattern.MatchString(contract.Schema.RawSHA256) || contract.Schema.Dialect != "https://json-schema.org/draft/2020-12/schema" {
		return fmt.Errorf("schema descriptor drifted: %#v", contract.Schema)
	}
	expectedFixtures := []fixtureRef{
		{ID: "valid-current-observation", Path: "tools/zerone-supabase-observatory/testdata/valid-current-observation.json", ExpectedDecision: "ACCEPT", ExpectedObservationCount: 1},
		{ID: "same-height-conflicts-preserved", Path: "tools/zerone-supabase-observatory/testdata/same-height-conflicts-preserved.json", ExpectedDecision: "ACCEPT", ExpectedObservationCount: 2, ExpectedConflictGroups: 1},
		{ID: "truncated-unavailable-source-fails-closed", Path: "tools/zerone-supabase-observatory/testdata/truncated-unavailable-source-fails-closed.json", ExpectedDecision: "REJECT"},
	}
	if len(contract.Fixtures) != len(expectedFixtures) {
		return fmt.Errorf("fixture count = %d, want %d", len(contract.Fixtures), len(expectedFixtures))
	}
	for index, expected := range expectedFixtures {
		actual := contract.Fixtures[index]
		if actual.ID != expected.ID || actual.Path != expected.Path || actual.ExpectedDecision != expected.ExpectedDecision || actual.ExpectedObservationCount != expected.ExpectedObservationCount || actual.ExpectedConflictGroups != expected.ExpectedConflictGroups || !digestPattern.MatchString(actual.RawSHA256) {
			return fmt.Errorf("fixture descriptor %d drifted: %#v", index, actual)
		}
	}
	expectedSemantics := manifestSemantics{
		GraphKinds: []string{"STATIC_TREE", "TOK_ONCHAIN", "KNOWLEDGE_GEOMETRY"}, ToKHeightMode: "CURRENT_ONLY", ToKRequestAtBlockHeight: 0,
		ToKRootScope: "VERSIONED_TOPOLOGY_AND_LIFECYCLE_NOT_FULL_NODE_PAYLOAD", RawPayloadDigestRequired: true,
		SameHeightConflictPolicy: "PRESERVE_ALL_DISTINCT_OBSERVATIONS", UnavailableSourcePolicy: "FAIL_CLOSED_NO_SUBSTITUTION",
		SupabaseProjectionRelation: "PROJECTS_REBUILDABLE_READ_MODEL", DatabaseOrderingAuthority: "NONE", DatabaseTimestampAuthority: "NONE",
		ScientificAuthority: "NONE", ObservationIDCanonicalizing: "SHA256_DOMAIN_NUL_RECURSIVE_ASCII_KEY_SORTED_COMPACT_JSON_WITHOUT_ID",
	}
	if !equalSemantics(contract.Semantics, expectedSemantics) {
		return fmt.Errorf("manifest semantics drifted: %#v", contract.Semantics)
	}
	if contract.CandidateEffects != (candidateEffects{}) {
		return fmt.Errorf("candidate effect vector is non-zero: %#v", contract.CandidateEffects)
	}
	if contract.ValidatorExecution != (validatorExecution{LocalFileRead: "BOUNDED_EXPLICIT_REPOSITORY_FILES"}) {
		return fmt.Errorf("validator execution boundary drifted: %#v", contract.ValidatorExecution)
	}
	return nil
}

func equalSemantics(left, right manifestSemantics) bool {
	leftKinds, rightKinds := left.GraphKinds, right.GraphKinds
	left.GraphKinds, right.GraphKinds = nil, nil
	return reflect.DeepEqual(left, right) && equalStrings(leftKinds, rightKinds)
}

func validateManifestShape(data []byte) error {
	root, err := requireExactFields(data, "$",
		"_format", "protocol", "status", "as_of", "source_pins", "pending_bindings",
		"schema", "fixtures", "semantics", "candidate_effects", "validator_execution")
	if err != nil {
		return err
	}
	var sourcePins []json.RawMessage
	if err := json.Unmarshal(root["source_pins"], &sourcePins); err != nil {
		return fmt.Errorf("$.source_pins must be an array: %w", err)
	}
	for index, raw := range sourcePins {
		if _, err := requireExactFields(raw, fmt.Sprintf("$.source_pins[%d]", index),
			"id", "owner", "source_state", "repository", "revision", "path", "raw_sha256", "pin_role", "verification"); err != nil {
			return err
		}
	}
	var pending []json.RawMessage
	if err := json.Unmarshal(root["pending_bindings"], &pending); err != nil {
		return fmt.Errorf("$.pending_bindings must be an array: %w", err)
	}
	for index, raw := range pending {
		if _, err := requireExactFields(raw, fmt.Sprintf("$.pending_bindings[%d]", index),
			"id", "owner", "state", "repository", "revision", "path", "raw_sha256", "authority_transfer", "integration_ready"); err != nil {
			return err
		}
	}
	if _, err := requireExactFields(root["schema"], "$.schema", "path", "raw_sha256", "dialect"); err != nil {
		return err
	}
	var fixtures []json.RawMessage
	if err := json.Unmarshal(root["fixtures"], &fixtures); err != nil {
		return fmt.Errorf("$.fixtures must be an array: %w", err)
	}
	for index, raw := range fixtures {
		if _, err := requireExactFields(raw, fmt.Sprintf("$.fixtures[%d]", index),
			"id", "path", "raw_sha256", "expected_decision", "expected_observation_count", "expected_conflict_groups"); err != nil {
			return err
		}
	}
	if _, err := requireExactFields(root["semantics"], "$.semantics",
		"graph_kinds", "tok_height_mode", "tok_request_at_block_height", "tok_root_scope",
		"raw_payload_digest_required", "same_height_conflict_policy", "unavailable_source_policy",
		"supabase_projection_relation", "database_ordering_authority", "database_timestamp_authority",
		"scientific_authority", "observation_id_canonicalization"); err != nil {
		return err
	}
	if _, err := requireExactFields(root["candidate_effects"], "$.candidate_effects", effectFields...); err != nil {
		return err
	}
	_, err = requireExactFields(root["validator_execution"], "$.validator_execution",
		"network", "local_file_read", "local_file_write", "database", "chain", "economic",
		"governance", "identity", "permission", "karma", "nen", "score")
	return err
}

func validateJournalShape(data []byte) error {
	root, err := requireExactFields(data, "$", "_format", "mode", "projection", "observations", "effects")
	if err != nil {
		return err
	}
	if _, err := requireExactFields(root["projection"], "$.projection",
		"provider", "relation", "mode", "authority", "rebuildable", "database_order_is_chain_order",
		"database_time_is_trusted_time", "row_is_scientific_truth", "preserves", "loses"); err != nil {
		return err
	}
	if _, err := requireExactFields(root["effects"], "$.effects", effectFields...); err != nil {
		return err
	}
	var observations []json.RawMessage
	if err := json.Unmarshal(root["observations"], &observations); err != nil {
		return fmt.Errorf("$.observations must be an array: %w", err)
	}
	for index, raw := range observations {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return fmt.Errorf("$.observations[%d] must be an object: %w", index, err)
		}
		var kind string
		if err := json.Unmarshal(object["graph_kind"], &kind); err != nil {
			return fmt.Errorf("$.observations[%d].graph_kind is required", index)
		}
		path := fmt.Sprintf("$.observations[%d]", index)
		switch kind {
		case "TOK_ONCHAIN":
			fields, err := requireExactFields(raw, path,
				"observation_id", "graph_kind", "source_kind", "source_status", "observed_at", "request", "response")
			if err != nil {
				return err
			}
			if _, err := requireExactFields(fields["request"], path+".request", "at_block_height", "selector_sha256"); err != nil {
				return err
			}
			if _, err := requireExactFields(fields["response"], path+".response",
				"returned_chain_id", "returned_actual_block_height", "returned_block_hash", "returned_app_hash",
				"tok_snapshot_root", "tok_root_version", "raw_payload_sha256", "raw_payload_media_type",
				"raw_payload_complete", "proof_posture"); err != nil {
				return err
			}
		case "STATIC_TREE":
			fields, err := requireExactFields(raw, path,
				"observation_id", "graph_kind", "source_kind", "source_status", "observed_at", "source", "interpretation")
			if err != nil {
				return err
			}
			if _, err := requireExactFields(fields["source"], path+".source",
				"repository", "revision", "path", "raw_payload_sha256", "raw_payload_complete"); err != nil {
				return err
			}
			if _, err := requireExactFields(fields["interpretation"], path+".interpretation",
				"authoritative", "network_observed", "reward_bearing"); err != nil {
				return err
			}
		case "KNOWLEDGE_GEOMETRY":
			fields, err := requireExactFields(raw, path,
				"observation_id", "graph_kind", "source_kind", "source_status", "observed_at", "response")
			if err != nil {
				return err
			}
			if _, err := requireExactFields(fields["response"], path+".response",
				"returned_chain_id", "returned_actual_block_height", "raw_payload_sha256",
				"raw_payload_complete", "completeness", "truncated", "proof_posture"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s has unknown graph_kind %q", path, kind)
		}
	}
	return nil
}

func validateJournal(data []byte) (journalValidation, error) {
	var document journal
	if err := decodeClosed(data, &document); err != nil {
		return journalValidation{}, err
	}
	if err := validateJournalShape(data); err != nil {
		return journalValidation{}, err
	}
	if document.Format != journalFormat || document.Mode != "SOURCE_ONLY_SYNTHETIC_FIXTURE" {
		return journalValidation{}, errors.New("journal identity or mode drifted")
	}
	if err := validateProjection(document.Projection); err != nil {
		return journalValidation{}, err
	}
	if document.Effects != (candidateEffects{}) {
		return journalValidation{}, fmt.Errorf("journal effect vector is non-zero: %#v", document.Effects)
	}
	if len(document.Observations) == 0 || len(document.Observations) > 256 {
		return journalValidation{}, errors.New("observations count must be between 1 and 256")
	}
	tokGroups := make(map[string]map[string][]string)
	seenIDs := make(map[string]struct{}, len(document.Observations))
	for index, raw := range document.Observations {
		var tag observationTag
		if err := json.Unmarshal(raw, &tag); err != nil {
			return journalValidation{}, fmt.Errorf("observation[%d] tag: %w", index, err)
		}
		var id string
		switch tag.GraphKind {
		case "TOK_ONCHAIN":
			var observation tokObservation
			if err := decodeClosed(raw, &observation); err != nil {
				return journalValidation{}, fmt.Errorf("observation[%d]: %w", index, err)
			}
			if err := validateToKObservation(observation); err != nil {
				return journalValidation{}, fmt.Errorf("observation[%d]: %w", index, err)
			}
			id = observation.ObservationID
			key := strings.Join([]string{observation.GraphKind, observation.Response.ReturnedChainID, observation.Response.ReturnedActualBlockHeight}, "\x00")
			fingerprint := strings.Join([]string{observation.Response.ReturnedBlockHash, observation.Response.ReturnedAppHash, observation.Response.ToKSnapshotRoot, observation.Response.ToKRootVersion, observation.Response.RawPayloadSHA256}, "\x00")
			if tokGroups[key] == nil {
				tokGroups[key] = make(map[string][]string)
			}
			tokGroups[key][fingerprint] = append(tokGroups[key][fingerprint], id)
		case "STATIC_TREE":
			var observation staticTreeObservation
			if err := decodeClosed(raw, &observation); err != nil {
				return journalValidation{}, fmt.Errorf("observation[%d]: %w", index, err)
			}
			if err := validateStaticTreeObservation(observation); err != nil {
				return journalValidation{}, fmt.Errorf("observation[%d]: %w", index, err)
			}
			id = observation.ObservationID
		case "KNOWLEDGE_GEOMETRY":
			var observation knowledgeGeometryObservation
			if err := decodeClosed(raw, &observation); err != nil {
				return journalValidation{}, fmt.Errorf("observation[%d]: %w", index, err)
			}
			if err := validateKnowledgeGeometryObservation(observation); err != nil {
				return journalValidation{}, fmt.Errorf("observation[%d]: %w", index, err)
			}
			id = observation.ObservationID
		default:
			return journalValidation{}, fmt.Errorf("observation[%d] has unknown graph_kind %q", index, tag.GraphKind)
		}
		expectedID, err := observationID(raw)
		if err != nil {
			return journalValidation{}, fmt.Errorf("observation[%d] ID: %w", index, err)
		}
		if id != expectedID {
			return journalValidation{}, fmt.Errorf("observation[%d] ID = %s, want %s", index, id, expectedID)
		}
		if _, exists := seenIDs[id]; exists {
			return journalValidation{}, fmt.Errorf("duplicate observation_id %s", id)
		}
		seenIDs[id] = struct{}{}
	}
	result := journalValidation{ObservationCount: len(document.Observations)}
	groupKeys := make([]string, 0, len(tokGroups))
	for key, fingerprints := range tokGroups {
		if len(fingerprints) > 1 {
			groupKeys = append(groupKeys, key)
		}
	}
	sort.Strings(groupKeys)
	for _, key := range groupKeys {
		parts := strings.Split(key, "\x00")
		ids := make([]string, 0)
		for _, fingerprintIDs := range tokGroups[key] {
			ids = append(ids, fingerprintIDs...)
		}
		sort.Strings(ids)
		result.ConflictGroups = append(result.ConflictGroups, conflictGroup{parts[0], parts[1], parts[2], ids})
	}
	return result, nil
}

func validateProjection(value projection) error {
	expected := projection{
		Provider: "SUPABASE_POSTGRESQL", Relation: "PROJECTS", Mode: "SCHEMA_DESIGN_ONLY", Authority: "NONE", Rebuildable: true,
		Preserves: []string{"GRAPH_KIND", "SOURCE_STATUS", "EXACT_RESPONSE_DIGESTS", "RETURNED_CHAIN_AND_HEIGHT", "SAME_HEIGHT_CONFLICT_MULTIPLICITY"},
		Loses:     []string{"RAW_PAYLOAD_BYTES", "SOURCE_ENDPOINT", "CHAIN_PROOF", "SCIENTIFIC_TRUTH", "IDENTITY_AND_CONTROLLER", "CONSENT_AND_AUTHORITY"},
	}
	if value.Provider != expected.Provider || value.Relation != expected.Relation || value.Mode != expected.Mode || value.Authority != expected.Authority || value.Rebuildable != expected.Rebuildable || value.DatabaseOrderIsChainOrder || value.DatabaseTimeIsTrustedTime || value.RowIsScientificTruth || !equalStrings(value.Preserves, expected.Preserves) || !equalStrings(value.Loses, expected.Loses) {
		return fmt.Errorf("Supabase projection boundary drifted: %#v", value)
	}
	return nil
}

func validateToKObservation(value tokObservation) error {
	if value.GraphKind != "TOK_ONCHAIN" || value.SourceKind != "ZERONE_CURRENT_ONLY_QUERY" {
		return errors.New("ToK observation graph/source kind drifted")
	}
	if value.SourceStatus != "COMPLETE" {
		return fmt.Errorf("ToK source_status %q fails closed", value.SourceStatus)
	}
	if err := validateTimestamp(value.ObservedAt); err != nil {
		return err
	}
	if !digestPattern.MatchString(value.ObservationID) || !digestPattern.MatchString(value.Request.SelectorSHA256) {
		return errors.New("ToK observation or selector digest is malformed")
	}
	if value.Request.AtBlockHeight != 0 {
		return fmt.Errorf("ToK at_block_height must be 0, got %d", value.Request.AtBlockHeight)
	}
	r := value.Response
	if !chainIDPattern.MatchString(r.ReturnedChainID) || !validHeight(r.ReturnedActualBlockHeight) || !hex32Pattern.MatchString(r.ReturnedBlockHash) || !hex32Pattern.MatchString(r.ReturnedAppHash) || !hex32Pattern.MatchString(r.ToKSnapshotRoot) || !digestPattern.MatchString(r.RawPayloadSHA256) {
		return errors.New("ToK response identifiers or digests are malformed")
	}
	if r.ToKRootVersion != "v1" && r.ToKRootVersion != "v2" {
		return fmt.Errorf("unsupported tok_root_version %q", r.ToKRootVersion)
	}
	if r.RawPayloadMediaType != "application/x-ndjson" || !r.RawPayloadComplete {
		return errors.New("ToK raw payload must be a complete application/x-ndjson digest source")
	}
	if r.ProofPosture != "SYNTHETIC_UNVERIFIED_RESPONSE" && r.ProofPosture != "UNVERIFIED_REMOTE_RESPONSE" {
		return fmt.Errorf("unsupported ToK proof_posture %q", r.ProofPosture)
	}
	return nil
}

func validateStaticTreeObservation(value staticTreeObservation) error {
	if value.GraphKind != "STATIC_TREE" || value.SourceKind != "REPOSITORY_BYTES" || value.SourceStatus != "COMPLETE" {
		return errors.New("static Tree graph/source/status drifted")
	}
	if err := validateTimestamp(value.ObservedAt); err != nil {
		return err
	}
	if !digestPattern.MatchString(value.ObservationID) ||
		value.Source.Repository != staticTreeRepository ||
		value.Source.Revision != staticTreeRevision ||
		value.Source.Path != staticTreePath ||
		value.Source.RawPayloadSHA256 != staticTreeRawPayloadSHA256 ||
		!value.Source.RawPayloadComplete {
		return errors.New("static Tree source must equal the exact pinned Zerone Tree bytes")
	}
	if value.Interpretation != (staticTreeInterpretation{}) {
		return errors.New("static Tree interpretation must remain non-authoritative, unobserved, and non-reward-bearing")
	}
	return nil
}

func validateKnowledgeGeometryObservation(value knowledgeGeometryObservation) error {
	if value.GraphKind != "KNOWLEDGE_GEOMETRY" || value.SourceKind != "ZERONE_BOUNDED_READ_PROJECTION" || value.SourceStatus != "COMPLETE" {
		return errors.New("Knowledge Geometry graph/source/status drifted")
	}
	if err := validateTimestamp(value.ObservedAt); err != nil {
		return err
	}
	r := value.Response
	if !digestPattern.MatchString(value.ObservationID) || r.ReturnedChainID != knowledgeGeometryChainID || !validHeight(r.ReturnedActualBlockHeight) || !digestPattern.MatchString(r.RawPayloadSHA256) || !r.RawPayloadComplete || r.Completeness != "NOT_CLAIMED" || r.Truncated {
		return errors.New("Knowledge Geometry response is incomplete, truncated, or malformed")
	}
	if r.ProofPosture != "SYNTHETIC_UNVERIFIED_RESPONSE" && r.ProofPosture != "UNVERIFIED_REMOTE_RESPONSE" {
		return fmt.Errorf("unsupported Knowledge Geometry proof_posture %q", r.ProofPosture)
	}
	return nil
}

func validateTimestamp(value string) error {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Format(time.RFC3339) != value || !strings.HasSuffix(value, "Z") {
		return fmt.Errorf("timestamp %q must be canonical UTC RFC3339 without fractional seconds", value)
	}
	return nil
}

func validHeight(value string) bool {
	if !heightPattern.MatchString(value) {
		return false
	}
	_, err := strconv.ParseUint(value, 10, 64)
	return err == nil
}

func validRelativePath(value string) bool {
	return value != "" && filepath.IsLocal(value) && filepath.ToSlash(filepath.Clean(value)) == value && !strings.Contains(value, `\`)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validateSchemaSurface(data []byte) error {
	if err := inspectJSON(data, nil); err != nil {
		return err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	var dialect, id, documentType string
	var additionalProperties bool
	if err := json.Unmarshal(document["$schema"], &dialect); err != nil || dialect != "https://json-schema.org/draft/2020-12/schema" {
		return errors.New("schema dialect is not JSON Schema 2020-12")
	}
	if err := json.Unmarshal(document["$id"], &id); err != nil || id != "urn:zerone:agenttool:supabase:observatory:journal:0.1" {
		return errors.New("schema ID drifted")
	}
	if err := json.Unmarshal(document["type"], &documentType); err != nil || documentType != "object" {
		return errors.New("schema root must be an object")
	}
	if err := json.Unmarshal(document["additionalProperties"], &additionalProperties); err != nil || additionalProperties {
		return errors.New("schema root must reject additional properties")
	}
	for _, required := range []string{"required", "properties", "$defs"} {
		if len(document[required]) == 0 {
			return fmt.Errorf("schema is missing %s", required)
		}
	}
	return nil
}

func readRootFile(root *os.Root, relativePath string, maximum int64) ([]byte, error) {
	if maximum < 0 || !validRelativePath(relativePath) {
		return nil, fmt.Errorf("invalid bounded repository path %q", relativePath)
	}
	parts := strings.Split(relativePath, "/")
	for index := range parts {
		component := strings.Join(parts[:index+1], "/")
		info, err := root.Lstat(component)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s contains a symbolic link", relativePath)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("%s contains a non-directory component", relativePath)
		}
	}
	before, err := root.Lstat(relativePath)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > maximum {
		return nil, fmt.Errorf("%s is not a bounded regular file", relativePath)
	}
	file, err := root.Open(relativePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() > maximum {
		return nil, fmt.Errorf("%s changed while opening", relativePath)
	}
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", relativePath, maximum)
	}
	afterDescriptor, err := file.Stat()
	if err != nil {
		return nil, err
	}
	afterPath, err := root.Lstat(relativePath)
	if err != nil || !os.SameFile(opened, afterDescriptor) || !os.SameFile(afterDescriptor, afterPath) || afterDescriptor.Size() != int64(len(data)) || afterDescriptor.ModTime() != opened.ModTime() {
		return nil, fmt.Errorf("%s changed while reading", relativePath)
	}
	return data, nil
}
