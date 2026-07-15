package crypto_test

// This file proves bidirectional interoperability with the upstream yopass
// implementation. github.com/jhaals/yopass is imported ONLY here, in a _test
// file, so it never links into the shipped ypcli binary.

import (
	"bytes"
	"strings"
	"testing"

	ypc "github.com/dantte-lp/ypcli/internal/crypto"
	"github.com/jhaals/yopass/pkg/yopass"
)

const interopKey = "interop-key-1234567890"

// ypcli encrypts -> upstream yopass decrypts (text).
func TestInteropYpcliToYopassText(t *testing.T) {
	const msg = "cross-implementation secret"
	ct, err := ypc.Encrypt(strings.NewReader(msg), interopKey)
	if err != nil {
		t.Fatalf("ypcli Encrypt: %v", err)
	}
	got, _, err := yopass.Decrypt(strings.NewReader(ct), interopKey)
	if err != nil {
		t.Fatalf("yopass Decrypt: %v", err)
	}
	if got != msg {
		t.Errorf("yopass decrypted %q, want %q", got, msg)
	}
}

// upstream yopass encrypts -> ypcli decrypts (text).
func TestInteropYopassToYpcliText(t *testing.T) {
	const msg = "produced by upstream"
	ct, err := yopass.Encrypt(strings.NewReader(msg), interopKey)
	if err != nil {
		t.Fatalf("yopass Encrypt: %v", err)
	}
	got, _, err := ypc.Decrypt(strings.NewReader(ct), interopKey)
	if err != nil {
		t.Fatalf("ypcli Decrypt: %v", err)
	}
	if got != msg {
		t.Errorf("ypcli decrypted %q, want %q", got, msg)
	}
}

// ypcli binary file encrypt -> upstream yopass decrypts, filename preserved.
func TestInteropYpcliToYopassFile(t *testing.T) {
	const name = "creds.env"
	const data = "USER=admin\nPASS=hunter2\n"
	ct, err := ypc.EncryptBinary(strings.NewReader(data), interopKey, name)
	if err != nil {
		t.Fatalf("ypcli EncryptBinary: %v", err)
	}
	got, filename, err := yopass.Decrypt(bytes.NewReader(ct), interopKey)
	if err != nil {
		t.Fatalf("yopass Decrypt: %v", err)
	}
	if got != data {
		t.Errorf("data mismatch")
	}
	if filename != name {
		t.Errorf("filename = %q, want %q", filename, name)
	}
}

// Argon2 ciphertext from ypcli decrypts under upstream yopass (S2K auto-detected).
func TestInteropArgon2(t *testing.T) {
	const msg = "argon2 interop"
	ct, err := ypc.EncryptWithArgon2(strings.NewReader(msg), interopKey)
	if err != nil {
		t.Fatalf("ypcli EncryptWithArgon2: %v", err)
	}
	got, _, err := yopass.Decrypt(strings.NewReader(ct), interopKey)
	if err != nil {
		t.Fatalf("yopass Decrypt: %v", err)
	}
	if got != msg {
		t.Errorf("decrypted %q, want %q", got, msg)
	}
}
