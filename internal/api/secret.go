package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Secret is the JSON payload for POST /create/secret.
type Secret struct {
	Message     string `json:"message"`
	Expiration  int32  `json:"expiration,omitempty"`
	OneTime     bool   `json:"one_time,omitempty"`
	RequireAuth bool   `json:"require_auth,omitempty"`
}

// CreateSecret stores an encrypted text secret and returns its ID.
func (c *Client) CreateSecret(ctx context.Context, s Secret) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode secret: %w", err)
	}
	return c.message(ctx, http.MethodPost, "/create/secret", bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
}

// FetchSecret retrieves the armored ciphertext of a text secret by ID.
func (c *Client) FetchSecret(ctx context.Context, id string) (string, error) {
	return c.message(ctx, http.MethodGet, "/secret/"+id, nil, nil)
}
