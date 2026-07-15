package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeYopass is a minimal in-memory yopass API for MCP server tests.
type fakeYopass struct {
	mu      sync.Mutex
	secrets map[string]string
	files   map[string][]byte
	n       int
}

func fakeServer(t *testing.T) (*httptest.Server, *fakeYopass) {
	t.Helper()
	f := &fakeYopass{secrets: map[string]string{}, files: map[string][]byte{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ARGON2": true})
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": "fake-13.0"})
	})
	mux.HandleFunc("/create/secret", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Message string `json:"message"`
		}
		json.NewDecoder(r.Body).Decode(&b)
		f.mu.Lock()
		f.n++
		id := "s" + string(rune('a'+f.n))
		f.secrets[id] = b.Message
		f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"message": id})
	})
	mux.HandleFunc("/create/file", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.n++
		id := "f" + string(rune('a'+f.n))
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
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, f
}

// connect wires an in-memory client to a fresh server pointed at cfgPath.
func connect(t *testing.T, opts Options) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := New(opts)
	ct, st := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any, dst any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %+v", name, res.Content)
	}
	if dst != nil {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, dst); err != nil {
			t.Fatalf("decode %s output: %v", name, err)
		}
	}
	return res
}

func writeConfig(t *testing.T, apiURL string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "defaults:\n  api: " + apiURL + "\n  url: " + apiURL + "\n" +
		"active: work\nprofiles:\n  work: {}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSendReceiveRoundTrip(t *testing.T) {
	srv, _ := fakeServer(t)
	cfg := writeConfig(t, srv.URL)
	cs := connect(t, Options{ConfigPath: cfg, Version: "test-1.0"})

	var sent sendOutput
	call(t, cs, "send_secret", map[string]any{"text": "hello mcp", "expiration": "1d"}, &sent)
	if !strings.Contains(sent.URL, "/#/s/") || sent.Expiration != "1d" || !sent.OneTime {
		t.Fatalf("send output = %+v", sent)
	}
	if !strings.HasPrefix(sent.URL, srv.URL) {
		t.Errorf("url not on configured server: %s", sent.URL)
	}

	var got receiveOutput
	call(t, cs, "receive_secret", map[string]any{"url": sent.URL}, &got)
	if got.Content != "hello mcp" || got.File {
		t.Errorf("receive output = %+v", got)
	}
}

func TestSendFileRoundTrip(t *testing.T) {
	srv, _ := fakeServer(t)
	cfg := writeConfig(t, srv.URL)
	cs := connect(t, Options{ConfigPath: cfg, Version: "test"})

	file := filepath.Join(t.TempDir(), "creds.env")
	if err := os.WriteFile(file, []byte("USER=admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var sent sendOutput
	call(t, cs, "send_file", map[string]any{"path": file}, &sent)
	if !sent.File || !strings.Contains(sent.URL, "/#/f/") {
		t.Fatalf("send_file output = %+v", sent)
	}
	var got receiveOutput
	call(t, cs, "receive_secret", map[string]any{"url": sent.URL}, &got)
	if got.Content != "USER=admin\n" || got.Filename != "creds.env" || !got.File {
		t.Errorf("receive output = %+v", got)
	}
}

func TestListProfilesAndVersion(t *testing.T) {
	srv, _ := fakeServer(t)
	cfg := writeConfig(t, srv.URL)
	cs := connect(t, Options{ConfigPath: cfg, Version: "test-1.0"})

	var list listProfilesOutput
	call(t, cs, "list_profiles", map[string]any{}, &list)
	if len(list.Profiles) != 1 || list.Profiles[0].Name != "work" || !list.Profiles[0].Active {
		t.Errorf("profiles = %+v", list.Profiles)
	}

	var ver versionOutput
	call(t, cs, "server_version", map[string]any{}, &ver)
	if ver.Client != "test-1.0" || ver.Server != "fake-13.0" {
		t.Errorf("version = %+v", ver)
	}
}

func callErr(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s transport error: %v", name, err)
	}
	if !res.IsError {
		t.Fatalf("call %s: expected a tool error, got %+v", name, res.StructuredContent)
	}
}

func TestToolErrorPaths(t *testing.T) {
	srv, _ := fakeServer(t)
	cfg := writeConfig(t, srv.URL)
	cs := connect(t, Options{ConfigPath: cfg, Version: "test"})

	callErr(t, cs, "send_secret", map[string]any{"text": "x", "expiration": "2w"}) // bad expiration
	callErr(t, cs, "send_secret", map[string]any{"text": "x", "profile": "ghost"}) // unknown profile
	callErr(t, cs, "send_file", map[string]any{"path": "relative/path"})           // not absolute
	callErr(t, cs, "receive_secret", map[string]any{"id": "abc"})                  // missing key
	callErr(t, cs, "receive_secret", map[string]any{})                             // no url/id
}

func TestBinaryReceiveIsBase64(t *testing.T) {
	srv, _ := fakeServer(t)
	cfg := writeConfig(t, srv.URL)
	cs := connect(t, Options{ConfigPath: cfg, Version: "test"})

	bin := []byte{0x00, 0x01, 0xff, 0xfe, 0x80}
	file := filepath.Join(t.TempDir(), "blob.bin")
	if err := os.WriteFile(file, bin, 0o600); err != nil {
		t.Fatal(err)
	}
	var sent sendOutput
	call(t, cs, "send_file", map[string]any{"path": file}, &sent)

	var got receiveOutput
	call(t, cs, "receive_secret", map[string]any{"url": sent.URL}, &got)
	if got.Content != "" || got.ContentBase64 == "" {
		t.Fatalf("binary payload should use content_base64: %+v", got)
	}
	decoded, err := base64.StdEncoding.DecodeString(got.ContentBase64)
	if err != nil || string(decoded) != string(bin) {
		t.Errorf("base64 round-trip mismatch: %v", err)
	}
}

func TestProfileArgumentHonored(t *testing.T) {
	srvA, _ := fakeServer(t)
	srvB, _ := fakeServer(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	body := "active: a\nprofiles:\n" +
		"  a:\n    api: " + srvA.URL + "\n    url: " + srvA.URL + "\n" +
		"  b:\n    api: " + srvB.URL + "\n    url: " + srvB.URL + "\n"
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, Options{ConfigPath: cfg, Version: "test"})

	var sent sendOutput
	call(t, cs, "send_secret", map[string]any{"text": "hi", "profile": "b"}, &sent)
	if !strings.HasPrefix(sent.URL, srvB.URL) {
		t.Errorf("profile b should target %s, got %s", srvB.URL, sent.URL)
	}
}

func TestReadOnlyOmitsReceive(t *testing.T) {
	srv, _ := fakeServer(t)
	cfg := writeConfig(t, srv.URL)
	cs := connect(t, Options{ConfigPath: cfg, ReadOnly: true, Version: "test"})

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "receive_secret" {
			t.Error("receive_secret must be absent in read-only mode")
		}
	}
}
