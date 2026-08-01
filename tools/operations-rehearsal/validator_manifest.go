package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

const validatorHomeManifestSchema = "zerone.validator-home-manifest/v1"

const (
	validatorStoppedEvidenceSchema = "zerone.validator-source-stopped-evidence/v1"
	validatorEd25519PublicKeyType  = "tendermint/PubKeyEd25519"
	validatorEd25519PrivateKeyType = "tendermint/PrivKeyEd25519"
)

var fileModePattern = regexp.MustCompile(`^[0-7]{4}$`)

type ValidatorFileRecord struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Mode   string `json:"mode"`
	SHA256 string `json:"sha256"`
}

type ValidatorDirectoryRecord struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type ValidatorContentManifest struct {
	Directories []ValidatorDirectoryRecord `json:"directories"`
	Files       []ValidatorFileRecord      `json:"files"`
	SHA256      string                     `json:"sha256"`
}

type ValidatorDatabaseManifest struct {
	Name   string                `json:"name"`
	Root   string                `json:"root"`
	Files  []ValidatorFileRecord `json:"files"`
	SHA256 string                `json:"sha256"`
}

type ValidatorChainIdentity struct {
	ChainID       string `json:"chain_id"`
	InitialHeight int64  `json:"initial_height"`
	GenesisSHA256 string `json:"genesis_sha256"`
}

type ValidatorConsensusIdentity struct {
	ConsensusAddress      string `json:"consensus_address"`
	ConsensusKeyType      string `json:"consensus_key_type"`
	ConsensusPubKeySHA256 string `json:"consensus_public_key_sha256"`
	NodeID                string `json:"node_id"`
	NodeKeyType           string `json:"node_key_type"`
	NodePublicKeySHA256   string `json:"node_public_key_sha256"`
}

type ValidatorSigningState struct {
	Height           int64  `json:"height"`
	Round            int64  `json:"round"`
	Step             int64  `json:"step"`
	SignaturePresent bool   `json:"signature_present"`
	SignatureSHA256  string `json:"signature_sha256"`
	SignBytesPresent bool   `json:"sign_bytes_present"`
	SignBytesSHA256  string `json:"sign_bytes_sha256"`
	StateFileSHA256  string `json:"state_file_sha256"`
}

