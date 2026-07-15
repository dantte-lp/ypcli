package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/clipboard"
	"github.com/dantte-lp/ypcli/internal/config"
	"github.com/dantte-lp/ypcli/internal/crypto"
	"github.com/dantte-lp/ypcli/internal/output"
	"github.com/dantte-lp/ypcli/internal/vault"
	"github.com/spf13/cobra"
)

func (a *app) newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Encrypt and publish a secret (text or file)",
		Long: "Encrypt text or a file client-side and publish it to a yopass server,\n" +
			"printing a one-time share URL. Input is taken from --file, --text, or stdin.",
		Example: "  printf 'secret' | ypcli send --one-time\n" +
			"  ypcli send --file db.env --expiration 1d --json\n" +
			"  ypcli send            # opens $EDITOR when run interactively\n" +
			"  ypcli send --vault-path secret/db --vault-field password",
		Args: cobra.NoArgs,
		RunE: a.runSend,
	}
	f := cmd.Flags()
	f.StringP("file", "f", "", "read secret from a file")
	f.StringP("text", "t", "", "secret text (instead of stdin/file)")
	f.Bool("editor", false, "compose the secret in $EDITOR (default when interactive)")
	f.StringP("expiration", "e", "", "lifetime: 1h, 1d or 1w")
	f.Bool("one-time", true, "delete after first view")
	f.Bool("require-auth", false, "require authentication to view (server support needed)")
	f.StringP("key", "k", "", "manual encryption key (omitted from the URL)")
	f.Bool("qr", false, "also render the URL as a terminal QR code")
	f.Bool("copy", false, "copy the URL to the system clipboard")
	// Read the secret payload from a Vault / OpenBao KV v2 engine.
	f.String("vault-path", "", "read the secret from a Vault/OpenBao KV v2 path")
	f.String("vault-field", "", "field to read from the Vault/OpenBao secret")
	f.String("vault-mount", "", "Vault/OpenBao KV v2 mount (default: secret)")
	f.String("vault-addr", "", "Vault/OpenBao address (default $VAULT_ADDR/$BAO_ADDR)")
	f.String("vault-token", "", "Vault/OpenBao token (default $VAULT_TOKEN/$BAO_TOKEN)")
	f.String("vault-namespace", "", "Vault/OpenBao namespace (default $VAULT_NAMESPACE/$BAO_NAMESPACE)")
	return cmd
}

func (a *app) runSend(cmd *cobra.Command, _ []string) error {
	s, err := a.resolve(cmd)
	if err != nil {
		return err
	}

	exp, err := resolveExpiration(cmd, s.profile)
	if err != nil {
		return err
	}
	oneTime := resolveOneTime(cmd, s.profile)
	requireAuth, _ := cmd.Flags().GetBool("require-auth")

	key, manualKey, err := resolveKey(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), s.timeout)
	defer cancel()

	token, err := config.ResolveToken(ctx, s.token, s.profile.TokenCommand)
	if err != nil {
		return err
	}
	client := newClient(s.api, token)
	useArgon2 := resolveArgon2(ctx, client, s.profile)
	s.log.Debug("sending secret", "api", s.api, "argon2", useArgon2,
		"one_time", oneTime, "expiration", expirationLabel(exp), "authenticated", token != "")

	vaultPath, _ := cmd.Flags().GetString("vault-path")
	filePath, _ := cmd.Flags().GetString("file")
	var (
		id      string
		fileOpt bool
	)
	switch {
	case vaultPath != "":
		var secret string
		secret, err = readFromVault(ctx, cmd, vaultPath, s.profile)
		if err == nil {
			id, err = sendMessage(ctx, client, stringReader(secret), key, exp, oneTime, requireAuth, useArgon2)
		}
	case filePath != "":
		id, err = sendFile(ctx, client, filePath, key, exp, oneTime, useArgon2)
		fileOpt = true
	default:
		var r io.Reader
		r, err = textReader(ctx, cmd)
		if err == nil {
			id, err = sendMessage(ctx, client, r, key, exp, oneTime, requireAuth, useArgon2)
		}
	}
	if err != nil {
		return err
	}

	shareURL := crypto.SecretURL(s.url, id, key, fileOpt, manualKey)
	return emitSend(cmd, s, output.SendResult{
		ID: id, URL: shareURL, Key: key, ManualKey: manualKey,
		File: fileOpt, OneTime: oneTime, Expiration: expirationLabel(exp),
	})
}

// sendMessage encrypts the plaintext from r and stores it as a text secret.
func sendMessage(ctx context.Context, client *api.Client, r io.Reader, key string, exp int32, oneTime, requireAuth, argon2 bool) (string, error) {
	enc := crypto.Encrypt
	if argon2 {
		enc = crypto.EncryptWithArgon2
	}
	msg, err := enc(r, key)
	if err != nil {
		return "", fmt.Errorf("encrypt secret: %w", err)
	}
	return client.CreateSecret(ctx, api.Secret{
		Message: msg, Expiration: exp, OneTime: oneTime, RequireAuth: requireAuth,
	})
}

