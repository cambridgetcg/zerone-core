package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func loadRestartInhibitEvidence(path string) (RestartInhibitEvidence, error) {
	data, err := loadCanonicalEvidenceFile(path)
	if err != nil {
		return RestartInhibitEvidence{}, err
	}
	var evidence RestartInhibitEvidence
	if err := decodeStrict(data, &evidence); err != nil {
		return RestartInhibitEvidence{}, fmt.Errorf("restart-inhibit evidence: %w", err)
	}
	if evidence.Schema != restartEvidenceSchema {
		return RestartInhibitEvidence{}, fmt.Errorf(
			"restart-inhibit evidence schema must be %q",
			restartEvidenceSchema,
		)
	}
	if err := validateCanonicalTime("restart-inhibit captured_at", evidence.CapturedAt); err != nil {
		return RestartInhibitEvidence{}, err
	}
	if err := validateCanonicalTime("process start time", evidence.ProcessStartTime); err != nil {
		return RestartInhibitEvidence{}, err
	}
	capturedAt, _ := time.Parse(time.RFC3339Nano, evidence.CapturedAt)
	processStart, _ := time.Parse(time.RFC3339Nano, evidence.ProcessStartTime)
	if !processStart.Before(capturedAt) {
		return RestartInhibitEvidence{}, fmt.Errorf(
			"restart-inhibit process start time must precede capture",
		)
	}
	if evidence.SourceHome == "" || !filepath.IsAbs(evidence.SourceHome) ||
		evidence.Supervisor == "" || strings.TrimSpace(evidence.Supervisor) != evidence.Supervisor ||
		evidence.Unit == "" || strings.TrimSpace(evidence.Unit) != evidence.Unit ||
		!evidence.RestartDisabled || !evidence.StartBlocked {
		return RestartInhibitEvidence{}, fmt.Errorf("restart-inhibit evidence is incomplete")
	}
	if err := validateSHA256("process identity SHA-256", evidence.ProcessIdentitySHA256); err != nil {
		return RestartInhibitEvidence{}, err
	}
	if err := validateSHA256("restart-inhibit evidence SHA-256", evidence.EvidenceSHA256); err != nil {
		return RestartInhibitEvidence{}, err
	}
	forHash := evidence
	forHash.EvidenceSHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return RestartInhibitEvidence{}, err
	}
	if digest != evidence.EvidenceSHA256 {
		return RestartInhibitEvidence{}, fmt.Errorf("restart-inhibit evidence self-hash mismatch")
	}
	if err := requireCanonicalDocument(data, evidence, "restart-inhibit evidence"); err != nil {
		return RestartInhibitEvidence{}, err
	}
	return evidence, nil
}

func loadSnapshotEvidence(path string) (SnapshotEvidence, error) {
	data, err := loadCanonicalEvidenceFile(path)
	if err != nil {
		return SnapshotEvidence{}, err
	}
	var evidence SnapshotEvidence
	if err := decodeStrict(data, &evidence); err != nil {
		return SnapshotEvidence{}, fmt.Errorf("snapshot evidence: %w", err)
	}
	if evidence.Schema != snapshotEvidenceSchema {
		return SnapshotEvidence{}, fmt.Errorf("snapshot evidence schema must be %q", snapshotEvidenceSchema)
	}
	if err := validateCanonicalTime("snapshot captured_at", evidence.CapturedAt); err != nil {
		return SnapshotEvidence{}, err
	}
	if evidence.SourceHome == "" || !filepath.IsAbs(evidence.SourceHome) ||
		evidence.SnapshotID == "" || strings.TrimSpace(evidence.SnapshotID) != evidence.SnapshotID ||
		evidence.Method == "" || strings.TrimSpace(evidence.Method) != evidence.Method ||
		!evidence.ReadOnly || !evidence.Isolated || !evidence.IncludesACLs ||
		!evidence.IncludesXAttrs {
		return SnapshotEvidence{}, fmt.Errorf(
			"snapshot evidence must assert a read-only isolated snapshot including ACLs and xattrs",
		)
	}
	if err := validateSHA256("snapshot evidence SHA-256", evidence.EvidenceSHA256); err != nil {
		return SnapshotEvidence{}, err
	}
	forHash := evidence
	forHash.EvidenceSHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return SnapshotEvidence{}, err
	}
	if digest != evidence.EvidenceSHA256 {
		return SnapshotEvidence{}, fmt.Errorf("snapshot evidence self-hash mismatch")
	}
	if err := requireCanonicalDocument(data, evidence, "snapshot evidence"); err != nil {
		return SnapshotEvidence{}, err
	}
	return evidence, nil
}

func loadVolumeEvidence(path string) (VolumeEvidence, error) {
	data, err := loadCanonicalEvidenceFile(path)
	if err != nil {
		return VolumeEvidence{}, err
	}
	var evidence VolumeEvidence
	if err := decodeStrict(data, &evidence); err != nil {
		return VolumeEvidence{}, fmt.Errorf("volume-control evidence: %w", err)
	}
	if evidence.Schema != volumeEvidenceSchema {
		return VolumeEvidence{}, fmt.Errorf("volume-control evidence schema must be %q", volumeEvidenceSchema)
	}
	if err := validateCanonicalTime("volume evidence captured_at", evidence.CapturedAt); err != nil {
		return VolumeEvidence{}, err
	}
	if evidence.DestinationVolumeID == "" ||
		strings.TrimSpace(evidence.DestinationVolumeID) != evidence.DestinationVolumeID ||
		evidence.Provider == "" || strings.TrimSpace(evidence.Provider) != evidence.Provider ||
		!evidence.ImmutableID || !evidence.EncryptionAtRest || !evidence.Fresh ||
		!evidence.EmptyBeforeRestore {
		return VolumeEvidence{}, fmt.Errorf(
			"volume-control evidence must assert immutable ID, encryption, freshness, and pre-restore emptiness",
		)
	}
	if err := validateSHA256("volume-control evidence SHA-256", evidence.EvidenceSHA256); err != nil {
		return VolumeEvidence{}, err
	}
	forHash := evidence
	forHash.EvidenceSHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return VolumeEvidence{}, err
	}
	if digest != evidence.EvidenceSHA256 {
		return VolumeEvidence{}, fmt.Errorf("volume-control evidence self-hash mismatch")
	}
	if err := requireCanonicalDocument(data, evidence, "volume-control evidence"); err != nil {
		return VolumeEvidence{}, err
	}
	return evidence, nil
}

func loadCanonicalEvidenceFile(path string) ([]byte, error) {
	if path == "" || path == "-" {
		return nil, fmt.Errorf("external evidence path must be a regular file")
	}
	data, _, err := readRegularFile(path, maxControlFileBytes)
	return data, err
}

func requireCanonicalDocument(data []byte, value any, label string) error {
	canonical, err := canonicalDocument(value)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, canonical) {
		return fmt.Errorf("%s is not exact canonical JSON", label)
	}
	return nil
}
