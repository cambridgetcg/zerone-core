package protocol

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// SignRecord is a pure offline constructor. It performs no network requests,
// clock reads, randomness, persistence, Zerone transactions, or economic
// actions. Callers must supply an Ed25519 private key and every semantic field.
func SignRecord(envelope Envelope, payloadJSON []byte, privateKey ed25519.PrivateKey) (Record, []byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return Record{}, nil, fmt.Errorf("Ed25519 private key must be %d bytes", ed25519.PrivateKeySize)
	}
	canonicalPayload, err := CanonicalJSON(payloadJSON)
	if err != nil {
		return Record{}, nil, fmt.Errorf("canonicalize payload: %w", err)
	}
	envelope.Protocol = Protocol
	envelope.SchemaHash, err = ExpectedSchemaHash(envelope.Kind)
	if err != nil {
		return Record{}, nil, err
	}
	envelope.PayloadRoot, err = PayloadRoot(envelope.Kind, envelope.Action, canonicalPayload)
	if err != nil {
		return Record{}, nil, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	fingerprint, err := KeyFingerprint(publicKey)
	if err != nil {
		return Record{}, nil, err
	}
	if envelope.Issuer.KeyFingerprint == "" {
		envelope.Issuer.KeyFingerprint = fingerprint
	}
	if envelope.Issuer.KeyFingerprint != fingerprint {
		return Record{}, nil, fmt.Errorf("private key does not match issuer.key_fingerprint")
	}
	commitment, err := Commitment(envelope)
	if err != nil {
		return Record{}, nil, err
	}
	commitmentBytes, _ := parseDigest(commitment)
	signature := ed25519.Sign(privateKey, commitmentBytes[:])
	record := Record{
		Envelope: envelope, Payload: json.RawMessage(canonicalPayload), Commitment: commitment,
		Signature: Signature{Algorithm: "Ed25519", PublicKey: hex.EncodeToString(publicKey), Value: hex.EncodeToString(signature)},
	}
	canonicalRecord, err := jsonBytes(record)
	if err != nil {
		return Record{}, nil, err
	}
	if _, err := Verify(canonicalRecord); err != nil {
		return Record{}, nil, fmt.Errorf("constructed record failed verification: %w", err)
	}
	return record, canonicalRecord, nil
}
