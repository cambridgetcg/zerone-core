package protocol

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"path"
	"sort"

	"filippo.io/edwards25519"
)

//go:embed schemas/*.schema.json
var schemaFS embed.FS

var schemaFiles = map[Kind]string{
	KindKingdomReleaseRoot:         "kingdom-release-root.schema.json",
	KindAgentToolSettlementRoot:    "agenttool-settlement-root.schema.json",
	KindAgentToolCapability:        "agenttool-capability.schema.json",
	KindAgentToolPublicRecognition: "agenttool-public-recognition.schema.json",
	KindAgentToolOffer:             "agenttool-offer.schema.json",
	KindWakePublicCheckpoint:       "wake-public-checkpoint.schema.json",
	KindIssuerKeyContinuity:        "issuer-key-continuity.schema.json",
	KindArtifactLineage:            "artifact-lineage.schema.json",
	KindCollaborationCheckpoint:    "collaboration-checkpoint.schema.json",
	KindDisputeTerminal:            "dispute-terminal.schema.json",
}

func domainDigest(label string, data []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte(Protocol))
	h.Write([]byte{0})
	h.Write([]byte(label))
	h.Write([]byte{0})
	h.Write(data)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func digestString(sum [32]byte) string {
	return "sha256:" + hex.EncodeToString(sum[:])
}

func parseDigest(value string) ([32]byte, error) {
	var out [32]byte
	if len(value) != len("sha256:")+64 || value[:len("sha256:")] != "sha256:" {
		return out, fmt.Errorf("expected sha256:<64 lowercase hex>, got %q", value)
	}
	decoded, err := hex.DecodeString(value[len("sha256:"):])
	if err != nil || hex.EncodeToString(decoded) != value[len("sha256:"):] {
		return out, fmt.Errorf("invalid lowercase SHA-256 digest %q", value)
	}
	copy(out[:], decoded)
	return out, nil
}

func PayloadRoot(kind Kind, action Action, payload []byte) (string, error) {
	canonical, err := CanonicalJSON(payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize payload: %w", err)
	}
	sum := domainDigest("payload/"+string(kind)+"/"+string(action), canonical)
	return digestString(sum), nil
}

func Commitment(envelope Envelope) (string, error) {
	encoded, err := jsonBytes(envelope)
	if err != nil {
		return "", fmt.Errorf("canonicalize envelope: %w", err)
	}
	sum := domainDigest("envelope", encoded)
	return digestString(sum), nil
}

func ExpectedSchemaHash(kind Kind) (string, error) {
	name, ok := schemaFiles[kind]
	if !ok {
		return "", fmt.Errorf("no schema for kind %q", kind)
	}
	return embeddedSchemaHash(name)
}

func RecordSchemaHash() (string, error) {
	return embeddedSchemaHash("record.schema.json")
}

func SettlementBatchSchemaHash() (string, error) {
	return embeddedSchemaHash("settlement-batch.schema.json")
}

func embeddedSchemaHash(name string) (string, error) {
	contents, err := schemaFS.ReadFile(path.Join("schemas", name))
	if err != nil {
		return "", fmt.Errorf("read embedded schema %q: %w", name, err)
	}
	canonical, err := CanonicalJSON(contents)
	if err != nil {
		return "", fmt.Errorf("canonicalize embedded schema %q: %w", name, err)
	}
	sum := domainDigest("schema", canonical)
	return digestString(sum), nil
}

func SchemaHashes() (map[Kind]string, error) {
	result := make(map[Kind]string, len(schemaFiles))
	for kind := range schemaFiles {
		hash, err := ExpectedSchemaHash(kind)
		if err != nil {
			return nil, err
		}
		result[kind] = hash
	}
	return result, nil
}

type SchemaHashEntry struct {
	Kind       Kind   `json:"kind"`
	SchemaHash string `json:"schema_hash"`
}

func SchemaSetDigest() (string, error) {
	recordHash, err := RecordSchemaHash()
	if err != nil {
		return "", err
	}
	batchHash, err := SettlementBatchSchemaHash()
	if err != nil {
		return "", err
	}
	hashes, err := SchemaHashes()
	if err != nil {
		return "", err
	}
	entries := make([]SchemaHashEntry, 0, len(hashes))
	for kind, hash := range hashes {
		entries = append(entries, SchemaHashEntry{Kind: kind, SchemaHash: hash})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Kind < entries[j].Kind })
	document := struct {
		RecordSchemaHash          string            `json:"record_schema_hash"`
		SettlementBatchSchemaHash string            `json:"settlement_batch_schema_hash"`
		PayloadSchemas            []SchemaHashEntry `json:"payload_schemas"`
	}{recordHash, batchHash, entries}
	canonical, err := jsonBytes(document)
	if err != nil {
		return "", err
	}
	sum := domainDigest("schema-set", canonical)
	return digestString(sum), nil
}

