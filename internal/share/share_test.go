package share

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/dantte-lp/ypcli/internal/api"
)

// fakeYopass is a minimal in-memory yopass API for share tests.
type fakeYopass struct {
	mu      sync.Mutex
	secrets map[string]string
	files   map[string][]byte
	argon2  bool
	n       int
}

func newFake(argon2 bool) *fakeYopass {
	return &fakeYopass{secrets: map[string]string{}, files: map[string][]byte{}, argon2: argon2}
}

func (f *fakeYopass) id() string { f.n++; return "id" + string(rune('a'+f.n)) }

func (f *fakeYopass) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ARGON2": f.argon2})
	})
	mux.HandleFunc("/create/secret", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Message string `json:"message"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		f.mu.Lock()
		id := f.id()
		f.secrets[id] = b.Message
		f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"message": id})
	})
	mux.HandleFunc("/create/file", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		id := f.id()
		f.files[id] = data
		f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"message": id})
	})
	mux.HandleFunc("/secret/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/secret/")
		f.mu.Lock()
		msg, ok := f.secrets[id]
		delete(f.secrets, id)
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "gone"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": msg})
	})
	mux.HandleFunc("/file/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/file/")
		f.mu.Lock()
		data, ok := f.files[id]
		delete(f.files, id)
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(data)
	})
	return httptest.NewServer(mux)
}

func TestSendTextReceiveRoundTrip(t *testing.T) {
	fake := newFake(true) // argon2 advertised → auto-detected
	srv := fake.server(t)
	defer srv.Close()
	client := api.New(srv.URL, api.WithHTTPClient(srv.Client()))
	ctx := context.Background()

	res, err := SendText(ctx, client, srv.URL, strings.NewReader("top secret"),
		Options{Expiration: 3600, OneTime: true})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if res.File || res.ManualKey || res.Key == "" || res.Expiration != "1h" || !res.OneTime {
		t.Errorf("unexpected result %+v", res)
	}
	if !strings.Contains(res.URL, "/#/s/") || !strings.Contains(res.URL, res.Key) {
		t.Errorf("url = %q", res.URL)
	}

	got, err := Receive(ctx, client, Target{ID: res.ID, Key: res.Key})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if got.Content != "top secret" || got.File {
		t.Errorf("received %+v", got)
	}
}

func TestSendFileReceiveRoundTrip(t *testing.T) {
	fake := newFake(false)
	srv := fake.server(t)
	defer srv.Close()
	client := api.New(srv.URL, api.WithHTTPClient(srv.Client()))
	ctx := context.Background()

	dir := t.TempDir()
	path := dir + "/creds.env"
	if err := os.WriteFile(path, []byte("USER=admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := SendFile(ctx, client, srv.URL, path, Options{Expiration: 86400})
	if err != nil {
		t.Fatalf("SendFile: %v", err)
	}
	if !res.File || !strings.Contains(res.URL, "/#/f/") || res.Expiration != "1d" {
		t.Errorf("unexpected result %+v", res)
	}

	var wrapped bool
	got, err := Receive(ctx, client, Target{
		ID: res.ID, Key: res.Key, File: true,
		Wrap: func(r io.Reader, _ int64) io.Reader { wrapped = true; return r },
	})
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if got.Content != "USER=admin\n" || got.Filename != "creds.env" || !got.File {
		t.Errorf("received %+v", got)
	}
	if !wrapped {
		t.Error("Wrap should be invoked for file downloads")
	}
}

func TestSendManualKeyAndArgon2Override(t *testing.T) {
	fake := newFake(false)
	srv := fake.server(t)
	defer srv.Close()
	client := api.New(srv.URL, api.WithHTTPClient(srv.Client()))
	ctx := context.Background()

	forceArgon2 := true
	res, err := SendText(ctx, client, srv.URL, strings.NewReader("x"),
		Options{Key: "manual-key-1234567890", Expiration: 3600, Argon2: &forceArgon2})
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if !res.ManualKey || strings.Contains(res.URL, res.Key) {
		t.Errorf("manual key should be omitted from URL: %+v", res)
	}
	got, err := Receive(ctx, client, Target{ID: res.ID, Key: "manual-key-1234567890"})
	if err != nil || got.Content != "x" {
		t.Errorf("received %q err %v", got.Content, err)
	}
}

func TestReceiveConsumedIsNotFound(t *testing.T) {
	fake := newFake(false)
	srv := fake.server(t)
	defer srv.Close()
	client := api.New(srv.URL, api.WithHTTPClient(srv.Client()))
	ctx := context.Background()

	res, _ := SendText(ctx, client, srv.URL, strings.NewReader("once"), Options{Expiration: 3600, OneTime: true})
	if _, err := Receive(ctx, client, Target{ID: res.ID, Key: res.Key}); err != nil {
		t.Fatalf("first receive: %v", err)
	}
	if _, err := Receive(ctx, client, Target{ID: res.ID, Key: res.Key}); err == nil {
		t.Error("second receive should fail (consumed)")
	}
}
