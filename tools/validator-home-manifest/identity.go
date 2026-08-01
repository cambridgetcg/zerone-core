package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type genesisDocument struct {
	ChainID       string          `json:"chain_id"`
	InitialHeight json.RawMessage `json:"initial_height"`
}

type encodedKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type validatorKeyDocument struct {
	Address string     `json:"address"`
	PubKey  encodedKey `json:"pub_key"`
	PrivKey encodedKey `json:"priv_key"`
}

type nodeKeyDocument struct {
	PrivKey encodedKey `json:"priv_key"`
}

type signingStateDocument struct {
	Height    string  `json:"height"`
	Round     int64   `json:"round"`
	Step      int64   `json:"step"`
	Signature *string `json:"signature"`
	SignBytes *string `json:"signbytes"`
}

const (
	cometEd25519PublicKeyType  = "tendermint/PubKeyEd25519"
	cometEd25519PrivateKeyType = "tendermint/PrivKeyEd25519"
)

func inspectChain(home string) (ChainIdentity, error) {
	path := filepath.Join(home, "config", "genesis.json")
	data, _, err := readRegularFile(path, maxControlFileBytes)
	if err != nil {
		return ChainIdentity{}, err
	}
	var document genesisDocument
	if err := decodeStrict(data, &document); err != nil {
		// Genesis has many fields, so strict unknown-field rejection cannot be
		// used against the projection struct.
		if err := rejectDuplicateKeys(data); err != nil {
			return ChainIdentity{}, fmt.Errorf("genesis: %w", err)
		}
		if err := json.Unmarshal(data, &document); err != nil {
			return ChainIdentity{}, fmt.Errorf("decode genesis: %w", err)
		}
	}
	if document.ChainID == "" || strings.TrimSpace(document.ChainID) != document.ChainID {
		return ChainIdentity{}, fmt.Errorf("genesis chain_id must be non-empty and trimmed")
	}
	initialHeight, err := parseInitialHeight(document.InitialHeight)
	if err != nil {
		return ChainIdentity{}, err
	}
	digest := sha256.Sum256(data)
	return ChainIdentity{
		ChainID:       document.ChainID,
		InitialHeight: initialHeight,
		GenesisSHA256: hex.EncodeToString(digest[:]),
	}, nil
}

func parseInitialHeight(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 1, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, fmt.Errorf("genesis initial_height must be a canonical positive decimal string")
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil || value <= 0 || strconv.FormatInt(value, 10) != text {
		return 0, fmt.Errorf("genesis initial_height must be a canonical positive decimal string")
	}
	return value, nil
}

func inspectConsensusIdentity(home string) (ConsensusIdentity, error) {
	validatorPath := filepath.Join(home, "config", "priv_validator_key.json")
	validatorData, validatorInfo, err := readRegularFile(validatorPath, maxControlFileBytes)
	if err != nil {
		return ConsensusIdentity{}, err
	}
	if validatorInfo.Mode().Perm()&0o077 != 0 {
		return ConsensusIdentity{}, fmt.Errorf("%s must not be group- or world-accessible", validatorPath)
	}
	if links, err := fileLinkCount(validatorInfo); err != nil || links != 1 {
		return ConsensusIdentity{}, fmt.Errorf("%s must have exactly one hard link", validatorPath)
	}
	var validator validatorKeyDocument
	if err := decodeStrict(validatorData, &validator); err != nil {
		return ConsensusIdentity{}, fmt.Errorf("validator key: %w", err)
	}
	validatorPublic, err := decodeCanonicalBase64("validator public key", validator.PubKey.Value)
	if err != nil {
		return ConsensusIdentity{}, err
	}
	if len(validatorPublic) != ed25519.PublicKeySize ||
		validator.PubKey.Type != cometEd25519PublicKeyType {
		return ConsensusIdentity{}, fmt.Errorf("validator public key must be a 32-byte Ed25519 key")
	}
	validatorAddressDigest := sha256.Sum256(validatorPublic)
	validatorAddress := strings.ToUpper(hex.EncodeToString(validatorAddressDigest[:20]))
	if validator.Address != validatorAddress {
		return ConsensusIdentity{}, fmt.Errorf(
			"validator address mismatch: file has %q, derived %q",
			validator.Address,
			validatorAddress,
		)
	}
	validatorPrivate, err := decodeCanonicalBase64("validator private key", validator.PrivKey.Value)
	if err != nil {
		return ConsensusIdentity{}, err
	}
	if validator.PrivKey.Type != cometEd25519PrivateKeyType {
		return ConsensusIdentity{}, fmt.Errorf("validator private key type must be Ed25519")
	}
	var validatorPublicFromPrivate []byte
	switch len(validatorPrivate) {
	case ed25519.SeedSize:
		derived := ed25519.NewKeyFromSeed(validatorPrivate)
		validatorPublicFromPrivate = derived.Public().(ed25519.PublicKey)
	case ed25519.PrivateKeySize:
		derived := ed25519.NewKeyFromSeed(validatorPrivate[:ed25519.SeedSize])
		if !bytes.Equal(validatorPrivate, derived) {
			return ConsensusIdentity{}, fmt.Errorf(
				"validator private key public suffix does not derive from its seed",
			)
		}
		validatorPublicFromPrivate = derived.Public().(ed25519.PublicKey)
	default:
		return ConsensusIdentity{}, fmt.Errorf("validator private key must be a 32- or 64-byte Ed25519 key")
	}
	if !ed25519.PublicKey(validatorPublicFromPrivate).Equal(ed25519.PublicKey(validatorPublic)) {
		return ConsensusIdentity{}, fmt.Errorf("validator private key does not match the declared public key")
	}

	nodePath := filepath.Join(home, "config", "node_key.json")
	nodeData, nodeInfo, err := readRegularFile(nodePath, maxControlFileBytes)
	if err != nil {
		return ConsensusIdentity{}, err
	}
	if nodeInfo.Mode().Perm()&0o077 != 0 {
		return ConsensusIdentity{}, fmt.Errorf("%s must not be group- or world-accessible", nodePath)
	}
	if links, err := fileLinkCount(nodeInfo); err != nil || links != 1 {
		return ConsensusIdentity{}, fmt.Errorf("%s must have exactly one hard link", nodePath)
	}
	var node nodeKeyDocument
	if err := decodeStrict(nodeData, &node); err != nil {
		return ConsensusIdentity{}, fmt.Errorf("node key: %w", err)
	}
	nodePrivate, err := decodeCanonicalBase64("node private key", node.PrivKey.Value)
	if err != nil {
		return ConsensusIdentity{}, err
	}
	var nodePublic []byte
	switch len(nodePrivate) {
	case ed25519.SeedSize:
		nodePublic = ed25519.NewKeyFromSeed(nodePrivate).Public().(ed25519.PublicKey)
	case ed25519.PrivateKeySize:
		derived := ed25519.NewKeyFromSeed(nodePrivate[:ed25519.SeedSize])
		if !bytes.Equal(nodePrivate, derived) {
			return ConsensusIdentity{}, fmt.Errorf(
				"node private key public suffix does not derive from its seed",
			)
		}
		nodePublic = derived.Public().(ed25519.PublicKey)
	default:
		return ConsensusIdentity{}, fmt.Errorf("node private key must be a 32- or 64-byte Ed25519 key")
	}
	if node.PrivKey.Type != cometEd25519PrivateKeyType {
		return ConsensusIdentity{}, fmt.Errorf("node private key type must be Ed25519")
	}
	nodeIDDigest := sha256.Sum256(nodePublic)
	validatorPublicDigest := sha256.Sum256(validatorPublic)
	nodePublicDigest := sha256.Sum256(nodePublic)
	return ConsensusIdentity{
		ConsensusAddress:      validatorAddress,
		ConsensusKeyType:      validator.PubKey.Type,
		ConsensusPubKeySHA256: hex.EncodeToString(validatorPublicDigest[:]),
		NodeID:                hex.EncodeToString(nodeIDDigest[:20]),
		NodeKeyType:           node.PrivKey.Type,
		NodePublicKeySHA256:   hex.EncodeToString(nodePublicDigest[:]),
	}, nil
}

