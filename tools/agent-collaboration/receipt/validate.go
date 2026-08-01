package receipt

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/oasisprotocol/curve25519-voi/curve"
)

var boundedLocalID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// ValidateManifest validates the immutable local collaboration boundary.
func ValidateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema {
		return fmt.Errorf("schema must be %q", ManifestSchema)
	}
	if manifest.Mode != ModeInternalLocal {
		return fmt.Errorf("mode must be %q", ModeInternalLocal)
	}
	if err := validateTimestamp("created_at", manifest.CreatedAt); err != nil {
		return err
	}
	if _, err := parsePrefixedHex("nonce", manifest.Nonce, "hex:", 32); err != nil {
		return err
	}
	if len(manifest.Participants) < 2 || len(manifest.Participants) > 16 {
		return errors.New("participants must contain between 2 and 16 entries")
	}
	if !reflect.DeepEqual(manifest.Effects, ZeroEffects()) {
		return errors.New("effects must be the exact v0 zero-effects boundary")
	}
	actorIDs := make(map[string]struct{}, len(manifest.Participants))
	keyIDs := make(map[string]struct{}, len(manifest.Participants))
	publicKeys := make(map[string]struct{}, len(manifest.Participants))
	previousActor := ""
	for index, participant := range manifest.Participants {
		if err := validateParticipant(participant); err != nil {
			return fmt.Errorf("participants[%d]: %w", index, err)
		}
		if index > 0 && participant.ActorID <= previousActor {
			return errors.New("participants must be strictly sorted by actor_id")
		}
		previousActor = participant.ActorID
		if _, exists := actorIDs[participant.ActorID]; exists {
			return fmt.Errorf("duplicate participant actor_id %q", participant.ActorID)
		}
		if _, exists := keyIDs[participant.KeyID]; exists {
			return fmt.Errorf("duplicate participant key_id %q", participant.KeyID)
		}
		if _, exists := publicKeys[participant.PublicKey]; exists {
			return errors.New("duplicate participant public_key")
		}
		actorIDs[participant.ActorID] = struct{}{}
		keyIDs[participant.KeyID] = struct{}{}
		publicKeys[participant.PublicKey] = struct{}{}
	}
	want, err := manifestID(manifest)
	if err != nil {
		return err
	}
	if manifest.CollaborationID != want {
		return fmt.Errorf("collaboration_id mismatch: expected %s", want)
	}
	return nil
}

func validateParticipant(participant Participant) error {
	if participant.Algorithm != AlgorithmEd25519 {
		return fmt.Errorf("algorithm must be %q", AlgorithmEd25519)
	}
	if err := validateText("label", participant.Label, 1, 64); err != nil {
		return err
	}
	publicKey, err := parsePrefixedHex("public_key", participant.PublicKey, "ed25519:", ed25519.PublicKeySize)
	if err != nil {
		return err
	}
	if err := validateEd25519PublicKey(publicKey); err != nil {
		return fmt.Errorf("public_key: %w", err)
	}
	if participant.ActorID != computeActorID(publicKey) {
		return errors.New("actor_id does not match the public key")
	}
	if participant.KeyID != computeKeyID(publicKey) {
		return errors.New("key_id does not match the public key")
	}
	return nil
}

func validateEd25519PublicKey(publicKey []byte) error {
	compressed, err := curve.NewCompressedEdwardsYFromBytes(publicKey)
	if err != nil {
		return errors.New("invalid compressed Edwards25519 encoding")
	}
	if !compressed.IsCanonicalVartime() {
		return errors.New("non-canonical Edwards25519 encoding")
	}
	var point curve.EdwardsPoint
	if _, err := point.SetCompressedY(compressed); err != nil {
		return errors.New("public key is not on Edwards25519")
	}
	if point.IsSmallOrder() {
		return errors.New("small-order Edwards25519 public key is forbidden")
	}
	if !point.IsTorsionFree() {
		return errors.New("Edwards25519 public key must be in the prime-order subgroup")
	}
	return nil
}

