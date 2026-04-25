package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the YAML configuration shape.
//
// Example:
//
//	listen_address: ":8443"
//	private_key_path: "/etc/corpus-vault/operator.pem"
//	vault_id: "love-corpus"
//	items_root: "/var/lib/corpus-vault/items"
//	auth_mode: "signed-challenge"
//	allowed_client_keys:
//	  - "ed25519:deadbeef..."
//	nonce_ttl: "5m"
//	access_log_path: "/var/lib/corpus-vault/access.log"
//	manifests:
//	  - id: "love-corpus#1"
//	    version: "1.0"
//	    description: "Initial release."
//	    item_root: "v1"        # subdir of items_root
type Config struct {
	ListenAddress     string         `yaml:"listen_address"`
	PrivateKeyPath    string         `yaml:"private_key_path"`
	VaultID           string         `yaml:"vault_id"`
	ItemsRoot         string         `yaml:"items_root"`
	AuthMode          AuthMode       `yaml:"auth_mode"`
	AllowedClientKeys []string       `yaml:"allowed_client_keys"`
	NonceTTL          time.Duration  `yaml:"nonce_ttl"`
	AccessLogPath     string         `yaml:"access_log_path"`
	Manifests         []ManifestSpec `yaml:"manifests"`
}

// ManifestSpec is one manifest the operator wants to serve. The server
// computes the manifest body at startup (or on demand) by walking
// ItemsRoot/ItemRoot.
type ManifestSpec struct {
	ID          string `yaml:"id"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
	ItemRoot    string `yaml:"item_root"` // relative to Config.ItemsRoot
}

// LoadConfig reads and validates a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &c, nil
}

func (c *Config) Validate() error {
	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address required")
	}
	if c.PrivateKeyPath == "" {
		return fmt.Errorf("private_key_path required")
	}
	if c.VaultID == "" {
		return fmt.Errorf("vault_id required")
	}
	if c.ItemsRoot == "" {
		return fmt.Errorf("items_root required")
	}
	switch c.AuthMode {
	case "":
		c.AuthMode = AuthModePublic
	case AuthModePublic:
	case AuthModeSignedChallenge:
		if len(c.AllowedClientKeys) == 0 {
			return fmt.Errorf("auth_mode=signed-challenge requires allowed_client_keys")
		}
	default:
		return fmt.Errorf("unknown auth_mode %q (expected public|signed-challenge)", c.AuthMode)
	}
	if len(c.Manifests) == 0 {
		return fmt.Errorf("at least one manifest is required")
	}
	seen := map[string]bool{}
	for i, m := range c.Manifests {
		if m.ID == "" {
			return fmt.Errorf("manifests[%d].id required", i)
		}
		if seen[m.ID] {
			return fmt.Errorf("duplicate manifest id %q", m.ID)
		}
		seen[m.ID] = true
		if m.ItemRoot == "" {
			return fmt.Errorf("manifests[%d].item_root required", i)
		}
	}
	return nil
}
