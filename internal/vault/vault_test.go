package vault

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dantte-lp/ypcli/internal/api"
)

func kvServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != token {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path != "/v1/secret/data/db" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"data": map[string]any{"password": "hunter2", "port": 5432}},
		})
	}))
}

func TestReadFieldString(t *testing.T) {
	srv := kvServer(t, "vt")
	defer srv.Close()
	c := Client{Addr: srv.URL, Token: "vt", HTTP: srv.Client()}
	got, err := c.ReadField(context.Background(), "secret", "db", "password")
	if err != nil {
		t.Fatalf("ReadField: %v", err)
	}
	if got != "hunter2" {
		t.Errorf("value = %q, want hunter2", got)
	}
}

func TestReadFieldNonString(t *testing.T) {
	srv := kvServer(t, "vt")
	defer srv.Close()
	c := Client{Addr: srv.URL, Token: "vt", HTTP: srv.Client()}
	got, err := c.ReadField(context.Background(), "", "db", "port") // mount defaults to secret
	if err != nil || got != "5432" {
		t.Errorf("value = %q err = %v, want 5432", got, err)
	}
}

func TestReadFieldMissingField(t *testing.T) {
	srv := kvServer(t, "vt")
	defer srv.Close()
	c := Client{Addr: srv.URL, Token: "vt", HTTP: srv.Client()}
	if _, err := c.ReadField(context.Background(), "secret", "db", "nope"); err == nil {
		t.Error("expected error for missing field")
	}
}

func TestReadFieldForbidden(t *testing.T) {
	srv := kvServer(t, "vt")
	defer srv.Close()
	c := Client{Addr: srv.URL, Token: "wrong", HTTP: srv.Client()}
	_, err := c.ReadField(context.Background(), "secret", "db", "password")
	if !errors.Is(err, api.ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestReadFieldNotFound(t *testing.T) {
	srv := kvServer(t, "vt")
	defer srv.Close()
	c := Client{Addr: srv.URL, Token: "vt", HTTP: srv.Client()}
	_, err := c.ReadField(context.Background(), "secret", "missing", "password")
	if !errors.Is(err, api.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestReadFieldNotConfigured(t *testing.T) {
	if _, err := (Client{}).ReadField(context.Background(), "secret", "db", "x"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}
