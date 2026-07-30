package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode"
)

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func canonicalDigest(value any) (string, error) {
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", errors.New("canonical JSON encoding failed")
	}
	return digestBytes(canonical), nil
}

func decodeExactJSON[T any](data []byte, documentName string) (T, error) {
	var zero T
	if len(data) == 0 {
		return zero, fmt.Errorf("%s is empty", documentName)
	}
	if err := rejectSecretBearingFields(data); err != nil {
		return zero, fmt.Errorf("%s: %w", documentName, err)
	}

	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("%s JSON does not match its exact schema", documentName)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return zero, fmt.Errorf("%s JSON must contain exactly one value", documentName)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return zero, fmt.Errorf("%s canonical encoding failed", documentName)
	}
	if !bytes.Equal(data, canonical) {
		return zero, fmt.Errorf("%s JSON is not exact canonical compact JSON", documentName)
	}
	return value, nil
}

func rejectSecretBearingFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return errors.New("invalid JSON")
	}
	if containsSecretBearingField(value) {
		return errors.New("forbidden secret-bearing field is present")
	}
	return nil
}

func containsSecretBearingField(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					return unicode.ToLower(r)
				}
				return -1
			}, key)
			switch normalized {
			case "privatekey", "privkey", "mnemonic", "seed", "seedphrase",
				"secret", "secretkey", "recoveryphrase":
				return true
			}
			if containsSecretBearingField(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsSecretBearingField(child) {
				return true
			}
		}
	}
	return false
}

func readPinnedRegularFile(path, expectedSHA256, documentName string) ([]byte, string, error) {
	return readPinnedRegularFileBounded(
		path,
		expectedSHA256,
		documentName,
		maxInputBytes,
	)
}

func readPinnedRegularFileBounded(
	path,
	expectedSHA256,
	documentName string,
	maximumBytes int64,
) ([]byte, string, error) {
	if path == "" {
		return nil, "", fmt.Errorf("%s path is required", documentName)
	}
	if err := validateSHA256(documentName+" external SHA-256", expectedSHA256, false); err != nil {
		return nil, "", fmt.Errorf("%s requires a separately obtained exact SHA-256", documentName)
	}
	lowerBase := strings.ToLower(filepathBase(path))
	if strings.Contains(lowerBase, "priv_validator_key") ||
		strings.Contains(lowerBase, "node_key") ||
		strings.Contains(lowerBase, "mnemonic") ||
		strings.Contains(lowerBase, "seed_phrase") {
		return nil, "", fmt.Errorf("%s path name is secret-bearing and is refused", documentName)
	}

	before, err := os.Lstat(path)
	if err != nil {
		return nil, "", fmt.Errorf("%s cannot be inspected", documentName)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s must be a non-symlink regular file", documentName)
	}

	fd, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NONBLOCK|syscall.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return nil, "", fmt.Errorf("%s cannot be opened safely", documentName)
	}
	file := os.NewFile(uintptr(fd), documentName)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, "", fmt.Errorf("%s cannot be opened safely", documentName)
	}
	defer file.Close()

	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, "", fmt.Errorf("%s changed while it was opened or is not regular", documentName)
	}
	reader := io.LimitReader(file, maximumBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("%s cannot be read", documentName)
	}
	if int64(len(data)) > maximumBytes {
		return nil, "", fmt.Errorf("%s exceeds the %d-byte limit", documentName, maximumBytes)
	}
	actualSHA256 := digestBytes(data)
	if actualSHA256 != expectedSHA256 {
		return nil, "", fmt.Errorf("%s does not match its separately pinned SHA-256", documentName)
	}
	return data, actualSHA256, nil
}

