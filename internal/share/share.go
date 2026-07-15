// Package share holds the transport-agnostic core of publishing and fetching
// yopass secrets: key generation, Argon2 selection, encryption, the API calls
// and share-URL assembly. Both the cobra CLI and the MCP server build on it so
// the behavior is identical regardless of entry point.
package share

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/crypto"
)

// Options controls a send operation.
type Options struct {
	Key         string // manual key; empty means generate one
	Expiration  int32  // seconds (1h/1d/1w)
	OneTime     bool
	RequireAuth bool
	Argon2      *bool // nil = auto-detect from the server /config
}

// SendResult describes a published secret.
type SendResult struct {
	ID         string
	URL        string
	Key        string
	ManualKey  bool
	File       bool
	OneTime    bool
	Expiration string
}

// SendText encrypts the plaintext from r and publishes it as a text secret.
func SendText(ctx context.Context, client *api.Client, publicURL string, r io.Reader, o Options) (SendResult, error) {
	key, manual, err := key(o.Key)
	if err != nil {
		return SendResult{}, err
	}
	enc := crypto.Encrypt
	if useArgon2(ctx, client, o.Argon2) {
		enc = crypto.EncryptWithArgon2
	}
	msg, err := enc(r, key)
	if err != nil {
		return SendResult{}, fmt.Errorf("encrypt secret: %w", err)
	}
	id, err := client.CreateSecret(ctx, api.Secret{
		Message: msg, Expiration: o.Expiration, OneTime: o.OneTime, RequireAuth: o.RequireAuth,
	})
	if err != nil {
		return SendResult{}, err
	}
	return result(id, key, publicURL, manual, false, o), nil
}

// SendFile encrypts the file at path and publishes it as a file secret.
func SendFile(ctx context.Context, client *api.Client, publicURL, path string, o Options) (SendResult, error) {
	key, manual, err := key(o.Key)
	if err != nil {
		return SendResult{}, err
	}
	f, err := os.Open(path) //nolint:gosec // path provided by the user by design
	if err != nil {
		return SendResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	encBin := crypto.EncryptBinary
	if useArgon2(ctx, client, o.Argon2) {
		encBin = crypto.EncryptBinaryWithArgon2
	}
	data, err := encBin(f, key, filepath.Base(path))
	if err != nil {
		return SendResult{}, fmt.Errorf("encrypt file: %w", err)
	}
	id, err := client.CreateFile(ctx, bytes.NewReader(data), o.Expiration, o.OneTime)
	if err != nil {
		return SendResult{}, err
	}
	return result(id, key, publicURL, manual, true, o), nil
}

// Target identifies a secret to receive.
type Target struct {
	ID   string
	Key  string
	File bool
	// Wrap optionally wraps the file download stream (e.g. a progress reader).
	Wrap func(r io.Reader, total int64) io.Reader
}

// ReceiveResult is the decrypted secret.
type ReceiveResult struct {
	Content  string
	Filename string
	File     bool
}

// Receive fetches and decrypts a secret.
func Receive(ctx context.Context, client *api.Client, t Target) (ReceiveResult, error) {
	if t.File {
		body, size, err := client.FetchFile(ctx, t.ID)
		if err != nil {
			return ReceiveResult{}, err
		}
		defer body.Close()

		var src io.Reader = body
		if t.Wrap != nil {
			src = t.Wrap(body, size)
		}
		plaintext, filename, err := crypto.Decrypt(src, t.Key)
		if err != nil {
			return ReceiveResult{}, err
		}
		return ReceiveResult{Content: plaintext, Filename: filename, File: true}, nil
	}

	msg, err := client.FetchSecret(ctx, t.ID)
	if err != nil {
		return ReceiveResult{}, err
	}
	plaintext, _, err := crypto.Decrypt(strings.NewReader(msg), t.Key)
	if err != nil {
		return ReceiveResult{}, err
	}
	return ReceiveResult{Content: plaintext, File: false}, nil
}

// key returns the encryption key: the manual key if given, else a fresh one.
func key(manual string) (k string, isManual bool, err error) {
	if manual != "" {
		return manual, true, nil
	}
	k, err = crypto.GenerateKey()
	if err != nil {
		return "", false, fmt.Errorf("generate key: %w", err)
	}
	return k, false, nil
}

// useArgon2 honors an explicit override, else asks the server /config; a failed
// lookup falls back to the default derivation, which every server accepts.
func useArgon2(ctx context.Context, client *api.Client, override *bool) bool {
	if override != nil {
		return *override
	}
	if cfg, err := client.Config(ctx); err == nil {
		return cfg.Argon2
	}
	return false
}

func result(id, k, publicURL string, manual, file bool, o Options) SendResult {
	return SendResult{
		ID:         id,
		URL:        crypto.SecretURL(publicURL, id, k, file, manual),
		Key:        k,
		ManualKey:  manual,
		File:       file,
		OneTime:    o.OneTime,
		Expiration: crypto.ExpirationLabel(o.Expiration),
	}
}
