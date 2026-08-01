package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	manifestSchema         = "zerone.validator-home-manifest/v1"
	stoppedEvidenceSchema  = "zerone.validator-source-stopped-evidence/v1"
	restartEvidenceSchema  = "zerone.validator-restart-inhibit-evidence/v1"
	snapshotEvidenceSchema = "zerone.validator-read-only-snapshot-evidence/v1"
	volumeEvidenceSchema   = "zerone.validator-volume-control-evidence/v1"
	maxControlFileBytes    = 32 << 20
	maxJSONDepth           = 128
	maxJSONEntries         = 2_000_000
)

type StoppedEvidence struct {
	Schema                       string `json:"schema"`
	CapturedAt                   string `json:"captured_at"`
	Method                       string `json:"method"`
	Observer                     string `json:"observer"`
	SourceHome                   string `json:"source_home"`
	ProcessID                    int    `json:"process_id"`
	ProcessStartTime             string `json:"process_start_time"`
	ProcessIdentitySHA256        string `json:"process_identity_sha256"`
	ProcessAbsent                bool   `json:"process_absent"`
	RestartInhibitEvidenceSHA256 string `json:"restart_inhibit_evidence_sha256"`
	LastHeight                   int64  `json:"last_height"`
	AppHash                      string `json:"app_hash"`
	EvidenceSHA256               string `json:"evidence_sha256"`
}

type RestartInhibitEvidence struct {
	Schema                string `json:"schema"`
	CapturedAt            string `json:"captured_at"`
	SourceHome            string `json:"source_home"`
	ProcessStartTime      string `json:"process_start_time"`
	ProcessIdentitySHA256 string `json:"process_identity_sha256"`
	Supervisor            string `json:"supervisor"`
	Unit                  string `json:"unit"`
	RestartDisabled       bool   `json:"restart_disabled"`
	StartBlocked          bool   `json:"start_blocked"`
	EvidenceSHA256        string `json:"evidence_sha256"`
}

type SnapshotEvidence struct {
	Schema         string `json:"schema"`
	CapturedAt     string `json:"captured_at"`
	SourceHome     string `json:"source_home"`
	SnapshotID     string `json:"snapshot_id"`
	Method         string `json:"method"`
	ReadOnly       bool   `json:"read_only"`
	Isolated       bool   `json:"isolated"`
	IncludesACLs   bool   `json:"includes_acls"`
	IncludesXAttrs bool   `json:"includes_xattrs"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}

type VolumeEvidence struct {
	Schema              string `json:"schema"`
	CapturedAt          string `json:"captured_at"`
	DestinationVolumeID string `json:"destination_volume_id"`
	Provider            string `json:"provider"`
	ImmutableID         bool   `json:"immutable_id"`
	EncryptionAtRest    bool   `json:"encryption_at_rest"`
	Fresh               bool   `json:"fresh"`
	EmptyBeforeRestore  bool   `json:"empty_before_restore"`
	EvidenceSHA256      string `json:"evidence_sha256"`
}

type FilesystemIdentity struct {
	CanonicalPath string `json:"canonical_path"`
	DeviceID      uint64 `json:"device_id"`
	RootInode     uint64 `json:"root_inode"`
	ReadOnly      bool   `json:"read_only"`
}

type SnapshotBinding struct {
	SnapshotID     string `json:"snapshot_id"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}

type DestinationBinding struct {
	VolumeID                string `json:"volume_id"`
	VolumeEvidenceSHA256    string `json:"volume_evidence_sha256"`
	ObservedEmptyBeforeCopy bool   `json:"observed_empty_before_copy"`
}

type FileRecord struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Mode   string `json:"mode"`
	SHA256 string `json:"sha256"`
}

type DirectoryRecord struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type ContentManifest struct {
	Directories []DirectoryRecord `json:"directories"`
	Files       []FileRecord      `json:"files"`
	SHA256      string            `json:"sha256"`
}

type DatabaseManifest struct {
	Name   string       `json:"name"`
	Root   string       `json:"root"`
	Files  []FileRecord `json:"files"`
	SHA256 string       `json:"sha256"`
}

type ChainIdentity struct {
	ChainID       string `json:"chain_id"`
	InitialHeight int64  `json:"initial_height"`
	GenesisSHA256 string `json:"genesis_sha256"`
}

type ConsensusIdentity struct {
	ConsensusAddress      string `json:"consensus_address"`
	ConsensusKeyType      string `json:"consensus_key_type"`
	ConsensusPubKeySHA256 string `json:"consensus_public_key_sha256"`
	NodeID                string `json:"node_id"`
	NodeKeyType           string `json:"node_key_type"`
	NodePublicKeySHA256   string `json:"node_public_key_sha256"`
}