// readFromVault fetches the secret payload from a Vault/OpenBao KV v2 engine.
// Connection settings resolve as flag > env (VAULT_*/BAO_*) > profile `vault`
// block; the token additionally falls back to the profile's vault token_command.
func readFromVault(ctx context.Context, cmd *cobra.Command, path string, prof config.Profile) (string, error) {
	field, _ := cmd.Flags().GetString("vault-field")
	if field == "" {
		return "", usage("--vault-field is required with --vault-path")
	}

	var pv config.VaultConfig
	if prof.Vault != nil {
		pv = *prof.Vault
	}

	addr := coalesce(changedString(cmd, "vault-addr"), os.Getenv("VAULT_ADDR"), os.Getenv("BAO_ADDR"), pv.Addr)
	namespace := coalesce(changedString(cmd, "vault-namespace"),
		os.Getenv("VAULT_NAMESPACE"), os.Getenv("BAO_NAMESPACE"), pv.Namespace)
	mount := coalesce(changedString(cmd, "vault-mount"), pv.Mount, "secret")

	explicitToken := coalesce(changedString(cmd, "vault-token"), os.Getenv("VAULT_TOKEN"), os.Getenv("BAO_TOKEN"))
	token, err := config.ResolveToken(ctx, explicitToken, pv.TokenCommand)
	if err != nil {
		return "", err
	}

	c := vault.Client{Addr: addr, Token: token, Namespace: namespace}
	val, err := c.ReadField(ctx, mount, path, field)
	if errors.Is(err, vault.ErrNotConfigured) {
		return "", usage("vault: set --vault-addr/--vault-token, VAULT_ADDR/VAULT_TOKEN, or a profile vault block")
	}
	return val, err
}

func sendFile(ctx context.Context, client *api.Client, path, key string, exp int32, oneTime, argon2 bool) (string, error) {
	f, err := os.Open(path) //nolint:gosec // path provided by the user by design
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	encBin := crypto.EncryptBinary
	if argon2 {
		encBin = crypto.EncryptBinaryWithArgon2
	}
	data, err := encBin(f, key, filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("encrypt file: %w", err)
	}
	return client.CreateFile(ctx, readerOf(data), exp, oneTime)
}

// textReader returns the plaintext source. Priority: --text, then piped stdin;
// when stdin is a terminal (or --editor is set) it opens the user's editor.
func textReader(ctx context.Context, cmd *cobra.Command) (io.Reader, error) {
	if text, _ := cmd.Flags().GetString("text"); text != "" {
		return stringReader(text), nil
	}

	forced, _ := cmd.Flags().GetBool("editor")
	interactive := false
	in := cmd.InOrStdin()
	if f, ok := in.(*os.File); ok {
		if info, err := f.Stat(); err == nil {
			interactive = info.Mode()&os.ModeCharDevice != 0
		}
	}

	if forced || interactive {
		content, err := openEditor(ctx)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(content) == "" {
			return nil, usage("empty secret; nothing to send")
		}
		return stringReader(content), nil
	}
	return in, nil
}

func emitSend(cmd *cobra.Command, s *settings, res output.SendResult) error {
	if copyFlag, _ := cmd.Flags().GetBool("copy"); copyFlag {
		if err := clipboard.Copy(res.URL); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning:", err)
		}
	}
	if qr, _ := cmd.Flags().GetBool("qr"); qr && !s.jsonMode {
		if art, err := output.QR(res.URL); err == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), art)
		}
	}
	return s.printer(cmd).Send(res)
}

// resolveExpiration picks the expiration flag if set, else the profile value,
// else the default, and validates it.
func resolveExpiration(cmd *cobra.Command, p config.Profile) (int32, error) {
	label := coalesce(changedString(cmd, "expiration"), p.Expiration, config.DefaultExpiration)
	seconds, ok := crypto.ExpirationSeconds(label)
	if !ok {
		return 0, usage("invalid expiration %q: use 1h, 1d or 1w", label)
	}
	return seconds, nil
}

func resolveOneTime(cmd *cobra.Command, p config.Profile) bool {
	if cmd.Flags().Changed("one-time") {
		v, _ := cmd.Flags().GetBool("one-time")
		return v
	}
	if p.OneTime != nil {
		return *p.OneTime
	}
	return true
}

func resolveKey(cmd *cobra.Command) (key string, manual bool, err error) {
	if k, _ := cmd.Flags().GetString("key"); k != "" {
		return k, true, nil
	}
	k, err := crypto.GenerateKey()
	if err != nil {
		return "", false, fmt.Errorf("generate key: %w", err)
	}
	return k, false, nil
}

// resolveArgon2 uses the profile override when set, else asks the server via
// /config. A failed lookup falls back to the default (non-Argon2) derivation,
// which every yopass server accepts.
func resolveArgon2(ctx context.Context, client *api.Client, p config.Profile) bool {
	if p.Argon2 != nil {
		return *p.Argon2
	}
	if cfg, err := client.Config(ctx); err == nil {
		return cfg.Argon2
	}
	return false
}

func changedString(cmd *cobra.Command, name string) string {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetString(name)
		return v
	}
	return ""
}

func expirationLabel(seconds int32) string {
	switch seconds {
	case 3600:
		return "1h"
	case 86400:
		return "1d"
	case 604800:
		return "1w"
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}
