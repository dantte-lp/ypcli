// Package crypto implements the OpenPGP symmetric encryption used by yopass,
// plus the key/ID generation and share-URL helpers. The packet configuration
// mirrors the yopass server and the openpgp.js frontend exactly so that
// secrets produced here decrypt in the browser and vice versa.
//
// This is a deliberately vendored ~150-line surface: it depends only on
// github.com/ProtonMail/go-crypto, keeping the shipped binary's supply chain
// minimal. Bidirectional interoperability with the upstream implementation is
// proven by interop_test.go, which uses github.com/jhaals/yopass as a
// test-only dependency and is excluded from production builds.
package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/ProtonMail/go-crypto/openpgp/s2k"
)

// Sentinel errors returned by the package.
var (
	// ErrEmptyKey is returned when no encryption key is provided.
	ErrEmptyKey = errors.New("empty encryption key")
	// ErrInvalidKey is returned when a decryption key is wrong.
	ErrInvalidKey = errors.New("invalid decryption key")
	// ErrInvalidMessage is returned when the ciphertext cannot be parsed.
	ErrInvalidMessage = errors.New("invalid message")
)

const armorHeader = "-----BEGIN PGP MESSAGE-----"

// pgpConfig is the default configuration: AES-256, SHA-256, no compression,
// AEAD GCM. It matches yopass server and openpgp.js defaults.
var pgpConfig = &packet.Config{
	DefaultHash:            crypto.SHA256,
	DefaultCipher:          packet.CipherAES256,
	DefaultCompressionAlgo: packet.CompressionNone,
	AEADConfig:             &packet.AEADConfig{DefaultMode: packet.AEADModeGCM},
}

// pgpConfigArgon2 mirrors pgpConfig but switches S2K key derivation to
// memory-hard Argon2id. Used only against servers advertising ARGON2 support.
var pgpConfigArgon2 = func() *packet.Config {
	cfg := *pgpConfig
	cfg.S2KConfig = &s2k.Config{S2KMode: s2k.Argon2S2K}
	return &cfg
}()

var pgpHeader = map[string]string{"Comment": "https://yopass.se"}

// expirations is the single source of truth for supported secret lifetimes.
var expirations = map[string]int32{
	"1h": 3600,
	"1d": 86400,
	"1w": 604800,
}

// ExpirationSeconds converts a human-readable duration ("1h", "1d", "1w") to
// seconds. ok is false for unsupported values.
func ExpirationSeconds(s string) (seconds int32, ok bool) {
	seconds, ok = expirations[s]
	return seconds, ok
}

// ValidExpirationSeconds reports whether seconds is a supported lifetime.
func ValidExpirationSeconds(seconds int32) bool {
	for _, ttl := range expirations {
		if ttl == seconds {
			return true
		}
	}
	return false
}

// Encrypt reads plaintext from r and returns ASCII-armored ciphertext.
func Encrypt(r io.Reader, key string) (string, error) {
	return encrypt(r, key, pgpConfig)
}

// EncryptWithArgon2 is Encrypt using Argon2id key derivation.
func EncryptWithArgon2(r io.Reader, key string) (string, error) {
	return encrypt(r, key, pgpConfigArgon2)
}

func encrypt(r io.Reader, key string, config *packet.Config) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}

	var hints *openpgp.FileHints
	if f, ok := r.(*os.File); ok && r != os.Stdin {
		if stat, err := f.Stat(); err == nil {
			hints = &openpgp.FileHints{
				IsBinary: true,
				FileName: stat.Name(),
				ModTime:  stat.ModTime(),
			}
		}
	}

	buf := new(bytes.Buffer)
	a, err := armor.Encode(buf, "PGP MESSAGE", pgpHeader)
	if err != nil {
		return "", fmt.Errorf("create armor encoder: %w", err)
	}
	w, err := openpgp.SymmetricallyEncrypt(a, []byte(key), hints, config)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}
	if _, err := io.Copy(w, r); err != nil {
		return "", fmt.Errorf("copy plaintext: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close writer: %w", err)
	}
	if err := a.Close(); err != nil {
		return "", fmt.Errorf("close armor: %w", err)
	}
	return buf.String(), nil
}

// EncryptBinary encrypts r as binary (non-armored) PGP, embedding filename in
// the OpenPGP literal-data metadata. Used for streaming file uploads.
func EncryptBinary(r io.Reader, key, filename string) ([]byte, error) {
	return encryptBinary(r, key, filename, pgpConfig)
}