type SigningState struct {
	Height           int64  `json:"height"`
	Round            int64  `json:"round"`
	Step             int64  `json:"step"`
	SignaturePresent bool   `json:"signature_present"`
	SignatureSHA256  string `json:"signature_sha256"`
	SignBytesPresent bool   `json:"sign_bytes_present"`
	SignBytesSHA256  string `json:"sign_bytes_sha256"`
	StateFileSHA256  string `json:"state_file_sha256"`
}

type ManifestStoppedEvidence struct {
	CapturedAt                   string `json:"captured_at"`
	Method                       string `json:"method"`
	Observer                     string `json:"observer"`
	ProcessID                    int    `json:"process_id"`
	ProcessStartTime             string `json:"process_start_time"`
	ProcessIdentitySHA256        string `json:"process_identity_sha256"`
	ProcessAbsent                bool   `json:"process_absent"`
	RestartInhibitEvidenceSHA256 string `json:"restart_inhibit_evidence_sha256"`
	EvidenceSHA256               string `json:"evidence_sha256"`
}

type ValidatorHomeManifest struct {
	Schema                string                  `json:"schema"`
	SourceHome            string                  `json:"source_home"`
	SourceFilesystem      FilesystemIdentity      `json:"source_filesystem"`
	DestinationFilesystem FilesystemIdentity      `json:"destination_filesystem"`
	Destination           DestinationBinding      `json:"destination"`
	Snapshot              SnapshotBinding         `json:"snapshot"`
	Chain                 ChainIdentity           `json:"chain"`
	LastHeight            int64                   `json:"last_height"`
	AppHash               string                  `json:"app_hash"`
	Consensus             ConsensusIdentity       `json:"consensus"`
	SigningState          SigningState            `json:"signing_state"`
	StoppedEvidence       ManifestStoppedEvidence `json:"stopped_evidence"`
	Databases             []DatabaseManifest      `json:"databases"`
	Content               ContentManifest         `json:"content"`
	ManifestSHA256        string                  `json:"manifest_sha256"`
}

func canonicalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}
	return data, nil
}

func canonicalDocument(value any) ([]byte, error) {
	data, err := canonicalJSON(value)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func hashCanonical(value any) (string, error) {
	data, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func decodeStrict(data []byte, destination any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailer any
	if err := decoder.Decode(&trailer); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode JSON: unexpected value after root")
		}
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSON(decoder, "$", 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("schema ambiguity: unexpected value after root")
		}
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return nil
}

func walkJSON(decoder *json.Decoder, path string, depth int) error {
	if depth > maxJSONDepth {
		return fmt.Errorf("schema ambiguity: JSON nesting exceeds %d levels at %s", maxJSONDepth, path)
	}
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode JSON at %s: %w", path, err)
	}
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		count := 0
		for decoder.More() {
			if count >= maxJSONEntries {
				return fmt.Errorf("schema ambiguity: object at %s exceeds %d fields", path, maxJSONEntries)
			}
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("decode object key at %s: %w", path, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("decode object key at %s: key is not a string", path)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("schema ambiguity: duplicate JSON key %q at %s", key, path)
			}
			seen[key] = struct{}{}
			if err := walkJSON(decoder, path+"."+key, depth+1); err != nil {
				return err
			}
			count++
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
			return fmt.Errorf("decode object end at %s", path)
		}
	case '[':
		index := 0
		for decoder.More() {
			if index >= maxJSONEntries {
				return fmt.Errorf("schema ambiguity: array at %s exceeds %d entries", path, maxJSONEntries)
			}
			if err := walkJSON(decoder, fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
				return err
			}
			index++
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
			return fmt.Errorf("decode array end at %s", path)
		}
	default:
		return fmt.Errorf("decode JSON at %s: unexpected delimiter %q", path, delim)
	}
	return nil
}

func validateSHA256(label, value string) error {
	if len(value) != 64 || strings.ToLower(value) != value {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", label)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", label)
	}
	return nil
}

func validateAppHash(value string) error {
	if err := validateSHA256("app hash", value); err != nil {
		return err
	}
	return nil
}

func validateCanonicalTime(label, value string) error {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return fmt.Errorf("%s must be RFC3339: %w", label, err)
	}
	if parsed.Location() != time.UTC || parsed.UTC().Format(time.RFC3339Nano) != value {
		return fmt.Errorf("%s must be canonical UTC RFC3339 using Z", label)
	}
	return nil
}
