package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ServerConfig is the subset of GET /config relevant to the CLI. Unknown
// fields are ignored so newer and older servers stay compatible.
type ServerConfig struct {
	Argon2              bool   `json:"ARGON2"`
	ReadOnly            bool   `json:"READ_ONLY"`
	DisableUpload       bool   `json:"DISABLE_UPLOAD"`
	ForceOneTimeSecrets bool   `json:"FORCE_ONETIME_SECRETS"`
	DefaultExpiry       int32  `json:"DEFAULT_EXPIRY"`
	ForceExpiration     int32  `json:"FORCE_EXPIRATION"`
	MaxFileSize         string `json:"MAX_FILE_SIZE"`
}

// Config fetches the server's public configuration.
func (c *Client) Config(ctx context.Context) (ServerConfig, error) {
	var cfg ServerConfig
	resp, err := c.do(ctx, http.MethodGet, "/config", nil, nil)
	if err != nil {
		return cfg, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return cfg, serverError(resp.StatusCode, raw)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

// Version fetches the server version. Servers without a /version endpoint
// (pre-#3374) return ErrUnsupported.
func (c *Client) Version(ctx context.Context) (string, error) {
	resp, err := c.do(ctx, http.MethodGet, "/version", nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrUnsupported
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", serverError(resp.StatusCode, raw)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("decode version: %w", err)
	}
	if v.Version == "" {
		return "", errors.New("empty version in response")
	}
	return v.Version, nil
}
