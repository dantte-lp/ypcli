package output

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestProgressReaderPassesBytesAndReports(t *testing.T) {
	const data = "hello world payload"
	var bar bytes.Buffer
	r := NewProgressReader(strings.NewReader(data), int64(len(data)), &bar, "downloading")

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != data {
		t.Errorf("data = %q, want %q", got, data)
	}
	out := bar.String()
	if !strings.Contains(out, "downloading") || !strings.Contains(out, "100%") {
		t.Errorf("progress output missing label or completion: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("progress should end with a newline on EOF: %q", out)
	}
}

func TestProgressReaderSilentWhenTotalUnknown(t *testing.T) {
	const data = "unknown length"
	var bar bytes.Buffer
	r := NewProgressReader(strings.NewReader(data), -1, &bar, "dl")

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != data {
		t.Errorf("data = %q", got)
	}
	if bar.Len() != 0 {
		t.Errorf("no progress should be printed when total is unknown, got %q", bar.String())
	}
}
