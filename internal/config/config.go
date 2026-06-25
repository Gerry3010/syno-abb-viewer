// Package config holds the connection settings for a Synology DiskStation and
// persists them (minus the password) under the user's config directory.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// AuthMethod selects how the SSH connection authenticates.
type AuthMethod string

const (
	AuthKey      AuthMethod = "key"      // public-key auth (KeyPath); Password is the optional passphrase
	AuthPassword AuthMethod = "password" // password auth (Password)
)

// Config is everything needed to reach the DiskStation and where to start browsing.
//
// Password is deliberately never written to disk (json:"-"): SSH passwords and
// key passphrases are prompted on each connect. Only the non-secret fields persist.
type Config struct {
	Host     string     `json:"host"`
	Port     int        `json:"port"`
	User     string     `json:"user"`
	Auth     AuthMethod `json:"auth"`
	KeyPath  string     `json:"key_path"`
	RootPath string     `json:"root_path"`
	Password string     `json:"-"`
}

// Default returns sensible starting values: standard SSH port, the common
// ed25519 key, and /volume1 (the typical Synology data volume root).
func Default() Config {
	key := "~/.ssh/id_ed25519"
	if home, err := os.UserHomeDir(); err == nil {
		key = filepath.Join(home, ".ssh", "id_ed25519")
	}
	return Config{
		Port:     22,
		Auth:     AuthKey,
		KeyPath:  key,
		RootPath: "/volume1",
	}
}

// Path is the on-disk location of the persisted config.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "syno-abb-viewer", "config.json"), nil
}

// Load reads the persisted config, falling back to Default for any missing file.
// A missing file is not an error — it just means "first run".
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Default(), err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}

// Save writes the config (without the password) to disk with 0600 permissions.
func Save(cfg Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
