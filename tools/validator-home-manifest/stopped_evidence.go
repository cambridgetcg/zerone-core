package main

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var nowUTC = func() time.Time {
	return time.Now().UTC()
}

var checkProcessAbsent = processAbsent

func processAbsent(processID int) error {
	if processID <= 1 {
		return fmt.Errorf("process ID must be greater than 1")
	}
	err := syscall.Kill(processID, syscall.Signal(0))
	switch {
	case err == nil:
		return fmt.Errorf("process %d is still running", processID)
	case errors.Is(err, syscall.EPERM):
		return fmt.Errorf("process %d exists but cannot be signalled", processID)
	case errors.Is(err, syscall.ESRCH):
		return nil
	default:
		return fmt.Errorf("inspect process %d: %w", processID, err)
	}
}

func createStoppedEvidence(
	home string,
	processID int,
	processStartTime string,
	processIdentitySHA256 string,
	restartInhibitEvidencePath string,
	lastHeight int64,
	appHash string,
	method string,
	observer string,
) (StoppedEvidence, error) {
	secureHome, err := secureAbsoluteDirectory(home)
	if err != nil {
		return StoppedEvidence{}, err
	}
	if lastHeight <= 0 || lastHeight == math.MaxInt64 {
		return StoppedEvidence{}, fmt.Errorf("last height must permit a successor height")
	}
	appHash = strings.ToLower(appHash)
	if err := validateAppHash(appHash); err != nil {
		return StoppedEvidence{}, err
	}
	if method == "" || strings.TrimSpace(method) != method {
		return StoppedEvidence{}, fmt.Errorf("method must be non-empty and trimmed")
	}
	if observer == "" || strings.TrimSpace(observer) != observer {
		return StoppedEvidence{}, fmt.Errorf("observer must be non-empty and trimmed")
	}
	if err := checkProcessAbsent(processID); err != nil {
		return StoppedEvidence{}, err
	}
	if err := validateCanonicalTime("process start time", processStartTime); err != nil {
		return StoppedEvidence{}, err
	}
	if err := validateSHA256("process identity SHA-256", processIdentitySHA256); err != nil {
		return StoppedEvidence{}, err
	}
	restartEvidence, err := loadRestartInhibitEvidence(restartInhibitEvidencePath)
	if err != nil {
		return StoppedEvidence{}, err
	}
	if restartEvidence.SourceHome != secureHome ||
		restartEvidence.ProcessStartTime != processStartTime ||
		restartEvidence.ProcessIdentitySHA256 != processIdentitySHA256 {
		return StoppedEvidence{}, fmt.Errorf(
			"restart-inhibit evidence does not match the source home and process identity",
		)
	}
	evidence := StoppedEvidence{
		Schema:                       stoppedEvidenceSchema,
		CapturedAt:                   nowUTC().Format(time.RFC3339Nano),
		Method:                       method,
		Observer:                     observer,
		SourceHome:                   secureHome,
		ProcessID:                    processID,
		ProcessStartTime:             processStartTime,
		ProcessIdentitySHA256:        processIdentitySHA256,
		ProcessAbsent:                true,
		RestartInhibitEvidenceSHA256: restartEvidence.EvidenceSHA256,
		LastHeight:                   lastHeight,
		AppHash:                      appHash,
	}
	evidenceForHash := evidence
	evidenceForHash.EvidenceSHA256 = ""
	evidence.EvidenceSHA256, err = hashCanonical(evidenceForHash)
	if err != nil {
		return StoppedEvidence{}, err
	}
	if err := validateStoppedEvidence(evidence); err != nil {
		return StoppedEvidence{}, err
	}
	restartCaptured, _ := time.Parse(time.RFC3339Nano, restartEvidence.CapturedAt)
	stoppedCaptured, _ := time.Parse(time.RFC3339Nano, evidence.CapturedAt)
	if restartCaptured.After(stoppedCaptured) {
		return StoppedEvidence{}, fmt.Errorf(
			"restart-inhibit evidence must be captured no later than stop evidence",
		)
	}
	if err := checkProcessAbsent(processID); err != nil {
		return StoppedEvidence{}, fmt.Errorf("source signer restarted during stop capture: %w", err)
	}
	return evidence, nil
}

