package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTextSend(t *testing.T) {
	var out, errw bytes.Buffer
	p := New(false, &out, &errw)
	err := p.Send(SendResult{URL: "https://y/#/s/abc/k1", Key: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "https://y/#/s/abc/k1" {
		t.Errorf("stdout = %q", got)
	}
	if errw.Len() != 0 {
		t.Errorf("stderr should be empty, got %q", errw.String())
	}
}

func TestTextSendManualKey(t *testing.T) {
	var out, errw bytes.Buffer
	p := New(false, &out, &errw)
	if err := p.Send(SendResult{URL: "https://y/#/c/abc", Key: "secretkey", ManualKey: true}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "https://y/#/c/abc" {
		t.Errorf("stdout = %q", out.String())
	}
	if !strings.Contains(errw.String(), "secretkey") {
		t.Errorf("stderr should carry the key, got %q", errw.String())
	}
}

func TestJSONSend(t *testing.T) {
	var out, errw bytes.Buffer
	p := New(true, &out, &errw)
	if err := p.Send(SendResult{
		ID: "abc", URL: "https://y/#/s/abc/k1", Key: "k1",
		OneTime: true, Expiration: "1h",
	}); err != nil {
		t.Fatal(err)
	}
	var got SendResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid json: %v", err)
	}
	if got.ID != "abc" || !got.OneTime || got.Expiration != "1h" {
		t.Errorf("decoded = %+v", got)
	}
}

func TestTextReceiveContentToStdout(t *testing.T) {
	var out, errw bytes.Buffer
	p := New(false, &out, &errw)
	if err := p.Receive(ReceiveResult{Content: "plaintext"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "plaintext" {
		t.Errorf("stdout = %q, want exact plaintext (no newline)", out.String())
	}
}

func TestTextReceiveFileToStderr(t *testing.T) {
	var out, errw bytes.Buffer
	p := New(false, &out, &errw)
	if err := p.Receive(ReceiveResult{Written: "/tmp/x", Filename: "x", Bytes: 42}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for file receive, got %q", out.String())
	}
	if !strings.Contains(errw.String(), "/tmp/x") || !strings.Contains(errw.String(), "42") {
		t.Errorf("stderr = %q", errw.String())
	}
}

func TestJSONError(t *testing.T) {
	var out, errw bytes.Buffer
	p := New(true, &out, &errw)
	if err := p.Error(5, "unauthorized"); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("json error should go to stderr, stdout = %q", out.String())
	}
	var env struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(errw.Bytes(), &env); err != nil {
		t.Fatalf("stderr not json: %v", err)
	}
	if env.Error.Code != 5 || env.Error.Message != "unauthorized" {
		t.Errorf("decoded = %+v", env.Error)
	}
}

func TestQR(t *testing.T) {
	s, err := QR("https://yopass.se/#/s/abc/k1")
	if err != nil {
		t.Fatal(err)
	}
	if len(s) == 0 || !strings.ContainsAny(s, "█▀▄ ") {
		t.Errorf("qr output does not look like block art: %q", s[:min(len(s), 20)])
	}
}