func KeyFingerprint(publicKey []byte) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", fmt.Errorf("Ed25519 public key must be %d bytes", ed25519.PublicKeySize)
	}
	sum := sha256.Sum256(publicKey)
	return "ed25519-sha256:" + hex.EncodeToString(sum[:]), nil
}

func CapabilityNullifier(envelope Envelope, payload CapabilityConsumePayload) (string, error) {
	grant, err := parseDigest(payload.GrantCommitment)
	if err != nil {
		return "", fmt.Errorf("grant_commitment: %w", err)
	}
	source, err := parseDigest(payload.SourceEventDigest)
	if err != nil {
		return "", fmt.Errorf("source_event_digest: %w", err)
	}
	asset, err := parseDigest(payload.AssetRef)
	if err != nil {
		return "", fmt.Errorf("asset_ref: %w", err)
	}
	h := sha256.New()
	h.Write([]byte(Protocol))
	h.Write([]byte{0})
	h.Write([]byte("capability-nullifier"))
	h.Write([]byte{0})
	h.Write([]byte(envelope.Audience))
	h.Write([]byte{0})
	h.Write([]byte(envelope.SubjectRef))
	h.Write([]byte{0})
	h.Write([]byte(payload.CapabilityRef))
	h.Write([]byte{0})
	h.Write(grant[:])
	h.Write([]byte{0})
	h.Write(asset[:])
	h.Write([]byte{0})
	h.Write(source[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return digestString(out), nil
}

func VerifySignature(record Record) error {
	commitment, err := parseDigest(record.Commitment)
	if err != nil {
		return fmt.Errorf("commitment: %w", err)
	}
	if record.Signature.Algorithm != "Ed25519" {
		return fmt.Errorf("signature.algorithm must be Ed25519")
	}
	publicKey, err := decodeExactLowerHex(record.Signature.PublicKey, ed25519.PublicKeySize)
	if err != nil {
		return fmt.Errorf("signature.public_key: %w", err)
	}
	signature, err := decodeExactLowerHex(record.Signature.Value, ed25519.SignatureSize)
	if err != nil {
		return fmt.Errorf("signature.value: %w", err)
	}
	fingerprint, _ := KeyFingerprint(publicKey)
	if fingerprint != record.Envelope.Issuer.KeyFingerprint {
		return fmt.Errorf("issuer key fingerprint mismatch: expected %s", fingerprint)
	}
	if err := strictEd25519Verify(publicKey, commitment[:], signature); err != nil {
		return err
	}
	return nil
}

var ed25519PrimeOrder = func() *big.Int {
	value, ok := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	if !ok {
		panic("invalid Ed25519 prime order")
	}
	return value
}()

func strictEd25519Verify(publicKey, message, signature []byte) error {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid Ed25519 key or signature length")
	}
	if err := validateStrictEd25519Point("public key", publicKey); err != nil {
		return err
	}
	if err := validateStrictEd25519Point("signature R", signature[:32]); err != nil {
		return err
	}
	if _, err := new(edwards25519.Scalar).SetCanonicalBytes(signature[32:]); err != nil {
		return fmt.Errorf("signature S is not a canonical Ed25519 scalar")
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), message, signature) {
		return fmt.Errorf("invalid Ed25519 signature")
	}
	return nil
}

func validateStrictEd25519Point(label string, encoded []byte) error {
	point, err := new(edwards25519.Point).SetBytes(encoded)
	if err != nil {
		return fmt.Errorf("%s is not an Ed25519 point", label)
	}
	if !bytes.Equal(point.Bytes(), encoded) {
		return fmt.Errorf("%s encoding is noncanonical", label)
	}
	identity := edwards25519.NewIdentityPoint()
	if point.Equal(identity) == 1 {
		return fmt.Errorf("%s is the identity/small-order point", label)
	}
	if new(edwards25519.Point).MultByCofactor(point).Equal(identity) == 1 {
		return fmt.Errorf("%s is a small-order point", label)
	}
	result := edwards25519.NewIdentityPoint()
	addend := new(edwards25519.Point).Set(point)
	order := new(big.Int).Set(ed25519PrimeOrder)
	for i := 0; i < order.BitLen(); i++ {
		if order.Bit(i) == 1 {
			result.Add(result, addend)
		}
		addend.Add(addend, addend)
	}
	if result.Equal(identity) != 1 {
		return fmt.Errorf("%s is not in the prime-order subgroup", label)
	}
	return nil
}

func decodeExactLowerHex(value string, size int) ([]byte, error) {
	if len(value) != size*2 {
		return nil, fmt.Errorf("expected %d lowercase hex characters", size*2)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("invalid lowercase hex")
	}
	return decoded, nil
}

func jsonBytes(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return CanonicalJSON(encoded)
}
