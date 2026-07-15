// Package vault reads a single secret field from a HashiCorp Vault or OpenBao
// KV v2 engine over HTTP. OpenBao is API-compatible, so the same client serves
// both; standard VAULT_* and BAO_* environment variables are honored by the
// caller. Only the read path is implemented — ypcli never writes to the store.
package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dantte-lp/ypcli/internal/api"
)

// ErrNotConfigured is returned when the address or token is missing.
var ErrNotConfigured = errors.New("vault: address and token are required")

// Client reads from a Vault/OpenBao KV v2 engine.
type Client struct {
	Addr      string
	Token     string
	Namespace string
	HTTP      *http.Client
}

// ReadField fetches mount/path from the KV v2 engine and returns the value of
// the given field. mount defaults to "secret" when empty. Transport and status
// failures are wrapped with the api sentinels so the CLI maps them to the same
// exit codes as yopass errors.
func (c Client) ReadField(ctx context.Context, mount, path, field string) (string, error) {
	if c.Addr == "" || c.Token == "" {
		return "", ErrNotConfigured
	}
	if mount == "" {
		mount = "secret"
	}
	httpc := c.HTTP
	if httpc == nil {
		httpc = http.DefaultClient
	}

	endpoint := fmt.Sprintf("%s/v1/%s/data/%s",
		strings.TrimSuffix(c.Addr, "/"), strings.Trim(mount, "/"), strings.Trim(path, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("vault: build request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.Token)
	if c.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.Namespace)
	}

	resp, err := httpc.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault: %w: %w", api.ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", statusError(resp)
	}

	// KV v2 wraps the payload as { "data": { "data": { <field>: <value> } } }.
	var body struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("vault: decode response: %w", err)
	}
	value, ok := body.Data.Data[field]
	if !ok {
		return "", fmt.Errorf("vault: field %q not found at %s/%s", field, mount, path)
	}
	if s, ok := value.(string); ok {
		return s, nil
	}
	return fmt.Sprint(value), nil
}

func statusError(resp *http.Response) error {
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusUnauthorized:
		return fmt.Errorf("vault: %w (status %d)", api.ErrUnauthorized, resp.StatusCode)
	case http.StatusNotFound:
		return fmt.Errorf("vault: %w: secret not found", api.ErrNotFound)
	default:
		return fmt.Errorf("vault: %w (status %d): %s", api.ErrServer, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
}
