package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"
)

var allowedMediaTypes = map[string]struct{}{
	"application/json":         {},
	"application/octet-stream": {},
	"text/plain":               {},
}

func secureEvidenceRoot(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("evidence root is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve evidence root: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect evidence root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("evidence root must be a real directory, not a symlink")
	}
	evaluated, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve evidence root: %w", err)
	}
	return filepath.Clean(evaluated), nil
}

func validateEvidenceRefShape(reference EvidenceRef) error {
	if reference.Kind == "" || strings.TrimSpace(reference.Kind) != reference.Kind {
		return fmt.Errorf("evidence kind must be non-empty and trimmed")
	}
	if err := validateRelativePath(reference.Path); err != nil {
		return err
	}
	if _, allowed := allowedMediaTypes[reference.MediaType]; !allowed {
		return fmt.Errorf("evidence %s has unsupported media type %q", reference.Path, reference.MediaType)
	}
	if reference.SizeBytes <= 0 {
		return fmt.Errorf("evidence %s must have a positive size", reference.Path)
	}
	if err := validateSHA256("evidence SHA-256", reference.SHA256); err != nil {
		return fmt.Errorf("evidence %s: %w", reference.Path, err)
	}
	return nil
}

func validateRelativePath(path string) error {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if path == "" || path != cleaned || filepath.IsAbs(path) ||
		path == "." || path == ".." || strings.HasPrefix(path, "../") ||
		strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe evidence-relative path %q", path)
	}
	return nil
}

func resolveEvidencePath(root, relative string) (string, error) {
	if err := validateRelativePath(relative); err != nil {
		return "", err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return "", fmt.Errorf("evidence root changed or is no longer a real directory")
	}
	current := root
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf("inspect evidence %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("evidence %s contains a symlink at component %d", relative, index)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("evidence %s has a non-directory path component", relative)
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return "", fmt.Errorf("evidence %s is not a regular file", relative)
		}
	}
	return current, nil
}

func inspectEvidence(root string, reference EvidenceRef) error {
	_, err := readVerifiedEvidence(root, reference)
	return err
}

// readVerifiedEvidence returns the exact bytes that were hashed for JSON and
// text artifacts. Binary artifacts are streamed and return nil bytes because
// there is no content parser for them.
func readVerifiedEvidence(root string, reference EvidenceRef) ([]byte, error) {
	if err := validateEvidenceRefShape(reference); err != nil {
		return nil, err
	}
	data, size, actualDigest, err := captureEvidence(
		root,
		reference.Path,
		reference.MediaType,
	)
	if err != nil {
		return nil, err
	}
	if size != reference.SizeBytes {
		return nil, fmt.Errorf(
			"evidence %s size mismatch: report=%d actual=%d",
			reference.Path,
			reference.SizeBytes,
			size,
		)
	}
	if actualDigest != reference.SHA256 {
		return nil, fmt.Errorf(
			"evidence %s SHA-256 mismatch: report=%s actual=%s",
			reference.Path,
			reference.SHA256,
			actualDigest,
		)
	}
	return data, nil
}

