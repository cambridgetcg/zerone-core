package receipt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
)

// GenerateKey creates an isolated internal Ed25519 identity using the
// operating system CSPRNG. The label has no authority semantics.
func GenerateKey(label string) (PrivateKeyFile, PublicKeyFile, error) {
	return generateKeyWithRandom(label, rand.Reader)
}

func generateKeyWithRandom(label string, randomness io.Reader) (PrivateKeyFile, PublicKeyFile, error) {
	if err := validateText("label", label, 1, 64); err != nil {
		return PrivateKeyFile{}, PublicKeyFile{}, err
	}
	publicKey, privateKey, err := ed25519.GenerateKey(randomness)
	if err != nil {
		return PrivateKeyFile{}, PublicKeyFile{}, fmt.Errorf("generate Ed25519 key: %w", err)
	}
	participant := Participant{
		ActorID:   computeActorID(publicKey),
		Label:     label,
		KeyID:     computeKeyID(publicKey),
		Algorithm: AlgorithmEd25519,
		PublicKey: "ed25519:" + hex.EncodeToString(publicKey),
	}
	private := PrivateKeyFile{
		Schema:     PrivateKeySchema,
		ActorID:    participant.ActorID,
		Label:      label,
		KeyID:      participant.KeyID,
		Algorithm:  AlgorithmEd25519,
		PublicKey:  participant.PublicKey,
		PrivateKey: "ed25519-seed:" + hex.EncodeToString(privateKey.Seed()),
	}
	public := PublicKeyFile{Schema: PublicKeySchema, Participant: participant}
	return private, public, nil
}

// PublicFromPrivate returns the exact roster-safe projection of a validated
// private key file.
func PublicFromPrivate(key PrivateKeyFile) (PublicKeyFile, error) {
	if err := ValidatePrivateKeyFile(key); err != nil {
		return PublicKeyFile{}, err
	}
	return PublicKeyFile{
		Schema: PublicKeySchema,
		Participant: Participant{
			ActorID:   key.ActorID,
			Label:     key.Label,
			KeyID:     key.KeyID,
			Algorithm: key.Algorithm,
			PublicKey: key.PublicKey,
		},
	}, nil
}

// NewManifest freezes a roster into one content-addressed local collaboration
// using a fresh OS-CSPRNG nonce. Participant order is canonical actor-ID order.
func NewManifest(participants []Participant, createdAt string) (Manifest, error) {
	if err := validateManifestInputs(participants, createdAt); err != nil {
		return Manifest{}, err
	}
	nonce, err := randomNonceWithReader(rand.Reader)
	if err != nil {
		return Manifest{}, err
	}
	return newManifestWithNonce(participants, createdAt, nonce)
}

func newManifestWithNonce(participants []Participant, createdAt, nonce string) (Manifest, error) {
	if err := validateManifestInputs(participants, createdAt); err != nil {
		return Manifest{}, err
	}
	if _, err := parsePrefixedHex("nonce", nonce, "hex:", 32); err != nil {
		return Manifest{}, err
	}
	copyOfParticipants := append([]Participant(nil), participants...)
	sort.Slice(copyOfParticipants, func(left, right int) bool {
		return copyOfParticipants[left].ActorID < copyOfParticipants[right].ActorID
	})
	manifest := Manifest{
		Schema:          ManifestSchema,
		Mode:            ModeInternalLocal,
		CollaborationID: "",
		CreatedAt:       createdAt,
		Nonce:           nonce,
		Participants:    copyOfParticipants,
		Effects:         ZeroEffects(),
	}
	id, err := manifestID(manifest)
	if err != nil {
		return Manifest{}, err
	}
	manifest.CollaborationID = id
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifestInputs(participants []Participant, createdAt string) error {
	if len(participants) < 2 || len(participants) > 16 {
		return errors.New("participants must contain between 2 and 16 entries")
	}
	if err := validateTimestamp("created_at", createdAt); err != nil {
		return err
	}
	for index, participant := range participants {
		if err := validateParticipant(participant); err != nil {
			return fmt.Errorf("participants[%d]: %w", index, err)
		}
	}
	return nil
}

func randomNonceWithReader(randomness io.Reader) (string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(randomness, raw); err != nil {
		return "", fmt.Errorf("read random nonce: %w", err)
	}
	return "hex:" + hex.EncodeToString(raw), nil
}

func privateSigningKey(key PrivateKeyFile) (ed25519.PrivateKey, error) {
	if err := ValidatePrivateKeyFile(key); err != nil {
		return nil, err
	}
	seed, err := parsePrefixedHex("private_key", key.PrivateKey, "ed25519-seed:", ed25519.SeedSize)
	if err != nil {
		return nil, err
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func participantFor(manifest Manifest, actorID string) (Participant, bool) {
	index := sort.Search(len(manifest.Participants), func(index int) bool {
		return manifest.Participants[index].ActorID >= actorID
	})
	if index >= len(manifest.Participants) || manifest.Participants[index].ActorID != actorID {
		return Participant{}, false
	}
	return manifest.Participants[index], true
}

func publicSigningKey(participant Participant) (ed25519.PublicKey, error) {
	raw, err := parsePrefixedHex("public_key", participant.PublicKey, "ed25519:", ed25519.PublicKeySize)
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(raw), nil
}

func decodeSignature(value string) ([]byte, error) {
	return parsePrefixedHex("signature.value", value, "ed25519:", ed25519.SignatureSize)
}

func requireKeyMatchesParticipant(key PrivateKeyFile, participant Participant) error {
	if key.ActorID != participant.ActorID || key.KeyID != participant.KeyID || key.PublicKey != participant.PublicKey || key.Algorithm != participant.Algorithm {
		return errors.New("private key does not match the manifest participant")
	}
	return nil
}
