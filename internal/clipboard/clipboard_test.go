package clipboard

import "testing"

// Copy must never panic and must return a clear error when no clipboard is
// available (e.g. headless CI). When a clipboard exists, it should round-trip.
func TestCopyGraceful(t *testing.T) {
	err := Copy("ypcli-clipboard-test")
	if Available() {
		if err != nil {
			t.Errorf("clipboard available but Copy failed: %v", err)
		}
		return
	}
	if err == nil {
		t.Error("clipboard unavailable but Copy returned nil error")
	}
}
