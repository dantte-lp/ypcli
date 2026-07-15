package crypto

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	k1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if len(k1) != 22 {
		t.Errorf("key length = %d, want 22", len(k1))
	}
	k2, _ := GenerateKey()
	if k1 == k2 {
		t.Error("two generated keys are equal")
	}
}

func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID: %v", err)
	}
	if len(id) != 22 {
		t.Errorf("id length = %d, want 22", len(id))
	}
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	for _, c := range id {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("id contains non-base62 rune %q", c)
		}
	}
}

func TestEncryptDecryptText(t *testing.T) {
	const key = "test-key-123456789012"
	const plaintext = "top secret message"

	ct, err := Encrypt(strings.NewReader(plaintext), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !strings.HasPrefix(ct, armorHeader) {
		t.Errorf("ciphertext is not armored: %q", ct[:min(len(ct), 40)])
	}

	got, filename, err := Decrypt(strings.NewReader(ct), key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Errorf("plaintext = %q, want %q", got, plaintext)
	}
	if filename != "" {
		t.Errorf("filename = %q, want empty for text", filename)
	}
}

func TestEncryptDecryptTextArgon2(t *testing.T) {
	const key = "argon2-key-1234567890"
	ct, err := EncryptWithArgon2(strings.NewReader("hello argon"), key)
	if err != nil {
		t.Fatalf("EncryptWithArgon2: %v", err)
	}
	got, _, err := Decrypt(strings.NewReader(ct), key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "hello argon" {
		t.Errorf("plaintext = %q", got)
	}
}

func TestEncryptBinaryDecryptFile(t *testing.T) {
	const key = "file-key-12345678901234"
	const name = "secret.conf"
	const data = "binary file contents\x00\x01\x02"

	ct, err := EncryptBinary(strings.NewReader(data), key, name)
	if err != nil {
		t.Fatalf("EncryptBinary: %v", err)
	}
	got, filename, err := Decrypt(strings.NewReader(string(ct)), key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != data {
		t.Errorf("data mismatch")
	}
	if filename != name {
		t.Errorf("filename = %q, want %q", filename, name)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	ct, _ := Encrypt(strings.NewReader("x"), "right-key-1234567890ab")
	_, _, err := Decrypt(strings.NewReader(ct), "wrong-key-1234567890ab")
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("err = %v, want ErrInvalidKey", err)
	}
}

func TestEncryptEmptyKey(t *testing.T) {
	if _, err := Encrypt(strings.NewReader("x"), ""); !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Encrypt err = %v, want ErrEmptyKey", err)
	}
	if _, err := EncryptBinary(strings.NewReader("x"), "", "f"); !errors.Is(err, ErrEmptyKey) {
		t.Errorf("EncryptBinary err = %v, want ErrEmptyKey", err)
	}
}

func TestDecryptInvalidMessage(t *testing.T) {
	_, _, err := Decrypt(strings.NewReader("not a pgp message at all"), "some-key-1234567890ab")
	if !errors.Is(err, ErrInvalidMessage) {
		t.Errorf("err = %v, want ErrInvalidMessage", err)
	}
}

func TestExpirationSeconds(t *testing.T) {
	cases := map[string]struct {
		want int32
		ok   bool
	}{
		"1h":  {3600, true},
		"1d":  {86400, true},
		"1w":  {604800, true},
		"2w":  {0, false},
		"":    {0, false},
		"1mo": {0, false},
	}
	for in, want := range cases {
		got, ok := ExpirationSeconds(in)
		if got != want.want || ok != want.ok {
			t.Errorf("ExpirationSeconds(%q) = (%d,%v), want (%d,%v)", in, got, ok, want.want, want.ok)
		}
	}
	if !ValidExpirationSeconds(3600) || ValidExpirationSeconds(42) {
		t.Error("ValidExpirationSeconds mismatch")
	}
}

func TestSecretURL(t *testing.T) {
	const base = "https://yopass.se"
	// SecretURL only ever emits the s/f prefixes (matching upstream yopass).
	// Manual-key mode omits the key from the URL; the c/d prefixes are
	// produced by the web frontend and only appear on the parse side.
	cases := []struct {
		name      string
		fileOpt   bool
		manualKey bool
		want      string
	}{
		{"secret+key", false, false, base + "/#/s/abc/k1"},
		{"file+key", true, false, base + "/#/f/abc/k1"},
		{"secret manual", false, true, base + "/#/s/abc"},
		{"file manual", true, true, base + "/#/f/abc"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SecretURL(base, "abc", "k1", c.fileOpt, c.manualKey); got != c.want {
				t.Errorf("SecretURL = %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseURL(t *testing.T) {
	cases := []struct {
		name          string
		url           string
		wantID, wantK string
		fileOpt       bool
		keyOpt        bool
	}{
		{"secret+key", "https://y/#/s/abc/k1", "abc", "k1", false, false},
		{"file+key", "https://y/#/f/abc/k1", "abc", "k1", true, false},
		{"secret manual", "https://y/#/c/abc", "abc", "", false, true},
		{"file manual", "https://y/#/d/abc", "abc", "", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id, key, fileOpt, keyOpt, err := ParseURL(c.url)
			if err != nil {
				t.Fatalf("ParseURL: %v", err)
			}
			if id != c.wantID || key != c.wantK || fileOpt != c.fileOpt || keyOpt != c.keyOpt {
				t.Errorf("got (id=%q key=%q file=%v keyOpt=%v)", id, key, fileOpt, keyOpt)
			}
		})
	}
}

func TestParseURLInvalid(t *testing.T) {
	for _, s := range []string{"https://x/#/z/abc", "https://x/#/s", "not a url at all ::"} {
		if _, _, _, _, err := ParseURL(s); err == nil {
			t.Errorf("ParseURL(%q) expected error", s)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
