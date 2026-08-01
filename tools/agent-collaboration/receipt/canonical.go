package receipt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const (
	manifestDomain = "zerone.agent-collaboration.manifest/v0"
	actorDomain    = "zerone.agent-collaboration.actor/v0"
	keyDomain      = "zerone.agent-collaboration.key/v0"
	consentDomain  = "zerone.agent-collaboration.consent/v0"
	eventDomain    = "zerone.agent-collaboration.event/v0"
	receiptDomain  = "zerone.agent-collaboration.receipt/v0"
)

func canonicalJSON(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}
	return encoded, nil
}

func domainDigest(domain string, body []byte) []byte {
	hash := sha256.New()
	hash.Write([]byte(domain))
	hash.Write([]byte{0})
	hash.Write(body)
	return hash.Sum(nil)
}

func digestText(raw []byte) string {
	return "sha256:" + hex.EncodeToString(raw)
}

func parseDigest(value string) ([]byte, error) {
	if len(value) != len("sha256:")+sha256.Size*2 || value[:len("sha256:")] != "sha256:" {
		return nil, fmt.Errorf("must be sha256:<64 lowercase hex>")
	}
	decoded, err := hex.DecodeString(value[len("sha256:"):])
	if err != nil || hex.EncodeToString(decoded) != value[len("sha256:"):] {
		return nil, fmt.Errorf("must be sha256:<64 lowercase hex>")
	}
	return decoded, nil
}

func computeActorID(publicKey []byte) string {
	return digestText(domainDigest(actorDomain, publicKey))
}

func computeKeyID(publicKey []byte) string {
	return digestText(domainDigest(keyDomain, publicKey))
}

// ConsentTermsDigest returns the exact digest a decision must echo.
func ConsentTermsDigest(terms ConsentTerms) (string, error) {
	if err := validateConsentTerms("consent_terms", terms); err != nil {
		return "", err
	}
	canonical, err := canonicalJSON(terms)
	if err != nil {
		return "", err
	}
	return digestText(domainDigest(consentDomain, canonical)), nil
}

func canonicalManifestCore(manifest Manifest) ([]byte, error) {
	core := manifest
	core.CollaborationID = ""
	return canonicalJSON(core)
}

func manifestID(manifest Manifest) (string, error) {
	canonical, err := canonicalManifestCore(manifest)
	if err != nil {
		return "", err
	}
	return digestText(domainDigest(manifestDomain, canonical)), nil
}

func canonicalEvent(event Event) ([]byte, error) {
	_, payload, err := DecodePayload(event.Kind, event.Payload)
	if err != nil {
		return nil, err
	}
	normalized := event
	normalized.Payload = payload
	return canonicalJSON(normalized)
}

func eventDigest(event Event) ([]byte, error) {
	canonical, err := canonicalEvent(event)
	if err != nil {
		return nil, err
	}
	return domainDigest(eventDomain, canonical), nil
}

func receiptDigest(receipt SignedReceipt) ([]byte, error) {
	normalized := receipt
	normalized.ReceiptSHA256 = ""
	_, payload, err := DecodePayload(normalized.Event.Kind, normalized.Event.Payload)
	if err != nil {
		return nil, err
	}
	normalized.Event.Payload = payload
	canonical, err := canonicalJSON(normalized)
	if err != nil {
		return nil, err
	}
	return domainDigest(receiptDomain, canonical), nil
}