// EncryptBinaryWithArgon2 is EncryptBinary using Argon2id key derivation.
func EncryptBinaryWithArgon2(r io.Reader, key, filename string) ([]byte, error) {
	return encryptBinary(r, key, filename, pgpConfigArgon2)
}

func encryptBinary(r io.Reader, key, filename string, config *packet.Config) ([]byte, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}
	hints := &openpgp.FileHints{IsBinary: true, FileName: filename}

	buf := new(bytes.Buffer)
	w, err := openpgp.SymmetricallyEncrypt(buf, []byte(key), hints, config)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	if _, err := io.Copy(w, r); err != nil {
		return nil, fmt.Errorf("copy data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}
	return buf.Bytes(), nil
}

// Decrypt reads ciphertext from r and returns the plaintext decrypted with
// key. It auto-detects armored versus binary PGP. For binary literal data the
// embedded filename is returned.
func Decrypt(r io.Reader, key string) (content, filename string, err error) {
	if key == "" {
		return "", "", ErrEmptyKey
	}

	tried := false
	prompt := func([]openpgp.Key, bool) ([]byte, error) {
		if tried {
			return nil, ErrInvalidKey
		}
		tried = true
		return []byte(key), nil
	}

	// Peek enough bytes to detect the armor header, then reconstruct a reader.
	head := make([]byte, len(armorHeader))
	n, readErr := io.ReadFull(r, head)
	combined := io.MultiReader(bytes.NewReader(head[:n]), r)

	msgReader := combined
	if readErr == nil && string(head) == armorHeader {
		a, decErr := armor.Decode(combined)
		if decErr != nil {
			return "", "", ErrInvalidMessage
		}
		msgReader = a.Body
	}

	m, err := openpgp.ReadMessage(msgReader, nil, prompt, pgpConfig)
	if err != nil {
		if errors.Is(err, ErrInvalidKey) {
			return "", "", fmt.Errorf("decrypt: %w", ErrInvalidKey)
		}
		return "", "", ErrInvalidMessage
	}
	p, err := io.ReadAll(m.UnverifiedBody)
	if err != nil {
		return "", "", fmt.Errorf("read plaintext: %w", err)
	}
	if m.LiteralData != nil && m.LiteralData.IsBinary {
		filename = m.LiteralData.FileName
	}
	return string(p), filename, nil
}

// GenerateKey returns a 22-character base64url key, matching the yopass
// JavaScript and CLI implementations.
func GenerateKey() (string, error) {
	const length = 22
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}

// GenerateID returns a 22-character base62 identifier from 16 random bytes
// (~128 bits of entropy), matching the yopass server implementation.
func GenerateID() (string, error) {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	n := new(big.Int).SetBytes(b)
	result := make([]byte, 22)
	base := big.NewInt(62)
	mod := new(big.Int)
	for i := 21; i >= 0; i-- {
		n.DivMod(n, base, mod)
		result[i] = charset[mod.Int64()]
	}
	return string(result), nil
}

// SecretURL builds a browser share URL for the given secret. fileOpt selects
// the file prefix; manualKeyOpt omits the key from the URL (recipient must
// supply it out of band).
func SecretURL(baseURL, id, key string, fileOpt, manualKeyOpt bool) string {
	prefix := "s"
	if fileOpt {
		prefix = "f"
	}
	path := id
	if !manualKeyOpt {
		path += "/" + key
	}
	return fmt.Sprintf("%s/#/%s/%s", strings.TrimSuffix(baseURL, "/"), prefix, path)
}

// ParseURL extracts the secret ID and key from a yopass share URL. fileOpt is
// true for file secrets; keyOpt is true when the URL carries no key.
func ParseURL(s string) (id, key string, fileOpt, keyOpt bool, err error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return "", "", false, false, fmt.Errorf("invalid URL: %w", err)
	}

	f := strings.Split(u.Fragment, "/")
	if len(f) < 3 || len(f) > 4 || f[0] != "" {
		return "", "", false, false, fmt.Errorf("unexpected URL: %q", s)
	}

	switch f[1] {
	case "s":
	case "c":
		keyOpt = true
	case "f":
		fileOpt = true
	case "d":
		fileOpt = true
		keyOpt = true
	default:
		return "", "", false, false, fmt.Errorf("unexpected URL: %q", s)
	}

	id = f[2]
	if len(f) == 4 {
		key = f[3]
	}
	return id, key, fileOpt, keyOpt, nil
}
