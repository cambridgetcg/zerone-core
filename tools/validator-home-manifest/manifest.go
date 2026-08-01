package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type CreateManifestOptions struct {
	DestinationHome            string
	DestinationVolumeID        string
	StoppedEvidencePath        string
	RestartInhibitEvidencePath string
	SnapshotEvidencePath       string
	VolumeEvidencePath         string
}

type VerifyManifestOptions struct {
	DestinationVolumeID        string
	RestartInhibitEvidencePath string
	SnapshotEvidencePath       string
	VolumeEvidencePath         string
}

func createManifest(home string, options CreateManifestOptions) (ValidatorHomeManifest, error) {
	secureHome, err := secureAbsoluteDirectory(home)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	secureDestination, err := secureAbsoluteDirectory(options.DestinationHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if secureDestination == secureHome {
		return ValidatorHomeManifest{}, fmt.Errorf("source snapshot and destination home must be different directories")
	}
	if options.DestinationVolumeID == "" ||
		strings.TrimSpace(options.DestinationVolumeID) != options.DestinationVolumeID {
		return ValidatorHomeManifest{}, fmt.Errorf("destination volume ID must be non-empty and trimmed")
	}
	sourceFilesystem, err := readFilesystemIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if !sourceFilesystem.ReadOnly {
		return ValidatorHomeManifest{}, fmt.Errorf("source home must be a locally read-only snapshot")
	}
	destinationFilesystem, err := readFilesystemIdentity(secureDestination)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if destinationFilesystem.ReadOnly {
		return ValidatorHomeManifest{}, fmt.Errorf("destination filesystem must be writable")
	}
	if destinationFilesystem.DeviceID == sourceFilesystem.DeviceID {
		return ValidatorHomeManifest{}, fmt.Errorf(
			"source and destination must be on different filesystem device IDs",
		)
	}
	if err := requireEmptyDirectory(secureDestination); err != nil {
		return ValidatorHomeManifest{}, err
	}
	evidence, err := loadStoppedEvidence(options.StoppedEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if evidence.SourceHome != secureHome {
		return ValidatorHomeManifest{}, fmt.Errorf(
			"stopped evidence source home %q does not match %q",
			evidence.SourceHome,
			secureHome,
		)
	}
	if err := checkProcessAbsent(evidence.ProcessID); err != nil {
		return ValidatorHomeManifest{}, fmt.Errorf("source signer absence re-check: %w", err)
	}
	restartEvidence, err := loadRestartInhibitEvidence(options.RestartInhibitEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if err := matchRestartEvidence(evidence, restartEvidence); err != nil {
		return ValidatorHomeManifest{}, err
	}
	snapshotEvidence, err := loadSnapshotEvidence(options.SnapshotEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if snapshotEvidence.SourceHome != secureHome {
		return ValidatorHomeManifest{}, fmt.Errorf("snapshot evidence source home does not match")
	}
	volumeEvidence, err := loadVolumeEvidence(options.VolumeEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if volumeEvidence.DestinationVolumeID != options.DestinationVolumeID {
		return ValidatorHomeManifest{}, fmt.Errorf("volume evidence destination ID does not match")
	}
	if err := validateExternalEvidenceOrder(
		evidence,
		snapshotEvidence,
		volumeEvidence,
	); err != nil {
		return ValidatorHomeManifest{}, err
	}
	content, err := scanHome(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	databases, err := databaseManifests(content)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	chain, err := inspectChain(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	consensus, err := inspectConsensusIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	signingState, err := inspectSigningState(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	finalContent, err := scanHome(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if !reflect.DeepEqual(finalContent, content) {
		return ValidatorHomeManifest{}, fmt.Errorf("validator home changed while its manifest was created")
	}
	finalConsensus, err := inspectConsensusIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalConsensus != consensus {
		return ValidatorHomeManifest{}, fmt.Errorf("validator identity changed while its manifest was created")
	}
	finalSigningState, err := inspectSigningState(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalSigningState != signingState {
		return ValidatorHomeManifest{}, fmt.Errorf("signing state changed while its manifest was created")
	}
	finalSourceFilesystem, err := readFilesystemIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalSourceFilesystem != sourceFilesystem {
		return ValidatorHomeManifest{}, fmt.Errorf("source filesystem identity changed during manifest creation")
	}
	finalDestinationFilesystem, err := readFilesystemIdentity(secureDestination)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalDestinationFilesystem != destinationFilesystem {
		return ValidatorHomeManifest{}, fmt.Errorf("destination filesystem identity changed during manifest creation")
	}
	if err := requireEmptyDirectory(secureDestination); err != nil {
		return ValidatorHomeManifest{}, fmt.Errorf("destination freshness re-check: %w", err)
	}
	if err := checkProcessAbsent(evidence.ProcessID); err != nil {
		return ValidatorHomeManifest{}, fmt.Errorf("source signer restarted during manifest creation: %w", err)
	}
	manifest := ValidatorHomeManifest{
		Schema:                manifestSchema,
		SourceHome:            secureHome,
		SourceFilesystem:      sourceFilesystem,
		DestinationFilesystem: destinationFilesystem,
		Destination: DestinationBinding{
			VolumeID:                options.DestinationVolumeID,
			VolumeEvidenceSHA256:    volumeEvidence.EvidenceSHA256,
			ObservedEmptyBeforeCopy: true,
		},
		Snapshot: SnapshotBinding{
			SnapshotID:     snapshotEvidence.SnapshotID,
			EvidenceSHA256: snapshotEvidence.EvidenceSHA256,
		},
		Chain:        chain,
		LastHeight:   evidence.LastHeight,
		AppHash:      evidence.AppHash,
		Consensus:    consensus,
		SigningState: signingState,
		StoppedEvidence: ManifestStoppedEvidence{
			CapturedAt:                   evidence.CapturedAt,
			Method:                       evidence.Method,
			Observer:                     evidence.Observer,
			ProcessID:                    evidence.ProcessID,
			ProcessStartTime:             evidence.ProcessStartTime,
			ProcessIdentitySHA256:        evidence.ProcessIdentitySHA256,
			ProcessAbsent:                evidence.ProcessAbsent,
			RestartInhibitEvidenceSHA256: evidence.RestartInhibitEvidenceSHA256,
			EvidenceSHA256:               evidence.EvidenceSHA256,
		},
		Databases: databases,
		Content:   content,
	}
	manifestForHash := manifest
	manifestForHash.ManifestSHA256 = ""
	manifest.ManifestSHA256, err = hashCanonical(manifestForHash)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if err := validateManifestShape(manifest); err != nil {
		return ValidatorHomeManifest{}, err
	}
	return manifest, nil
}

func verifyManifest(
	data []byte,
	home string,
	options VerifyManifestOptions,
	requireCanonical bool,
) (ValidatorHomeManifest, error) {
	var manifest ValidatorHomeManifest
	if err := decodeStrict(data, &manifest); err != nil {
		return ValidatorHomeManifest{}, err
	}
	if err := validateManifestShape(manifest); err != nil {
		return ValidatorHomeManifest{}, err
	}
	manifestForHash := manifest
	manifestForHash.ManifestSHA256 = ""
	digest, err := hashCanonical(manifestForHash)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if digest != manifest.ManifestSHA256 {
		return ValidatorHomeManifest{}, fmt.Errorf(
			"manifest self-hash mismatch: expected %s, computed %s",
			manifest.ManifestSHA256,
			digest,
		)
	}
	if requireCanonical {
		canonical, err := canonicalDocument(manifest)
		if err != nil {
			return ValidatorHomeManifest{}, err
		}
		if !bytes.Equal(data, canonical) {
			return ValidatorHomeManifest{}, fmt.Errorf("manifest is not exact canonical JSON")
		}
	}
	if options.DestinationVolumeID == "" {
		return ValidatorHomeManifest{}, fmt.Errorf("expected destination volume ID is required")
	}
	if manifest.Destination.VolumeID != options.DestinationVolumeID {
		return ValidatorHomeManifest{}, fmt.Errorf(
			"destination volume mismatch: manifest has %q, expected %q",
			manifest.Destination.VolumeID,
			options.DestinationVolumeID,
		)
	}
	secureHome, err := secureAbsoluteDirectory(home)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if secureHome != manifest.DestinationFilesystem.CanonicalPath {
		return ValidatorHomeManifest{}, fmt.Errorf("destination canonical path does not match the manifest")
	}
	actualFilesystem, err := readFilesystemIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if actualFilesystem != manifest.DestinationFilesystem {
		return ValidatorHomeManifest{}, fmt.Errorf("destination filesystem device/inode identity does not match")
	}
	if manifest.SourceFilesystem.DeviceID == actualFilesystem.DeviceID {
		return ValidatorHomeManifest{}, fmt.Errorf("destination reuses the source filesystem device")
	}
	if err := checkProcessAbsent(manifest.StoppedEvidence.ProcessID); err != nil {
		return ValidatorHomeManifest{}, fmt.Errorf("source signer absence re-check: %w", err)
	}
	restartEvidence, err := loadRestartInhibitEvidence(options.RestartInhibitEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if err := matchManifestRestartEvidence(manifest, restartEvidence); err != nil {
		return ValidatorHomeManifest{}, err
	}
	snapshotEvidence, err := loadSnapshotEvidence(options.SnapshotEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if snapshotEvidence.SourceHome != manifest.SourceHome ||
		snapshotEvidence.SnapshotID != manifest.Snapshot.SnapshotID ||
		snapshotEvidence.EvidenceSHA256 != manifest.Snapshot.EvidenceSHA256 {
		return ValidatorHomeManifest{}, fmt.Errorf("snapshot evidence does not match the manifest")
	}
	volumeEvidence, err := loadVolumeEvidence(options.VolumeEvidencePath)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if volumeEvidence.DestinationVolumeID != manifest.Destination.VolumeID ||
		volumeEvidence.EvidenceSHA256 != manifest.Destination.VolumeEvidenceSHA256 {
		return ValidatorHomeManifest{}, fmt.Errorf("volume-control evidence does not match the manifest")
	}
	if err := validateExternalEvidenceOrder(
		stoppedEvidenceFromManifest(manifest),
		snapshotEvidence,
		volumeEvidence,
	); err != nil {
		return ValidatorHomeManifest{}, err
	}
	actualContent, err := scanHome(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if !reflect.DeepEqual(actualContent, manifest.Content) {
		return ValidatorHomeManifest{}, fmt.Errorf("validator home content does not match the manifest")
	}
	actualDatabases, err := databaseManifests(actualContent)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if !reflect.DeepEqual(actualDatabases, manifest.Databases) {
		return ValidatorHomeManifest{}, fmt.Errorf("database file manifests do not match")
	}
	actualChain, err := inspectChain(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if actualChain != manifest.Chain {
		return ValidatorHomeManifest{}, fmt.Errorf("chain/genesis identity does not match")
	}
	actualConsensus, err := inspectConsensusIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if actualConsensus != manifest.Consensus {
		return ValidatorHomeManifest{}, fmt.Errorf("consensus or node identity does not match")
	}
	actualSigningState, err := inspectSigningState(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if actualSigningState != manifest.SigningState {
		return ValidatorHomeManifest{}, fmt.Errorf("validator signing state does not match")
	}
	finalContent, err := scanHome(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if !reflect.DeepEqual(finalContent, actualContent) {
		return ValidatorHomeManifest{}, fmt.Errorf("validator home changed while it was verified")
	}
	finalConsensus, err := inspectConsensusIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalConsensus != actualConsensus {
		return ValidatorHomeManifest{}, fmt.Errorf("validator identity changed while it was verified")
	}
	finalSigningState, err := inspectSigningState(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalSigningState != actualSigningState {
		return ValidatorHomeManifest{}, fmt.Errorf("signing state changed while it was verified")
	}
	finalFilesystem, err := readFilesystemIdentity(secureHome)
	if err != nil {
		return ValidatorHomeManifest{}, err
	}
	if finalFilesystem != actualFilesystem {
		return ValidatorHomeManifest{}, fmt.Errorf("destination filesystem identity changed during verification")
	}
	if err := checkProcessAbsent(manifest.StoppedEvidence.ProcessID); err != nil {
		return ValidatorHomeManifest{}, fmt.Errorf("source signer restarted during verification: %w", err)
	}
	return manifest, nil
}

func validateManifestShape(manifest ValidatorHomeManifest) error {
	if manifest.Schema != manifestSchema {
		return fmt.Errorf("manifest schema must be %q", manifestSchema)
	}
	if manifest.SourceHome == "" || !filepath.IsAbs(manifest.SourceHome) ||
		filepath.Clean(manifest.SourceHome) != manifest.SourceHome {
		return fmt.Errorf("manifest source_home must be absolute")
	}
	if manifest.SourceFilesystem.CanonicalPath != manifest.SourceHome ||
		!manifest.SourceFilesystem.ReadOnly ||
		manifest.SourceFilesystem.DeviceID == 0 ||
		manifest.SourceFilesystem.RootInode == 0 {
		return fmt.Errorf("manifest source filesystem must bind the read-only source snapshot")
	}
	if manifest.DestinationFilesystem.CanonicalPath == "" ||
		!filepath.IsAbs(manifest.DestinationFilesystem.CanonicalPath) ||
		filepath.Clean(manifest.DestinationFilesystem.CanonicalPath) !=
			manifest.DestinationFilesystem.CanonicalPath ||
		manifest.DestinationFilesystem.CanonicalPath == manifest.SourceHome ||
		manifest.DestinationFilesystem.ReadOnly ||
		manifest.DestinationFilesystem.DeviceID == 0 ||
		manifest.DestinationFilesystem.RootInode == 0 ||
		manifest.DestinationFilesystem.DeviceID == manifest.SourceFilesystem.DeviceID {
		return fmt.Errorf("manifest destination filesystem identity is invalid")
	}
	if manifest.Destination.VolumeID == "" ||
		strings.TrimSpace(manifest.Destination.VolumeID) != manifest.Destination.VolumeID ||
		!manifest.Destination.ObservedEmptyBeforeCopy {
		return fmt.Errorf("manifest destination binding is incomplete")
	}
	if err := validateSHA256(
		"destination volume-control evidence SHA-256",
		manifest.Destination.VolumeEvidenceSHA256,
	); err != nil {
		return err
	}
	if manifest.Snapshot.SnapshotID == "" ||
		strings.TrimSpace(manifest.Snapshot.SnapshotID) != manifest.Snapshot.SnapshotID {
		return fmt.Errorf("manifest snapshot binding is incomplete")
	}
	if err := validateSHA256("snapshot evidence SHA-256", manifest.Snapshot.EvidenceSHA256); err != nil {
		return err
	}
	if manifest.Chain.ChainID == "" || manifest.Chain.InitialHeight <= 0 {
		return fmt.Errorf("manifest chain identity is incomplete")
	}
	if err := validateSHA256("genesis SHA-256", manifest.Chain.GenesisSHA256); err != nil {
		return err
	}
	if manifest.LastHeight <= 0 || manifest.LastHeight == math.MaxInt64 {
		return fmt.Errorf("manifest last_height must permit a successor height")
	}
	if err := validateAppHash(manifest.AppHash); err != nil {
		return err
	}
	if len(manifest.Consensus.ConsensusAddress) != 40 ||
		strings.ToUpper(manifest.Consensus.ConsensusAddress) != manifest.Consensus.ConsensusAddress {
		return fmt.Errorf("manifest consensus address must be 40 uppercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(manifest.Consensus.ConsensusAddress); err != nil {
		return fmt.Errorf("manifest consensus address must be 40 uppercase hexadecimal characters")
	}
	if len(manifest.Consensus.NodeID) != 40 ||
		strings.ToLower(manifest.Consensus.NodeID) != manifest.Consensus.NodeID {
		return fmt.Errorf("manifest node ID must be 40 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(manifest.Consensus.NodeID); err != nil {
		return fmt.Errorf("manifest node ID must be 40 lowercase hexadecimal characters")
	}
	if manifest.Consensus.ConsensusKeyType != cometEd25519PublicKeyType ||
		manifest.Consensus.NodeKeyType != cometEd25519PrivateKeyType {
		return fmt.Errorf("manifest consensus and node key types must be canonical Ed25519 types")
	}
	for label, digest := range map[string]string{
		"consensus public key SHA-256": manifest.Consensus.ConsensusPubKeySHA256,
		"node public key SHA-256":      manifest.Consensus.NodePublicKeySHA256,
		"signing-state file SHA-256":   manifest.SigningState.StateFileSHA256,
		"stopped-evidence SHA-256":     manifest.StoppedEvidence.EvidenceSHA256,
		"content SHA-256":              manifest.Content.SHA256,
		"manifest SHA-256":             manifest.ManifestSHA256,
	} {
		if err := validateSHA256(label, digest); err != nil {
			return err
		}
	}
	if err := validateSigningStateManifest(manifest.SigningState); err != nil {
		return err
	}
	if manifest.SigningState.Height > manifest.LastHeight+1 {
		return fmt.Errorf("manifest signing state cannot be beyond last_height + 1")
	}
	if !manifest.StoppedEvidence.ProcessAbsent || manifest.StoppedEvidence.ProcessID <= 1 {
		return fmt.Errorf("manifest must bind evidence that the source process was absent")
	}
	if err := validateCanonicalTime("stopped evidence captured_at", manifest.StoppedEvidence.CapturedAt); err != nil {
		return err
	}
	if err := validateCanonicalTime(
		"stopped evidence process_start_time",
		manifest.StoppedEvidence.ProcessStartTime,
	); err != nil {
		return err
	}
	if err := validateSHA256(
		"stopped evidence process identity SHA-256",
		manifest.StoppedEvidence.ProcessIdentitySHA256,
	); err != nil {
		return err
	}
	if err := validateSHA256(
		"stopped evidence restart-inhibit evidence SHA-256",
		manifest.StoppedEvidence.RestartInhibitEvidenceSHA256,
	); err != nil {
		return err
	}
	if manifest.StoppedEvidence.Method == "" ||
		strings.TrimSpace(manifest.StoppedEvidence.Method) != manifest.StoppedEvidence.Method ||
		manifest.StoppedEvidence.Observer == "" ||
		strings.TrimSpace(manifest.StoppedEvidence.Observer) != manifest.StoppedEvidence.Observer {
		return fmt.Errorf("manifest stopped evidence method and observer must be non-empty and trimmed")
	}
	stopped := stoppedEvidenceFromManifest(manifest)
	if err := validateStoppedEvidence(stopped); err != nil {
		return err
	}
	stoppedForHash := stopped
	stoppedForHash.EvidenceSHA256 = ""
	stoppedDigest, err := hashCanonical(stoppedForHash)
	if err != nil {
		return err
	}
	if stoppedDigest != manifest.StoppedEvidence.EvidenceSHA256 {
		return fmt.Errorf("manifest stopped-evidence self-hash mismatch")
	}
	if len(manifest.Databases) != len(requiredDatabaseRoots) {
		return fmt.Errorf("manifest must contain exactly %d database groups", len(requiredDatabaseRoots))
	}
	contentFiles, err := validateContentManifest(manifest.Content)
	if err != nil {
		return err
	}
	for index, required := range requiredDatabaseRoots {
		database := manifest.Databases[index]
		if database.Name != required.name || database.Root != required.root || len(database.Files) == 0 {
			return fmt.Errorf("database group %d must be %s at %s and non-empty", index, required.name, required.root)
		}
		if err := validateSHA256("database group SHA-256", database.SHA256); err != nil {
			return err
		}
		expectedFiles := make([]FileRecord, 0)
		for _, file := range manifest.Content.Files {
			if strings.HasPrefix(file.Path, database.Root+"/") {
				expectedFiles = append(expectedFiles, file)
			}
		}
		if !reflect.DeepEqual(database.Files, expectedFiles) {
			return fmt.Errorf("database group %s does not contain its exact content file set", database.Name)
		}
		for _, file := range database.Files {
			if contentFile, exists := contentFiles[file.Path]; !exists ||
				!reflect.DeepEqual(contentFile, file) {
				return fmt.Errorf("database file %s does not match full content", file.Path)
			}
		}
		forHash := database
		forHash.SHA256 = ""
		digest, err := hashCanonical(forHash)
		if err != nil {
			return err
		}
		if digest != database.SHA256 {
			return fmt.Errorf("database group %s self-hash mismatch", database.Name)
		}
	}
	return nil
}

func validateContentManifest(content ContentManifest) (map[string]FileRecord, error) {
	if len(content.Files) == 0 || len(content.Directories) == 0 {
		return nil, fmt.Errorf("manifest full content must contain files and directories")
	}
	previous := ""
	directories := make(map[string]struct{}, len(content.Directories))
	for _, directory := range content.Directories {
		if err := validateRelativeManifestPath(directory.Path); err != nil {
			return nil, err
		}
		if previous != "" && directory.Path <= previous {
			return nil, fmt.Errorf("manifest directories are not strictly sorted")
		}
		if !validModeText(directory.Mode) {
			return nil, fmt.Errorf("manifest directory %s has a non-canonical mode", directory.Path)
		}
		previous = directory.Path
		directories[directory.Path] = struct{}{}
	}
	for _, required := range requiredDatabaseRoots {
		if _, exists := directories[required.root]; !exists {
			return nil, fmt.Errorf("manifest content is missing database directory %s", required.root)
		}
	}
	for directory := range directories {
		parent := filepath.ToSlash(filepath.Dir(directory))
		if parent != "." {
			if _, exists := directories[parent]; !exists {
				return nil, fmt.Errorf(
					"manifest directory %s has missing parent %s",
					directory,
					parent,
				)
			}
		}
	}
	previous = ""
	files := make(map[string]FileRecord, len(content.Files))
	for _, file := range content.Files {
		if err := validateRelativeManifestPath(file.Path); err != nil {
			return nil, err
		}
		if previous != "" && file.Path <= previous {
			return nil, fmt.Errorf("manifest files are not strictly sorted")
		}
		if file.Size < 0 || !validModeText(file.Mode) {
			return nil, fmt.Errorf("manifest file %s has an invalid size or mode", file.Path)
		}
		if err := validateSHA256("file SHA-256", file.SHA256); err != nil {
			return nil, err
		}
		if _, collision := directories[file.Path]; collision {
			return nil, fmt.Errorf("manifest path %s is both a file and directory", file.Path)
		}
		parent := filepath.ToSlash(filepath.Dir(file.Path))
		if parent != "." {
			if _, exists := directories[parent]; !exists {
				return nil, fmt.Errorf(
					"manifest file %s has missing parent %s",
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
			return nil, fmt.Errorf("manifest content is missing required file %s", required)
		}
	}
	forHash := content
	forHash.SHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return nil, err
	}
	if digest != content.SHA256 {
		return nil, fmt.Errorf("manifest full-content self-hash mismatch")
	}
	return files, nil
}

func validModeText(value string) bool {
	if len(value) != 4 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '7' {
			return false
		}
	}
	return true
}

func validateRelativeManifestPath(path string) error {
	if path == "" || path != filepath.ToSlash(filepath.Clean(path)) ||
		filepath.IsAbs(path) || path == "." || path == ".." ||
		strings.HasPrefix(path, "../") || strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe manifest-relative path %q", path)
	}
	return nil
}

func validateSigningStateManifest(state SigningState) error {
	if state.Height < 0 || state.Round < 0 || state.Step < 0 || state.Step > 3 {
		return fmt.Errorf("manifest signing-state height/round/step are outside Comet bounds")
	}
	if state.SignaturePresent != state.SignBytesPresent {
		return fmt.Errorf("manifest signing-state signature and sign bytes must be present together")
	}
	if state.Height == 0 {
		if state.Round != 0 || state.Step != 0 ||
			state.SignaturePresent || state.SignBytesPresent {
			return fmt.Errorf("genesis signing state must be exactly height/round/step 0 with no signature")
		}
	} else if state.Step < 1 || !state.SignaturePresent || !state.SignBytesPresent {
		return fmt.Errorf("non-genesis signing state requires Comet step 1..3 and signature/sign bytes")
	}
	if state.SignaturePresent {
		if err := validateSHA256("signature SHA-256", state.SignatureSHA256); err != nil {
			return err
		}
	} else if state.SignatureSHA256 != "" {
		return fmt.Errorf("signature SHA-256 must be empty when no signature is present")
	}
	if state.SignBytesPresent {
		if err := validateSHA256("sign-bytes SHA-256", state.SignBytesSHA256); err != nil {
			return err
		}
	} else if state.SignBytesSHA256 != "" {
		return fmt.Errorf("sign-bytes SHA-256 must be empty when no sign bytes are present")
	}
	return nil
}

func matchRestartEvidence(
	stopped StoppedEvidence,
	restart RestartInhibitEvidence,
) error {
	if restart.SourceHome != stopped.SourceHome ||
		restart.ProcessStartTime != stopped.ProcessStartTime ||
		restart.ProcessIdentitySHA256 != stopped.ProcessIdentitySHA256 ||
		restart.EvidenceSHA256 != stopped.RestartInhibitEvidenceSHA256 {
		return fmt.Errorf("restart-inhibit evidence does not match stopped-process evidence")
	}
	restartCaptured, _ := time.Parse(time.RFC3339Nano, restart.CapturedAt)
	stoppedCaptured, _ := time.Parse(time.RFC3339Nano, stopped.CapturedAt)
	if restartCaptured.After(stoppedCaptured) {
		return fmt.Errorf("restart-inhibit evidence was captured after stopped-process evidence")
	}
	return nil
}

func matchManifestRestartEvidence(
	manifest ValidatorHomeManifest,
	restart RestartInhibitEvidence,
) error {
	if restart.SourceHome != manifest.SourceHome ||
		restart.ProcessStartTime != manifest.StoppedEvidence.ProcessStartTime ||
		restart.ProcessIdentitySHA256 != manifest.StoppedEvidence.ProcessIdentitySHA256 ||
		restart.EvidenceSHA256 != manifest.StoppedEvidence.RestartInhibitEvidenceSHA256 {
		return fmt.Errorf("restart-inhibit evidence does not match the manifest")
	}
	restartCaptured, _ := time.Parse(time.RFC3339Nano, restart.CapturedAt)
	stoppedCaptured, _ := time.Parse(
		time.RFC3339Nano,
		manifest.StoppedEvidence.CapturedAt,
	)
	if restartCaptured.After(stoppedCaptured) {
		return fmt.Errorf("restart-inhibit evidence was captured after stopped-process evidence")
	}
	return nil
}

func stoppedEvidenceFromManifest(manifest ValidatorHomeManifest) StoppedEvidence {
	return StoppedEvidence{
		Schema:                       stoppedEvidenceSchema,
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
}

func validateExternalEvidenceOrder(
	stopped StoppedEvidence,
	snapshot SnapshotEvidence,
	volume VolumeEvidence,
) error {
	stoppedCaptured, _ := time.Parse(time.RFC3339Nano, stopped.CapturedAt)
	snapshotCaptured, _ := time.Parse(time.RFC3339Nano, snapshot.CapturedAt)
	volumeCaptured, _ := time.Parse(time.RFC3339Nano, volume.CapturedAt)
	if snapshotCaptured.Before(stoppedCaptured) {
		return fmt.Errorf("snapshot evidence predates stopped-process evidence")
	}
	if volumeCaptured.Before(stoppedCaptured) {
		return fmt.Errorf("volume-control freshness evidence predates stopped-process evidence")
	}
	return nil
}

func loadManifestFile(path string) ([]byte, error) {
	if path == "" || path == "-" {
		return nil, fmt.Errorf("manifest path must be a regular file, not stdin")
	}
	data, _, err := readRegularFile(path, maxControlFileBytes)
	return data, err
}

func manifestSummary(manifest ValidatorHomeManifest) string {
	return fmt.Sprintf(
		"VERIFIED_LOCAL_MANIFEST external_control_assertions=unverified schema=%s chain_id=%s height=%d app_hash=%s consensus_address=%s node_id=%s volume_id=%s files=%d manifest_sha256=%s",
		manifest.Schema,
		manifest.Chain.ChainID,
		manifest.LastHeight,
		manifest.AppHash,
		manifest.Consensus.ConsensusAddress,
		manifest.Consensus.NodeID,
		manifest.Destination.VolumeID,
		len(manifest.Content.Files),
		manifest.ManifestSHA256,
	)
}