// ValidatePrivateKeyFile validates both halves of a local signing key.
func ValidatePrivateKeyFile(key PrivateKeyFile) error {
	if key.Schema != PrivateKeySchema {
		return fmt.Errorf("schema must be %q", PrivateKeySchema)
	}
	participant := Participant{ActorID: key.ActorID, Label: key.Label, KeyID: key.KeyID, Algorithm: key.Algorithm, PublicKey: key.PublicKey}
	if err := validateParticipant(participant); err != nil {
		return err
	}
	seed, err := parsePrefixedHex("private_key", key.PrivateKey, "ed25519-seed:", ed25519.SeedSize)
	if err != nil {
		return err
	}
	derived := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	publicKey, _ := parsePrefixedHex("public_key", key.PublicKey, "ed25519:", ed25519.PublicKeySize)
	if !reflect.DeepEqual([]byte(derived), publicKey) {
		return errors.New("private_key does not match public_key")
	}
	return nil
}

// ValidatePublicKeyFile validates a roster-safe public key document.
func ValidatePublicKeyFile(key PublicKeyFile) error {
	if key.Schema != PublicKeySchema {
		return fmt.Errorf("schema must be %q", PublicKeySchema)
	}
	return validateParticipant(key.Participant)
}

func validateEffects(effects Effects) error {
	if !reflect.DeepEqual(effects, ZeroEffects()) {
		return errors.New("effects must be the exact v0 zero-effects boundary")
	}
	return nil
}

func validateTimestamp(path, value string) error {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil || parsed.Format(time.RFC3339) != value || !strings.HasSuffix(value, "Z") {
		return fmt.Errorf("%s must be canonical UTC RFC3339 seconds", path)
	}
	return nil
}

func validateSequence(value string) (uint64, error) {
	if value == "" || value == "0" || (len(value) > 1 && value[0] == '0') {
		return 0, errors.New("sequence must be a canonical positive uint64 decimal string")
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, errors.New("sequence must be a canonical positive uint64 decimal string")
	}
	return parsed, nil
}

func validateText(path, value string, minimum, maximum int) error {
	if !utf8.ValidString(value) || len(value) < minimum || len(value) > maximum {
		return fmt.Errorf("%s must be valid UTF-8 between %d and %d bytes", path, minimum, maximum)
	}
	for _, character := range value {
		if character == 0 || unicode.IsControl(character) {
			return fmt.Errorf("%s must not contain control characters", path)
		}
		if isBidiFormattingControl(character) {
			return fmt.Errorf("%s must not contain bidirectional formatting controls", path)
		}
	}
	return nil
}

func isBidiFormattingControl(character rune) bool {
	return character == '\u061c' ||
		character == '\u200e' || character == '\u200f' ||
		(character >= '\u202a' && character <= '\u202e') ||
		(character >= '\u2066' && character <= '\u2069')
}

func validateLocalID(path, value string) error {
	if !boundedLocalID.MatchString(value) {
		return fmt.Errorf("%s must match %s", path, boundedLocalID.String())
	}
	return nil
}

func validateDigest(path, value string) error {
	if _, err := parseDigest(value); err != nil {
		return fmt.Errorf("%s %w", path, err)
	}
	return nil
}

func parsePrefixedHex(path, value, prefix string, byteLength int) ([]byte, error) {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+byteLength*2 {
		return nil, fmt.Errorf("%s must be %s followed by %d lowercase hex characters", path, prefix, byteLength*2)
	}
	encoded := value[len(prefix):]
	decoded, err := hex.DecodeString(encoded)
	if err != nil || hex.EncodeToString(decoded) != encoded {
		return nil, fmt.Errorf("%s must use canonical lowercase hex", path)
	}
	return decoded, nil
}

func validateSortedUnique(path string, values []string, maximum int, validate func(string, string) error) error {
	if values == nil {
		return fmt.Errorf("%s must be an array, not an omitted or null set", path)
	}
	if len(values) > maximum {
		return fmt.Errorf("%s exceeds %d entries", path, maximum)
	}
	if !sort.StringsAreSorted(values) {
		return fmt.Errorf("%s must be sorted", path)
	}
	for index, value := range values {
		if index > 0 && value == values[index-1] {
			return fmt.Errorf("%s must not contain duplicates", path)
		}
		if validate != nil {
			if err := validate(fmt.Sprintf("%s[%d]", path, index), value); err != nil {
				return err
			}
		}
	}
	return nil
}
