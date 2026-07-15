package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newClient(srv *httptest.Server, opts ...Option) *Client {
	opts = append(opts, WithHTTPClient(srv.Client()))
	return New(srv.URL, opts...)
}

func TestBearerHeaderSet(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(envelope{Message: "id123"})
	}))
	defer srv.Close()

	c := newClient(srv, WithToken("s3cr3t"))
	if _, err := c.FetchSecret(context.Background(), "x"); err != nil {
		t.Fatalf("FetchSecret: %v", err)
	}
	if gotAuth != "Bearer s3cr3t" {
		t.Errorf("Authorization = %q, want Bearer s3cr3t", gotAuth)
	}
}

func TestNoTokenNoHeader(t *testing.T) {
	var hadAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadAuth = r.Header["Authorization"]
		json.NewEncoder(w).Encode(envelope{Message: "ok"})
	}))
	defer srv.Close()

	if _, err := newClient(srv).FetchSecret(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if hadAuth {
		t.Error("Authorization header should be absent without a token")
	}
}

func TestStatusClassification(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrUnauthorized},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusGone, ErrNotFound},
		{http.StatusInternalServerError, ErrServer},
		{http.StatusBadRequest, ErrServer},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			json.NewEncoder(w).Encode(envelope{Message: "boom"})
		}))
		_, err := newClient(srv).FetchSecret(context.Background(), "x")
		srv.Close()
		if !errors.Is(err, c.want) {
			t.Errorf("status %d: err = %v, want %v", c.status, err, c.want)
		}
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Message != "boom" {
			t.Errorf("status %d: want *Error with server message, got %v", c.status, err)
		}
	}
}

func TestContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(envelope{Message: "late"})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := newClient(srv).FetchSecret(ctx, "x")
	if !errors.Is(err, ErrNetwork) {
		t.Errorf("err = %v, want ErrNetwork on timeout", err)
	}
}

func TestCreateSecret(t *testing.T) {
	var body Secret
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/create/secret" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(envelope{Message: "newid"})
	}))
	defer srv.Close()

	id, err := newClient(srv).CreateSecret(context.Background(), Secret{
		Message: "cipher", Expiration: 3600, OneTime: true, RequireAuth: true,
	})
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if id != "newid" {
		t.Errorf("id = %q, want newid", id)
	}
	if body.Message != "cipher" || body.Expiration != 3600 || !body.OneTime || !body.RequireAuth {
		t.Errorf("server received %+v", body)
	}
}

func TestFetchSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/secret/abc" {
			t.Errorf("path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(envelope{Message: "-----BEGIN PGP MESSAGE-----"})
	}))
	defer srv.Close()

	msg, err := newClient(srv).FetchSecret(context.Background(), "abc")
	if err != nil || !strings.HasPrefix(msg, "-----BEGIN PGP") {
		t.Errorf("msg = %q err = %v", msg, err)
	}
}

func TestCreateFile(t *testing.T) {
	var gotExp, gotOne, gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotExp = r.Header.Get("X-Yopass-Expiration")
		gotOne = r.Header.Get("X-Yopass-OneTime")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		json.NewEncoder(w).Encode(envelope{Message: "fileid"})
	}))
	defer srv.Close()

	id, err := newClient(srv).CreateFile(context.Background(), strings.NewReader("BINARY"), 86400, true)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if id != "fileid" {
		t.Errorf("id = %q", id)
	}
	if gotExp != "86400" || gotOne != "true" || gotCT != "application/octet-stream" {
		t.Errorf("headers: exp=%q one=%q ct=%q", gotExp, gotOne, gotCT)
	}
	if string(gotBody) != "BINARY" {
		t.Errorf("body = %q", gotBody)
	}
}

func TestFetchFileBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/file/xyz" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte("encrypted-bytes"))
	}))
	defer srv.Close()

	data, err := newClient(srv).FetchFileBytes(context.Background(), "xyz")
	if err != nil || string(data) != "encrypted-bytes" {
		t.Errorf("data = %q err = %v", data, err)
	}
}

func TestConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"ARGON2":true,"READ_ONLY":false,"FORCE_ONETIME_SECRETS":true,"DEFAULT_EXPIRY":3600}`))
	}))
	defer srv.Close()

	cfg, err := newClient(srv).Config(context.Background())
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if !cfg.Argon2 || !cfg.ForceOneTimeSecrets || cfg.DefaultExpiry != 3600 {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"version":"13.1.0"}`))
	}))
	defer srv.Close()
	v, err := newClient(srv).Version(context.Background())
	if err != nil || v != "13.1.0" {
		t.Errorf("version = %q err = %v", v, err)
	}
}

func TestVersionUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err := newClient(srv).Version(context.Background())
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("err = %v, want ErrUnsupported", err)
	}
}