func captureEvidence(
	root string,
	relative string,
	mediaType string,
) ([]byte, int64, string, error) {
	path, err := resolveEvidencePath(root, relative)
	if err != nil {
		return nil, 0, "", err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, 0, "", fmt.Errorf("inspect evidence %s: %w", relative, err)
	}
	file, err := openEvidenceNoFollow(path)
	if err != nil {
		return nil, 0, "", fmt.Errorf("open evidence %s: %w", relative, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, 0, "", fmt.Errorf("stat evidence %s: %w", relative, err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, 0, "", fmt.Errorf("evidence %s changed while it was opened", relative)
	}

	hasher := sha256.New()
	var data []byte
	var size int64
	switch mediaType {
	case "application/json", "text/plain":
		if opened.Size() > maxJSONEvidence {
			return nil, 0, "", fmt.Errorf(
				"evidence %s exceeds the %d-byte content validation limit",
				relative,
				maxJSONEvidence,
			)
		}
		data, err = io.ReadAll(io.TeeReader(
			io.LimitReader(file, maxJSONEvidence+1),
			hasher,
		))
		size = int64(len(data))
		if err == nil && size > maxJSONEvidence {
			err = fmt.Errorf("content exceeds the %d-byte validation limit", maxJSONEvidence)
		}
	default:
		size, err = io.Copy(hasher, file)
	}
	if err != nil {
		return nil, 0, "", fmt.Errorf("hash evidence %s: %w", relative, err)
	}
	resolvedAfter, resolveErr := resolveEvidencePath(root, relative)
	if resolveErr != nil || resolvedAfter != path {
		return nil, 0, "", fmt.Errorf(
			"evidence %s path changed while it was hashed",
			relative,
		)
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(opened, after) || after.Size() != size ||
		after.ModTime() != opened.ModTime() || after.Mode() != opened.Mode() {
		return nil, 0, "", fmt.Errorf("evidence %s changed while it was hashed", relative)
	}
	if err := validateEvidenceContent(data, mediaType, size); err != nil {
		return nil, 0, "", fmt.Errorf("evidence %s: %w", relative, err)
	}
	return data, size, hex.EncodeToString(hasher.Sum(nil)), nil
}

func openEvidenceNoFollow(path string) (*os.File, error) {
	fd, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func validateEvidenceContent(data []byte, mediaType string, size int64) error {
	switch mediaType {
	case "application/json":
		if size > maxJSONEvidence {
			return fmt.Errorf("JSON exceeds the %d-byte validation limit", maxJSONEvidence)
		}
		var document any
		if err := decodeJSONEvidence(data, &document); err != nil {
			return err
		}
	case "text/plain":
		if size > maxJSONEvidence {
			return fmt.Errorf("text exceeds the %d-byte validation limit", maxJSONEvidence)
		}
		if !utf8.Valid(data) || strings.IndexByte(string(data), 0) >= 0 {
			return fmt.Errorf("text/plain evidence must be valid UTF-8 without NUL bytes")
		}
	}
	return nil
}

func decodeJSONEvidence(data []byte, destination any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return fmt.Errorf("malformed JSON: %w", err)
	}
	decoder := jsonDecoder(data)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("malformed JSON: %w", err)
	}
	var trailer any
	if err := decoder.Decode(&trailer); err != io.EOF {
		if err == nil {
			return fmt.Errorf("malformed JSON: multiple root values")
		}
		return fmt.Errorf("malformed JSON trailer: %w", err)
	}
	return nil
}

func makeEvidenceRef(root, relative, kind, mediaType string) (EvidenceRef, error) {
	if _, allowed := allowedMediaTypes[mediaType]; !allowed {
		return EvidenceRef{}, fmt.Errorf("unsupported media type %q", mediaType)
	}
	_, size, digest, err := captureEvidence(root, relative, mediaType)
	if err != nil {
		return EvidenceRef{}, err
	}
	reference := EvidenceRef{
		Kind:      kind,
		Path:      relative,
		MediaType: mediaType,
		SizeBytes: size,
		SHA256:    digest,
	}
	if err := validateEvidenceRefShape(reference); err != nil {
		return EvidenceRef{}, err
	}
	return reference, nil
}

func allEvidence(report Report) ([]EvidenceRef, error) {
	references := make([]EvidenceRef, 0)
	appendReferences := func(values []EvidenceRef) {
		references = append(references, values...)
	}
	appendReferences(report.Upgrade.Evidence)
	appendReferences(report.Quarantine.Evidence)
	appendReferences(report.Recovery.Evidence)
	appendReferences(report.H1Latch.Evidence)
	appendReferences(report.FreshVolume.Evidence)
	appendReferences(report.Observer.Evidence)
	for _, fault := range report.Faults {
		appendReferences(fault.Evidence)
	}
	seen := make(map[string]struct{}, len(references))
	for _, reference := range references {
		if err := validateEvidenceRefShape(reference); err != nil {
			return nil, err
		}
		if _, duplicate := seen[reference.Path]; duplicate {
			return nil, fmt.Errorf("evidence path %q is referenced more than once", reference.Path)
		}
		seen[reference.Path] = struct{}{}
	}
	sort.Slice(references, func(i, j int) bool {
		return references[i].Path < references[j].Path
	})
	return references, nil
}

func evidenceManifestDigest(report Report) (string, error) {
	references, err := allEvidence(report)
	if err != nil {
		return "", err
	}
	return hashCanonical(EvidenceIndex{
		Schema:    evidenceIndexSchema,
		Artifacts: references,
	})
}

func verifyEvidenceFiles(report Report, root string) error {
	references, err := allEvidence(report)
	if err != nil {
		return err
	}
	for _, reference := range references {
		if err := inspectEvidence(root, reference); err != nil {
			return err
		}
	}
	return nil
}

func verifyEvidenceCrossLinks(report Report, root string) error {
	return verifyTypedEvidence(report, root)
}

func findEvidenceByKind(evidence []EvidenceRef, kind string) (EvidenceRef, error) {
	for _, reference := range evidence {
		if reference.Kind == kind {
			return reference, nil
		}
	}
	return EvidenceRef{}, fmt.Errorf("required evidence kind %q is missing", kind)
}

func jsonDecoder(data []byte) *json.Decoder {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	return decoder
}