type ValidatorStoppedEvidence struct {
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

type ValidatorStoppedEvidenceDocument struct {
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

type ValidatorFilesystemIdentity struct {
	CanonicalPath string `json:"canonical_path"`
	DeviceID      uint64 `json:"device_id"`
	RootInode     uint64 `json:"root_inode"`
	ReadOnly      bool   `json:"read_only"`
}

type ValidatorSnapshotBinding struct {
	SnapshotID     string `json:"snapshot_id"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}

type ValidatorDestinationBinding struct {
	VolumeID                string `json:"volume_id"`
	VolumeEvidenceSHA256    string `json:"volume_evidence_sha256"`
	ObservedEmptyBeforeCopy bool   `json:"observed_empty_before_copy"`
}

type FullValidatorHomeManifest struct {
	Schema                string                      `json:"schema"`
	SourceHome            string                      `json:"source_home"`
	SourceFilesystem      ValidatorFilesystemIdentity `json:"source_filesystem"`
	DestinationFilesystem ValidatorFilesystemIdentity `json:"destination_filesystem"`
	Destination           ValidatorDestinationBinding `json:"destination"`
	Snapshot              ValidatorSnapshotBinding    `json:"snapshot"`
	Chain                 ValidatorChainIdentity      `json:"chain"`
	LastHeight            int64                       `json:"last_height"`
	AppHash               string                      `json:"app_hash"`
	Consensus             ValidatorConsensusIdentity  `json:"consensus"`
	SigningState          ValidatorSigningState       `json:"signing_state"`
	StoppedEvidence       ValidatorStoppedEvidence    `json:"stopped_evidence"`
	Databases             []ValidatorDatabaseManifest `json:"databases"`
	Content               ValidatorContentManifest    `json:"content"`
	ManifestSHA256        string                      `json:"manifest_sha256"`
}

var validatorDatabaseRoots = []struct {
	name string
	root string
}{
	{name: "application", root: "data/application.db"},
	{name: "blockstore", root: "data/blockstore.db"},
	{name: "comet_state", root: "data/state.db"},
}

func loadValidatorHomeManifest(reference EvidenceRef, root string) (FullValidatorHomeManifest, error) {
	if reference.Kind != "source-home-manifest" ||
		reference.MediaType != "application/json" {
		return FullValidatorHomeManifest{}, fmt.Errorf("source-home-manifest must be canonical application/json")
	}
	data, err := readVerifiedEvidence(root, reference)
	if err != nil {
		return FullValidatorHomeManifest{}, fmt.Errorf("read source home manifest: %w", err)
	}
	var manifest FullValidatorHomeManifest
	if err := decodeStrict(data, &manifest); err != nil {
		return FullValidatorHomeManifest{}, fmt.Errorf("source home manifest: %w", err)
	}
	if err := validateFullValidatorHomeManifest(manifest); err != nil {
		return FullValidatorHomeManifest{}, err
	}
	forHash := manifest
	forHash.ManifestSHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return FullValidatorHomeManifest{}, err
	}
	if digest != manifest.ManifestSHA256 {
		return FullValidatorHomeManifest{}, fmt.Errorf("source home manifest self-hash mismatch")
	}
	canonical, err := canonicalDocument(manifest)
	if err != nil {
		return FullValidatorHomeManifest{}, err
	}
	if !bytes.Equal(data, canonical) {
		return FullValidatorHomeManifest{}, fmt.Errorf("source home manifest is not exact canonical JSON")
	}
	return manifest, nil
}

func validateFullValidatorHomeManifest(manifest FullValidatorHomeManifest) error {
	if manifest.Schema != validatorHomeManifestSchema {
		return fmt.Errorf("source home manifest schema must be %q", validatorHomeManifestSchema)
	}
	if manifest.SourceHome == "" || !filepath.IsAbs(manifest.SourceHome) ||
		filepath.Clean(manifest.SourceHome) != manifest.SourceHome ||
		manifest.SourceFilesystem.CanonicalPath != manifest.SourceHome ||
		!manifest.SourceFilesystem.ReadOnly ||
		manifest.SourceFilesystem.DeviceID == 0 ||
		manifest.DestinationFilesystem.CanonicalPath == "" ||
		!filepath.IsAbs(manifest.DestinationFilesystem.CanonicalPath) ||
		filepath.Clean(manifest.DestinationFilesystem.CanonicalPath) !=
			manifest.DestinationFilesystem.CanonicalPath ||
		manifest.DestinationFilesystem.CanonicalPath == manifest.SourceHome ||
		manifest.DestinationFilesystem.ReadOnly ||
		manifest.DestinationFilesystem.DeviceID == 0 ||
		manifest.SourceFilesystem.DeviceID == manifest.DestinationFilesystem.DeviceID ||
		manifest.SourceFilesystem.RootInode == 0 ||
		manifest.DestinationFilesystem.RootInode == 0 ||
		manifest.Destination.VolumeID == "" ||
		strings.TrimSpace(manifest.Destination.VolumeID) != manifest.Destination.VolumeID ||
		!manifest.Destination.ObservedEmptyBeforeCopy ||
		manifest.Snapshot.SnapshotID == "" {
		return fmt.Errorf("source home manifest source path and destination volume are incomplete")
	}
	if err := validateSHA256(
		"source volume-control evidence SHA-256",
		manifest.Destination.VolumeEvidenceSHA256,
	); err != nil {
		return err
	}
	if err := validateSHA256(
		"source snapshot evidence SHA-256",
		manifest.Snapshot.EvidenceSHA256,
	); err != nil {
		return err
	}
	if manifest.Chain.ChainID == "" || manifest.Chain.InitialHeight <= 0 {
		return fmt.Errorf("source home manifest chain identity is incomplete")
	}
	if err := validateSHA256("source genesis SHA-256", manifest.Chain.GenesisSHA256); err != nil {
		return err
	}
	if manifest.LastHeight <= 0 || manifest.LastHeight == math.MaxInt64 {
		return fmt.Errorf("source home manifest last height must permit H+1")
	}
	if err := validateSHA256("source AppHash", manifest.AppHash); err != nil {
		return err
	}
	if len(manifest.Consensus.ConsensusAddress) != 40 ||
		strings.ToUpper(manifest.Consensus.ConsensusAddress) != manifest.Consensus.ConsensusAddress {
		return fmt.Errorf("source consensus address must be 40 uppercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(manifest.Consensus.ConsensusAddress); err != nil {
		return fmt.Errorf("source consensus address is not hexadecimal")
	}
	if len(manifest.Consensus.NodeID) != 40 ||
		strings.ToLower(manifest.Consensus.NodeID) != manifest.Consensus.NodeID {
		return fmt.Errorf("source node ID must be 40 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(manifest.Consensus.NodeID); err != nil {
		return fmt.Errorf("source node ID is not hexadecimal")
	}
	if manifest.Consensus.ConsensusKeyType != validatorEd25519PublicKeyType ||
		manifest.Consensus.NodeKeyType != validatorEd25519PrivateKeyType {
		return fmt.Errorf("source consensus key types are not canonical Ed25519 types")
	}
	for label, digest := range map[string]string{
		"source consensus public key": manifest.Consensus.ConsensusPubKeySHA256,
		"source node public key":      manifest.Consensus.NodePublicKeySHA256,
		"source signing-state file":   manifest.SigningState.StateFileSHA256,
		"source stopped evidence":     manifest.StoppedEvidence.EvidenceSHA256,
		"source process identity":     manifest.StoppedEvidence.ProcessIdentitySHA256,
		"source restart inhibit":      manifest.StoppedEvidence.RestartInhibitEvidenceSHA256,
		"source manifest":             manifest.ManifestSHA256,
	} {
		if err := validateSHA256(label+" SHA-256", digest); err != nil {
			return err
		}
	}
	if manifest.SigningState.Height < 0 || manifest.SigningState.Round < 0 ||
		manifest.SigningState.Step < 0 || manifest.SigningState.Step > 3 {
		return fmt.Errorf("source signing-state coordinates are outside Comet bounds")
	}
	if manifest.SigningState.SignaturePresent != manifest.SigningState.SignBytesPresent {
		return fmt.Errorf("source signing-state signature and sign bytes must be present together")
	}
	if manifest.SigningState.Height == 0 {
		if manifest.SigningState.Round != 0 || manifest.SigningState.Step != 0 ||
			manifest.SigningState.SignaturePresent {
			return fmt.Errorf("source genesis signing state is inconsistent")
		}
	} else if manifest.SigningState.Step < 1 ||
		!manifest.SigningState.SignaturePresent {
		return fmt.Errorf("source non-genesis signing state lacks a complete signed step")
	}
	if manifest.SigningState.Height > manifest.LastHeight+1 {
		return fmt.Errorf("source signing state cannot be beyond last_height + 1")
	}
	if err := validateOptionalDigest(
		"source signature",
		manifest.SigningState.SignaturePresent,
		manifest.SigningState.SignatureSHA256,
	); err != nil {
		return err
	}
	if err := validateOptionalDigest(
		"source sign bytes",
		manifest.SigningState.SignBytesPresent,
		manifest.SigningState.SignBytesSHA256,
	); err != nil {
		return err
	}
	stoppedCaptured, err := validateCanonicalTime(
		"source stopped evidence captured_at",
		manifest.StoppedEvidence.CapturedAt,
	)
	if err != nil {
		return err
	}
	processStart, err := validateCanonicalTime(
		"source stopped evidence process_start_time",
		manifest.StoppedEvidence.ProcessStartTime,
	)
	if err != nil {
		return err
	}
	if !processStart.Before(stoppedCaptured) {
		return fmt.Errorf("source process start time must precede stopped evidence capture")
	}
	if manifest.StoppedEvidence.Method == "" || manifest.StoppedEvidence.Observer == "" ||
		strings.TrimSpace(manifest.StoppedEvidence.Method) != manifest.StoppedEvidence.Method ||
		strings.TrimSpace(manifest.StoppedEvidence.Observer) != manifest.StoppedEvidence.Observer ||
		manifest.StoppedEvidence.ProcessID <= 1 || !manifest.StoppedEvidence.ProcessAbsent {
		return fmt.Errorf("source stopped evidence is incomplete")
	}
	stopped := ValidatorStoppedEvidenceDocument{
		Schema:                       validatorStoppedEvidenceSchema,
		CapturedAt:                   manifest.StoppedEvidence.CapturedAt,
		Method:                       manifest.StoppedEvidence.Method,
		Observer:                     manifest.StoppedEvidence.Observer,
		SourceHome:                   manifest.SourceHome,
		ProcessID:                    manifest.StoppedEvidence.ProcessID,
		ProcessStartTime:             manifest.StoppedEvidence.ProcessStartTime,
		ProcessIdentitySHA256:        manifest.StoppedEvidence.ProcessIdentitySHA256,
		ProcessAbsent:                manifest.StoppedEvidence.ProcessAbsent,
		RestartInhibitEvidenceSHA256: manifest.StoppedEvidence.RestartInhibitEvidenceSHA256,
		LastHeight:                   manifest.LastHeight,
		AppHash:                      manifest.AppHash,
		EvidenceSHA256:               manifest.StoppedEvidence.EvidenceSHA256,
	}
	stoppedForHash := stopped
	stoppedForHash.EvidenceSHA256 = ""
	stoppedDigest, err := hashCanonical(stoppedForHash)
	if err != nil {
		return err
	}
	if stoppedDigest != manifest.StoppedEvidence.EvidenceSHA256 {
		return fmt.Errorf("source stopped evidence self-hash mismatch")
	}
	if len(manifest.Databases) != len(validatorDatabaseRoots) {
		return fmt.Errorf("source home manifest must contain exactly three database groups")
	}
	contentFiles := make(map[string]ValidatorFileRecord, len(manifest.Content.Files))
	if err := validateValidatorContent(manifest.Content, contentFiles); err != nil {
		return err
	}
	for index, expected := range validatorDatabaseRoots {
		database := manifest.Databases[index]
		if database.Name != expected.name || database.Root != expected.root ||
			len(database.Files) == 0 {
			return fmt.Errorf("source database group %d must be %s at %s", index, expected.name, expected.root)
		}
		previous := ""
		for _, file := range database.Files {
			if previous != "" && file.Path <= previous {
				return fmt.Errorf("source database %s files are not strictly sorted", database.Name)
			}
			previous = file.Path
			if !strings.HasPrefix(file.Path, database.Root+"/") {
				return fmt.Errorf("source database %s contains out-of-root path %s", database.Name, file.Path)
			}
			contentFile, exists := contentFiles[file.Path]
			if !exists || !reflect.DeepEqual(contentFile, file) {
				return fmt.Errorf("source database file %s does not match full content", file.Path)
			}
		}
		expectedFiles := make([]ValidatorFileRecord, 0)
		for _, contentFile := range manifest.Content.Files {
			if strings.HasPrefix(contentFile.Path, database.Root+"/") {
				expectedFiles = append(expectedFiles, contentFile)
			}
		}
		if !reflect.DeepEqual(database.Files, expectedFiles) {
			return fmt.Errorf(
				"source database %s manifest does not contain its exact full-content file set",
				database.Name,
			)
		}
		forHash := database
		forHash.SHA256 = ""
		digest, err := hashCanonical(forHash)
		if err != nil {
			return err
		}
		if digest != database.SHA256 {
			return fmt.Errorf("source database %s manifest self-hash mismatch", database.Name)
		}
	}
	return nil
}

func validateValidatorContent(
	content ValidatorContentManifest,
	files map[string]ValidatorFileRecord,
) error {
	if len(content.Directories) == 0 || len(content.Files) == 0 {
		return fmt.Errorf("source full content manifest is empty")
	}
	previous := ""
	directories := make(map[string]struct{}, len(content.Directories))
	for _, directory := range content.Directories {
		if err := validateRelativePath(directory.Path); err != nil {
			return err
		}
		if previous != "" && directory.Path <= previous {
			return fmt.Errorf("source content directories are not strictly sorted")
		}
		if !fileModePattern.MatchString(directory.Mode) {
			return fmt.Errorf("source directory %s has non-canonical mode", directory.Path)
		}
		previous = directory.Path
		directories[directory.Path] = struct{}{}
	}
	for _, required := range validatorDatabaseRoots {
		if _, exists := directories[required.root]; !exists {
			return fmt.Errorf("source content is missing database directory %s", required.root)
		}
	}
	for directory := range directories {
		parent := filepath.ToSlash(filepath.Dir(directory))
		if parent != "." {
			if _, exists := directories[parent]; !exists {
				return fmt.Errorf(
					"source content directory %s has missing parent %s",
					directory,
					parent,
				)
			}
		}
	}
	previous = ""
	for _, file := range content.Files {
		if err := validateRelativePath(file.Path); err != nil {
			return err
		}
		if previous != "" && file.Path <= previous {
			return fmt.Errorf("source content files are not strictly sorted")
		}
		if file.Size < 0 || !fileModePattern.MatchString(file.Mode) {
			return fmt.Errorf("source file %s has invalid size or mode", file.Path)
		}
		if err := validateSHA256("source file SHA-256", file.SHA256); err != nil {
			return err
		}
		if _, collision := directories[file.Path]; collision {
			return fmt.Errorf("source content path %s is both a file and directory", file.Path)
		}
		parent := filepath.ToSlash(filepath.Dir(file.Path))
		if parent != "." {
			if _, exists := directories[parent]; !exists {
				return fmt.Errorf(
					"source content file %s has missing parent %s",
					file.Path,
					parent,
				)
			}
		}
		previous = file.Path
		files[file.Path] = file
	}
	for _, required := range []string{
		"config/genesis.json",
		"config/node_key.json",
		"config/priv_validator_key.json",
		"data/priv_validator_state.json",
	} {
		if _, exists := files[required]; !exists {
			return fmt.Errorf("source content is missing required file %s", required)
		}
	}
	forHash := content
	forHash.SHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return err
	}
	if digest != content.SHA256 {
		return fmt.Errorf("source full content manifest self-hash mismatch")
	}
	return nil
}

func validateOptionalDigest(label string, present bool, digest string) error {
	if present {
		return validateSHA256(label+" SHA-256", digest)
	}
	if digest != "" {
		return fmt.Errorf("%s SHA-256 must be empty when absent", label)
	}
	return nil
}