// filepathBase avoids accepting secret-bearing default filenames without
// putting path contents into errors. It is intentionally tiny and portable.
func filepathBase(path string) string {
	path = strings.TrimRight(path, `/\`)
	if index := strings.LastIndexAny(path, `/\`); index >= 0 {
		return path[index+1:]
	}
	return path
}

func validateSHA256(name, value string, allowEmpty bool) error {
	if value == "" && allowEmpty {
		return nil
	}
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", name)
	}
	return nil
}

func validatePublicKey(name, value string) error {
	if len(value) != ed25519.PublicKeySize*2 {
		return fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 public key", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 public key", name)
	}
	return nil
}

func validateSignature(name, value string) error {
	if len(value) != ed25519.SignatureSize*2 {
		return fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 signature", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 signature", name)
	}
	return nil
}

func validateLabel(name, value string, maximum int) error {
	if value == "" || len(value) > maximum || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must be non-empty, trimmed, and at most %d bytes", name, maximum)
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return fmt.Errorf("%s must contain only printable ASCII without spaces", name)
		}
	}
	return nil
}

func validateTimestamp(name, value string) error {
	if value == "" || !strings.HasSuffix(value, "Z") {
		return fmt.Errorf("%s must be an RFC3339 UTC timestamp", name)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Format(time.RFC3339) != value {
		return fmt.Errorf("%s must be an exact RFC3339 UTC timestamp", name)
	}
	return nil
}

func validateCheckpoint(checkpoint Checkpoint) error {
	if checkpoint.Height == 0 {
		return errors.New("checkpoint height must be greater than zero")
	}
	if _, err := parseCanonicalUTCTime("checkpoint block time", checkpoint.BlockTime); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"checkpoint block ID":      checkpoint.BlockIDSHA256,
		"checkpoint app hash":      checkpoint.AppHashSHA256,
		"checkpoint signed commit": checkpoint.SignedCommitSHA256,
		"checkpoint validator set": checkpoint.ValidatorSetSHA256,
	} {
		if err := validateSHA256(name, value, false); err != nil {
			return err
		}
	}
	return nil
}

func validateValidatorIdentity(name string, identity ValidatorIdentity) error {
	if err := validateZRNValoper(name+" SDK operator address", identity.SDKOperatorAddress); err != nil {
		return err
	}
	if err := validatePublicKey(name+" consensus public key", identity.ConsensusPublicKey); err != nil {
		return err
	}
	expectedConsensusAddress, err := consensusAddressFromPublicKeyHex(
		identity.ConsensusPublicKey,
	)
	if err != nil || identity.ConsensusAddress != expectedConsensusAddress {
		return fmt.Errorf("%s consensus address does not derive from its public key", name)
	}
	if err := validatePublicKey(name+" node public key", identity.NodePublicKey); err != nil {
		return err
	}
	if identity.ConsensusPublicKey == identity.NodePublicKey {
		return fmt.Errorf("%s consensus and node public keys must be distinct", name)
	}
	expectedNodeID, err := nodeIDFromPublicKeyHex(identity.NodePublicKey)
	if err != nil || identity.NodeID != expectedNodeID {
		return fmt.Errorf("%s node ID does not derive from its public key", name)
	}
	for fieldName, value := range map[string]string{
		name + " validator key digest": identity.ValidatorKeySHA256,
		name + " node key digest":      identity.NodeKeySHA256,
		name + " signing state digest": identity.SigningStateSHA256,
	} {
		if err := validateSHA256(fieldName, value, false); err != nil {
			return err
		}
	}
	if identity.ValidatorKeySHA256 == identity.NodeKeySHA256 ||
		identity.ValidatorKeySHA256 == identity.SigningStateSHA256 ||
		identity.NodeKeySHA256 == identity.SigningStateSHA256 {
		return fmt.Errorf("%s key-file digests must be pairwise distinct", name)
	}
	return nil
}

func validateEvidence(evidence []Evidence) error {
	if evidence == nil {
		return errors.New("evidence must be [] rather than null")
	}
	if len(evidence) > maxCollectionEntries {
		return errors.New("evidence contains too many entries")
	}
	if !sort.SliceIsSorted(evidence, func(i, j int) bool {
		if evidence[i].Type != evidence[j].Type {
			return evidence[i].Type < evidence[j].Type
		}
		if evidence[i].SHA256 != evidence[j].SHA256 {
			return evidence[i].SHA256 < evidence[j].SHA256
		}
		return evidence[i].URI < evidence[j].URI
	}) {
		return errors.New("evidence must be sorted by type, digest, then URI")
	}
	seen := make(map[string]bool, len(evidence))
	for _, item := range evidence {
		if err := validateLabel("evidence type", item.Type, 128); err != nil {
			return err
		}
		if err := validateSHA256("evidence digest", item.SHA256, false); err != nil {
			return err
		}
		if item.URI == "" || len(item.URI) > 2048 || strings.TrimSpace(item.URI) != item.URI {
			return errors.New("evidence URI must be non-empty, trimmed, and at most 2048 bytes")
		}
		key := item.Type + "\x00" + item.SHA256 + "\x00" + item.URI
		if seen[key] {
			return errors.New("evidence entries must be unique")
		}
		seen[key] = true
	}
	return nil
}

func validateSortedUniqueStrings(name string, values []string, validator func(string, string) error) error {
	if values == nil {
		return fmt.Errorf("%s must be [] rather than null", name)
	}
	if len(values) > maxCollectionEntries {
		return fmt.Errorf("%s contains too many entries", name)
	}
	if !sort.StringsAreSorted(values) {
		return fmt.Errorf("%s must be sorted", name)
	}
	for index, value := range values {
		if index > 0 && value == values[index-1] {
			return fmt.Errorf("%s must contain unique entries", name)
		}
		if err := validator(name, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSortedHashes(name string, values []string) error {
	return validateSortedUniqueStrings(name, values, func(fieldName, value string) error {
		return validateSHA256(fieldName, value, false)
	})
}

func validateSortedLabels(name string, values []string) error {
	return validateSortedUniqueStrings(name, values, func(fieldName, value string) error {
		return validateLabel(fieldName, value, 256)
	})
}

func containsString(values []string, target string) bool {
	position := sort.SearchStrings(values, target)
	return position < len(values) && values[position] == target
}

func parseCanonicalPositiveInteger(name, value string) (*big.Int, error) {
	if value == "" || value == "0" || (len(value) > 1 && value[0] == '0') {
		return nil, fmt.Errorf("%s must be a canonical positive decimal integer", name)
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return nil, fmt.Errorf("%s must be a canonical positive decimal integer", name)
		}
	}
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || parsed.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be a canonical positive decimal integer", name)
	}
	return parsed, nil
}

func approvalLess(left, right Approval) bool {
	if left.Role != right.Role {
		return left.Role < right.Role
	}
	if left.Identity != right.Identity {
		return left.Identity < right.Identity
	}
	if left.ControlDomain != right.ControlDomain {
		return left.ControlDomain < right.ControlDomain
	}
	return left.PublicKey < right.PublicKey
}

func validateApprovalShape(approvals []Approval) error {
	if approvals == nil {
		return errors.New("approvals must be [] rather than null")
	}
	if len(approvals) > maxCollectionEntries {
		return errors.New("approvals contains too many entries")
	}
	if !sort.SliceIsSorted(approvals, func(i, j int) bool {
		return approvalLess(approvals[i], approvals[j])
	}) {
		return errors.New("approvals must be sorted by role, identity, control domain, then public key")
	}
	seen := make(map[string]bool, len(approvals))
	for _, approval := range approvals {
		if err := validateLabel("approval role", approval.Role, 128); err != nil {
			return err
		}
		if err := validateLabel("approval identity", approval.Identity, 256); err != nil {
			return err
		}
		if err := validateLabel("approval control domain", approval.ControlDomain, 256); err != nil {
			return err
		}
		if err := validatePublicKey("approval public key", approval.PublicKey); err != nil {
			return err
		}
		if err := validateSHA256("approval statement digest", approval.StatementSHA256, false); err != nil {
			return err
		}
		if err := validateSignature("approval signature", approval.Signature); err != nil {
			return err
		}
		key := approval.Role + "\x00" + approval.Identity + "\x00" +
			approval.ControlDomain + "\x00" + approval.PublicKey
		if seen[key] {
			return errors.New("approval tuples must be unique")
		}
		seen[key] = true
	}
	return nil
}

func makeApprovalStatement(domain string, body any, approval Approval) (string, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", errors.New("approval body encoding failed")
	}
	bodyDigest := sha256.Sum256(bodyJSON)
	publicKey, err := hex.DecodeString(approval.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New("approval public key is invalid")
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write(bodyDigest[:])
	for _, field := range []string{
		approval.Role,
		approval.Identity,
		approval.ControlDomain,
	} {
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(field))
	}
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(publicKey)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func verifyApprovalSignature(approval Approval, expectedStatementSHA256 string) error {
	if approval.StatementSHA256 != expectedStatementSHA256 {
		return errors.New("approval statement digest mismatch")
	}
	publicKey, err := hex.DecodeString(approval.PublicKey)
	if err != nil {
		return errors.New("approval public key is invalid")
	}
	signature, err := hex.DecodeString(approval.Signature)
	if err != nil {
		return errors.New("approval signature is invalid")
	}
	statement, err := hex.DecodeString(expectedStatementSHA256)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), statement, signature) {
		return errors.New("approval signature verification failed")
	}
	return nil
}

func custodyApprovalBody(assessment CustodyAssessment) CustodyAssessment {
	assessment.Approvals = []Approval{}
	assessment.AssessmentSHA256 = ""
	return assessment
}

func controlledApprovalBody(plan ControlledTransition) ControlledTransition {
	plan.Approvals = []Approval{}
	plan.PlanSHA256 = ""
	return plan
}

func forkReleaseApprovalBody(release ForkRelease) ForkRelease {
	release.Approvals = []Approval{}
	release.ReleaseSHA256 = ""
	return release
}

func forkChoiceApprovalBody(choice ForkChoice) ForkChoice {
	choice.Approvals = []Approval{}
	choice.ChoiceSHA256 = ""
	return choice
}

func custodyApprovalStatement(assessment CustodyAssessment, approval Approval) (string, error) {
	return makeApprovalStatement(custodyApprovalDomain, custodyApprovalBody(assessment), approval)
}

func controlledApprovalStatement(plan ControlledTransition, approval Approval) (string, error) {
	return makeApprovalStatement(controlledApprovalDomain, controlledApprovalBody(plan), approval)
}

func forkReleaseApprovalStatement(release ForkRelease, approval Approval) (string, error) {
	return makeApprovalStatement(forkReleaseApprovalDomain, forkReleaseApprovalBody(release), approval)
}

func forkChoiceApprovalStatement(choice ForkChoice, approval Approval) (string, error) {
	return makeApprovalStatement(forkChoiceApprovalDomain, forkChoiceApprovalBody(choice), approval)
}

func genesisReproductionBody(release ForkRelease) ForkRelease {
	release.GenesisReproductions = []GenesisReproduction{}
	release.Approvals = []Approval{}
	release.ReleaseSHA256 = ""
	return release
}

func genesisReproductionStatement(
	release ForkRelease,
	reproduction GenesisReproduction,
) (string, error) {
	bodyJSON, err := json.Marshal(genesisReproductionBody(release))
	if err != nil {
		return "", errors.New("genesis reproduction body encoding failed")
	}
	bodyDigest := sha256.Sum256(bodyJSON)
	publicKey, err := hex.DecodeString(reproduction.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New("genesis reproduction public key is invalid")
	}
	genesisDigest, err := hex.DecodeString(reproduction.GenesisSHA256)
	if err != nil || len(genesisDigest) != sha256.Size {
		return "", errors.New("genesis reproduction digest is invalid")
	}
	reportDigest, err := hex.DecodeString(reproduction.CompilerReportFileSHA256)
	if err != nil || len(reportDigest) != sha256.Size {
		return "", errors.New("compiler report file digest is invalid")
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(genesisReproductionDomain))
	_, _ = hasher.Write(bodyDigest[:])
	for _, field := range []string{
		reproduction.Identity,
		reproduction.ControlDomain,
	} {
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(field))
	}
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(publicKey)
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(genesisDigest)
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(reportDigest)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func sealCustody(assessment CustodyAssessment) (CustodyAssessment, error) {
	assessment.AssessmentSHA256 = ""
	digest, err := canonicalDigest(assessment)
	assessment.AssessmentSHA256 = digest
	return assessment, err
}

func sealControlled(plan ControlledTransition) (ControlledTransition, error) {
	plan.PlanSHA256 = ""
	digest, err := canonicalDigest(plan)
	plan.PlanSHA256 = digest
	return plan, err
}

func sealForkRelease(release ForkRelease) (ForkRelease, error) {
	release.ReleaseSHA256 = ""
	digest, err := canonicalDigest(release)
	release.ReleaseSHA256 = digest
	return release, err
}

func sealForkChoice(choice ForkChoice) (ForkChoice, error) {
	choice.ChoiceSHA256 = ""
	digest, err := canonicalDigest(choice)
	choice.ChoiceSHA256 = digest
	return choice, err
}

func verifyCustodySeal(assessment CustodyAssessment) error {
	actual := assessment.AssessmentSHA256
	sealed, err := sealCustody(assessment)
	if err != nil || actual == "" || sealed.AssessmentSHA256 != actual {
		return errors.New("custody assessment self hash mismatch")
	}
	return nil
}

func verifyControlledSeal(plan ControlledTransition) error {
	actual := plan.PlanSHA256
	sealed, err := sealControlled(plan)
	if err != nil || actual == "" || sealed.PlanSHA256 != actual {
		return errors.New("controlled transition self hash mismatch")
	}
	return nil
}

func verifyForkReleaseSeal(release ForkRelease) error {
	actual := release.ReleaseSHA256
	sealed, err := sealForkRelease(release)
	if err != nil || actual == "" || sealed.ReleaseSHA256 != actual {
		return errors.New("fork release self hash mismatch")
	}
	return nil
}

func verifyForkChoiceSeal(choice ForkChoice) error {
	actual := choice.ChoiceSHA256
	sealed, err := sealForkChoice(choice)
	if err != nil || actual == "" || sealed.ChoiceSHA256 != actual {
		return errors.New("fork choice self hash mismatch")
	}
	return nil
}