func loadStoppedEvidence(path string) (StoppedEvidence, error) {
	if path == "" || path == "-" {
		return StoppedEvidence{}, fmt.Errorf("stopped evidence path must be a regular file")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return StoppedEvidence{}, fmt.Errorf("resolve stopped evidence: %w", err)
	}
	data, _, err := readRegularFile(absolute, maxControlFileBytes)
	if err != nil {
		return StoppedEvidence{}, err
	}
	var evidence StoppedEvidence
	if err := decodeStrict(data, &evidence); err != nil {
		return StoppedEvidence{}, fmt.Errorf("stopped evidence: %w", err)
	}
	if err := validateStoppedEvidence(evidence); err != nil {
		return StoppedEvidence{}, err
	}
	evidenceForHash := evidence
	evidenceForHash.EvidenceSHA256 = ""
	digest, err := hashCanonical(evidenceForHash)
	if err != nil {
		return StoppedEvidence{}, err
	}
	if digest != evidence.EvidenceSHA256 {
		return StoppedEvidence{}, fmt.Errorf("stopped evidence self-hash mismatch")
	}
	canonical, err := canonicalDocument(evidence)
	if err != nil {
		return StoppedEvidence{}, err
	}
	if !bytes.Equal(data, canonical) {
		return StoppedEvidence{}, fmt.Errorf("stopped evidence is not exact canonical JSON")
	}
	return evidence, nil
}

func validateStoppedEvidence(evidence StoppedEvidence) error {
	if evidence.Schema != stoppedEvidenceSchema {
		return fmt.Errorf("stopped evidence schema must be %q", stoppedEvidenceSchema)
	}
	if err := validateCanonicalTime("stopped evidence captured_at", evidence.CapturedAt); err != nil {
		return err
	}
	if evidence.Method == "" || strings.TrimSpace(evidence.Method) != evidence.Method {
		return fmt.Errorf("stopped evidence method must be non-empty and trimmed")
	}
	if evidence.Observer == "" || strings.TrimSpace(evidence.Observer) != evidence.Observer {
		return fmt.Errorf("stopped evidence observer must be non-empty and trimmed")
	}
	if evidence.SourceHome == "" || !filepath.IsAbs(evidence.SourceHome) {
		return fmt.Errorf("stopped evidence source_home must be absolute")
	}
	if evidence.ProcessID <= 1 || !evidence.ProcessAbsent {
		return fmt.Errorf("stopped evidence must record an absent process ID greater than 1")
	}
	if err := validateCanonicalTime("stopped evidence process_start_time", evidence.ProcessStartTime); err != nil {
		return err
	}
	capturedAt, _ := time.Parse(time.RFC3339Nano, evidence.CapturedAt)
	processStart, _ := time.Parse(time.RFC3339Nano, evidence.ProcessStartTime)
	if !processStart.Before(capturedAt) {
		return fmt.Errorf("stopped evidence process start time must precede capture")
	}
	if err := validateSHA256("stopped evidence process identity SHA-256", evidence.ProcessIdentitySHA256); err != nil {
		return err
	}
	if err := validateSHA256(
		"stopped evidence restart-inhibit evidence SHA-256",
		evidence.RestartInhibitEvidenceSHA256,
	); err != nil {
		return err
	}
	if evidence.LastHeight <= 0 || evidence.LastHeight == math.MaxInt64 {
		return fmt.Errorf("stopped evidence last_height must permit a successor height")
	}
	if err := validateAppHash(evidence.AppHash); err != nil {
		return err
	}
	return validateSHA256("stopped evidence SHA-256", evidence.EvidenceSHA256)
}

func stoppedEvidenceSummary(evidence StoppedEvidence) string {
	return fmt.Sprintf(
		"CAPTURED_LOCAL_STOP_EVIDENCE external_restart_assertion=unverified schema=%s source_home=%s process_id=%d process_identity_sha256=%s height=%d app_hash=%s evidence_sha256=%s",
		evidence.Schema,
		evidence.SourceHome,
		evidence.ProcessID,
		evidence.ProcessIdentitySHA256,
		evidence.LastHeight,
		evidence.AppHash,
		evidence.EvidenceSHA256,
	)
}

func evidenceOutputOutsideHome(home, output string) error {
	return ensureOutsideHome(home, output)
}
