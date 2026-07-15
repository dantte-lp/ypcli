// Package config models ypcli's on-disk configuration: named server profiles
// persisted as YAML, resolution of the effective settings, and non-persistent
// token sourcing via an external command.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Public yopass defaults, used when no profile overrides them.
const (
	DefaultAPI        = "https://api.yopass.se"
	DefaultURL        = "https://yopass.se"
	DefaultExpiration = "1h"
)

// Profile describes how to reach one yopass instance.
type Profile struct {
	API          string `yaml:"api,omitempty"`
	URL          string `yaml:"url,omitempty"`
	Expiration   string `yaml:"expiration,omitempty"`
	OneTime      *bool  `yaml:"one_time,omitempty"`
	Argon2       *bool  `yaml:"argon2,omitempty"`
	TokenCommand string `yaml:"token_command,omitempty"`
}

// Config is the root document persisted to disk.
type Config struct {
	Active   string             `yaml:"active,omitempty"`
	Profiles map[string]Profile `yaml:"profiles,omitempty"`
}

// DefaultProfile returns the built-in profile targeting the public yopass.
func DefaultProfile() Profile {
	return Profile{API: DefaultAPI, URL: DefaultURL, Expiration: DefaultExpiration}
}

// DefaultPath returns $XDG_CONFIG_HOME/ypcli/config.yaml, falling back to
// ~/.config/ypcli/config.yaml.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "ypcli", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ypcli", "config.yaml"), nil
}

// Load reads the configuration from path. A missing file yields an empty
// Config (not an error) so first-run usage works without setup.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is user-controlled by design
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Profiles: map[string]Profile{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	return &c, nil
}

// Save writes the configuration to path, creating parent directories. The
// file is written with 0600 permissions as it may reference token sources.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Profile returns the named profile, or the active one when name is empty.
// When neither is set it returns the public default profile.
func (c *Config) Profile(name string) (Profile, error) {
	if name == "" {
		name = c.Active
	}
	if name == "" {
		return DefaultProfile(), nil
	}
	p, ok := c.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown profile %q", name)
	}
	return p, nil
}

// Overlay returns a copy of p with every non-zero field of o applied on top.
// It implements the profile < flag/env precedence merge.
func (p Profile) Overlay(o Profile) Profile {
	if o.API != "" {
		p.API = o.API
	}
	if o.URL != "" {
		p.URL = o.URL
	}
	if o.Expiration != "" {
		p.Expiration = o.Expiration
	}
	if o.OneTime != nil {
		p.OneTime = o.OneTime
	}
	if o.Argon2 != nil {
		p.Argon2 = o.Argon2
	}
	if o.TokenCommand != "" {
		p.TokenCommand = o.TokenCommand
	}
	return p
}

// ResolveToken returns the bearer token for a request. An explicit token
// (from --token or YPCLI_TOKEN) wins; otherwise tokenCommand is executed and
// its trimmed stdout is used. Tokens are never persisted to disk.
func ResolveToken(ctx context.Context, explicit, tokenCommand string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if tokenCommand == "" {
		return "", nil
	}

	name, args := shell()
	cmd := exec.CommandContext(ctx, name, append(args, tokenCommand)...) //nolint:gosec // user-configured
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("token_command failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// shell returns the OS shell invocation used to run token_command.
func shell() (name string, args []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c"}
	}
	return "sh", []string{"-c"}
}
