package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ManifestItem describes one item in a manifest. The shape on the wire
// matches PROTOCOL.md: clients hash the same canonical JSON the server
// signs, and any drift breaks verification.
type ManifestItem struct {
	ID        string `json:"id"`
	SizeBytes uint64 `json:"size_bytes"`
	Hash      string `json:"hash"` // hex SHA-256 of the item bytes
	URL       string `json:"url"`  // relative URL clients follow
}

// ManifestBody is the unsigned body that gets hashed and signed.
// Clients hash the same canonical JSON to verify the on-chain
// content_hash; the server signs the same canonical JSON for
// authenticity.
type ManifestBody struct {
	ManifestID string         `json:"manifest_id"`
	VaultID    string         `json:"vault_id"`
	Items      []ManifestItem `json:"items"`
}

// SignedManifest is what the server returns over the wire.
type SignedManifest struct {
	ManifestBody
	// Hex-encoded ed25519 signature over CanonicalJSON(ManifestBody).
	Signature string `json:"signature"`
}

// CanonicalJSON returns the canonical JSON encoding used for hashing
// and signing. Standard library json.Marshal of a struct is canonical
// when fields are tagged in a fixed order — which they are above.
// We also explicitly sort items by ID so manifest construction
// produces a deterministic byte sequence regardless of read order.
func CanonicalJSON(m ManifestBody) ([]byte, error) {
	sorted := make([]ManifestItem, len(m.Items))
	copy(sorted, m.Items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	canonical := ManifestBody{
		ManifestID: m.ManifestID,
		VaultID:    m.VaultID,
		Items:      sorted,
	}
	return json.Marshal(canonical)
}

// ContentHash returns the hex-encoded SHA-256 of the canonical JSON.
// This is what the operator publishes on-chain via MsgPublishManifest.
// Clients re-compute this value over the manifest they receive and
// compare against the on-chain record.
func ContentHash(m ManifestBody) (string, error) {
	bz, err := CanonicalJSON(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bz)
	return hex.EncodeToString(sum[:]), nil
}

// HashFile returns the hex-encoded SHA-256 of a file's contents.
// Used to populate per-item Hash fields when building a manifest from
// a directory of files.
func HashFile(path string) (string, uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), uint64(len(data)), nil
}

// BuildManifestFromDir walks a directory, hashes every regular file
// (recursively) into a ManifestItem, and returns the populated body.
// The item ID is the file's path relative to the root.
//
// itemURLPrefix is prepended to each item's URL. Typically
// "/item/" so a client receiving the manifest can fetch
// `<server>/item/<path-relative-to-root>` to retrieve the file.
func BuildManifestFromDir(manifestID, vaultID, root, itemURLPrefix string) (ManifestBody, error) {
	body := ManifestBody{
		ManifestID: manifestID,
		VaultID:    vaultID,
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Skip dotfiles.
		if name := filepath.Base(path); len(name) > 0 && name[0] == '.' {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// On Windows, filepath uses backslash; normalise to forward slash
		// so item IDs are stable across platforms.
		rel = filepath.ToSlash(rel)
		hash, size, err := HashFile(path)
		if err != nil {
			return err
		}
		body.Items = append(body.Items, ManifestItem{
			ID:        rel,
			SizeBytes: size,
			Hash:      hash,
			URL:       itemURLPrefix + rel,
		})
		return nil
	})
	if err != nil {
		return ManifestBody{}, err
	}
	if len(body.Items) == 0 {
		return ManifestBody{}, fmt.Errorf("no items found under %s", root)
	}
	return body, nil
}