func inspectSigningState(home string) (SigningState, error) {
	path := filepath.Join(home, "data", "priv_validator_state.json")
	data, info, err := readRegularFile(path, maxControlFileBytes)
	if err != nil {
		return SigningState{}, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return SigningState{}, fmt.Errorf("%s must not be group- or world-accessible", path)
	}
	if links, err := fileLinkCount(info); err != nil || links != 1 {
		return SigningState{}, fmt.Errorf("%s must have exactly one hard link", path)
	}
	var document signingStateDocument
	if err := decodeStrict(data, &document); err != nil {
		return SigningState{}, fmt.Errorf("signing state: %w", err)
	}
	height, err := parseCanonicalNonNegative("signing state height", document.Height)
	if err != nil {
		return SigningState{}, err
	}
	if document.Round < 0 || document.Step < 0 {
		return SigningState{}, fmt.Errorf("signing state round and step must be non-negative")
	}
	signatureDigest, signaturePresent, signatureLength, err := digestOptionalBase64(
		"signing state signature",
		document.Signature,
	)
	if err != nil {
		return SigningState{}, err
	}
	signBytesDigest, signBytesPresent, signBytesLength, err := digestOptionalBase64(
		"signing state signbytes",
		document.SignBytes,
	)
	if err != nil {
		return SigningState{}, err
	}
	stateDigest := sha256.Sum256(data)
	state := SigningState{
		Height:           height,
		Round:            document.Round,
		Step:             document.Step,
		SignaturePresent: signaturePresent,
		SignatureSHA256:  signatureDigest,
		SignBytesPresent: signBytesPresent,
		SignBytesSHA256:  signBytesDigest,
		StateFileSHA256:  hex.EncodeToString(stateDigest[:]),
	}
	if signaturePresent && signatureLength != ed25519.SignatureSize {
		return SigningState{}, fmt.Errorf("signing-state signature must be exactly 64 bytes")
	}
	if signBytesPresent && signBytesLength == 0 {
		return SigningState{}, fmt.Errorf("signing-state sign bytes must not be empty")
	}
	if err := validateSigningStateManifest(state); err != nil {
		return SigningState{}, err
	}
	return state, nil
}

func parseCanonicalNonNegative(label, text string) (int64, error) {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil || value < 0 || strconv.FormatInt(value, 10) != text {
		return 0, fmt.Errorf("%s must be a canonical non-negative decimal string", label)
	}
	return value, nil
}

func decodeCanonicalBase64(label, text string) ([]byte, error) {
	decoded, err := base64.StdEncoding.Strict().DecodeString(text)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != text {
		return nil, fmt.Errorf("%s must be canonical padded base64", label)
	}
	return decoded, nil
}

func digestOptionalBase64(label string, text *string) (string, bool, int, error) {
	if text == nil {
		return "", false, 0, nil
	}
	decoded, err := decodeCanonicalBase64(label, *text)
	if err != nil {
		return "", false, 0, err
	}
	digest := sha256.Sum256(decoded)
	return hex.EncodeToString(digest[:]), true, len(decoded), nil
}
