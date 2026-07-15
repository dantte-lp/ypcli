package clipboard

import (
	"errors"
	"testing"
)

func stub(t *testing.T, unsup bool, w func(string) error) {
	t.Helper()
	prevU, prevW := unsupported, writeAll
	unsupported, writeAll = unsup, w
	t.Cleanup(func() { unsupported, writeAll = prevU, prevW })
}

func TestCopySuccess(t *testing.T) {
	var got string
	stub(t, false, func(s string) error { got = s; return nil })

	if !Available() {
		t.Error("Available should be true when a clipboard exists")
	}
	if err := Copy("hello"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if got != "hello" {
		t.Errorf("clipboard got %q, want hello", got)
	}
}

func TestCopyWriteError(t *testing.T) {
	stub(t, false, func(string) error { return errors.New("boom") })
	if err := Copy("x"); err == nil {
		t.Error("expected error when the clipboard write fails")
	}
}

func TestCopyUnsupported(t *testing.T) {
	stub(t, true, func(string) error { t.Fatal("write should not run when unsupported"); return nil })
	if Available() {
		t.Error("Available should be false when unsupported")
	}
	if err := Copy("x"); !errors.Is(err, errUnavailable) {
		t.Errorf("err = %v, want errUnavailable", err)
	}
}
