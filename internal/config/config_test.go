package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadMissingReturnsEmpty(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Profiles) != 0 || c.Active != "" {
		t.Errorf("expected empty config, got %+v", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	in := &Config{
		Active: "work",
		Profiles: map[string]Profile{
			"work": {
				API:          "https://api.corp",
				URL:          "https://yp.corp",
				Expiration:   "1d",
				OneTime:      new(false),
				Argon2:       new(true),
				TokenCommand: "vault read -field=token secret/yopass",
			},
		},
	}
	if err := in.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := out.Profile("work")
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	want := in.Profiles["work"]
	if got.API != want.API || got.URL != want.URL || got.Expiration != want.Expiration ||
		got.OneTime == nil || *got.OneTime != false || got.Argon2 == nil || *got.Argon2 != true ||
		got.TokenCommand != want.TokenCommand {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	c := &Config{Profiles: map[string]Profile{"x": {API: "a"}}}
	if err := c.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

func TestProfileSelection(t *testing.T) {
	c := &Config{
		Active:   "a",
		Profiles: map[string]Profile{"a": {API: "aa"}, "b": {API: "bb"}},
	}
	if p, _ := c.Profile(""); p.API != "aa" {
		t.Errorf("active profile API = %q, want aa", p.API)
	}
	if p, _ := c.Profile("b"); p.API != "bb" {
		t.Errorf("named profile API = %q, want bb", p.API)
	}
	if _, err := c.Profile("missing"); err == nil {
		t.Error("expected error for missing profile")
	}

	empty := &Config{Profiles: map[string]Profile{}}
	if p, _ := empty.Profile(""); p.API != DefaultAPI {
		t.Errorf("fallback API = %q, want default", p.API)
	}
}

func TestOverlayPrecedence(t *testing.T) {
	base := Profile{API: "base-api", URL: "base-url", Expiration: "1h", OneTime: new(true)}
	over := Profile{API: "over-api", Argon2: new(true)}
	got := base.Overlay(over)

	if got.API != "over-api" {
		t.Errorf("API = %q, want override to win", got.API)
	}
	if got.URL != "base-url" || got.Expiration != "1h" {
		t.Errorf("unset override fields should keep base: %+v", got)
	}
	if got.OneTime == nil || !*got.OneTime {
		t.Error("base OneTime should be preserved")
	}
	if got.Argon2 == nil || !*got.Argon2 {
		t.Error("override Argon2 should be applied")
	}
}

func TestResolveTokenExplicitWins(t *testing.T) {
	tok, err := ResolveToken(context.Background(), "explicit-token", "echo should-not-run")
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if tok != "explicit-token" {
		t.Errorf("token = %q, want explicit-token", tok)
	}
}

func TestResolveTokenFromCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	tok, err := ResolveToken(context.Background(), "", "printf 'cmd-token\n'")
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if tok != "cmd-token" {
		t.Errorf("token = %q, want cmd-token (trimmed)", tok)
	}
}

func TestResolveTokenEmpty(t *testing.T) {
	tok, err := ResolveToken(context.Background(), "", "")
	if err != nil || tok != "" {
		t.Errorf("expected empty token, got %q err %v", tok, err)
	}
}
