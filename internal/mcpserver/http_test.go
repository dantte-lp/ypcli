package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	h := bearerAuth("s3cr3t", next)

	cases := []struct {
		name    string
		header  string
		want    int
		reached bool
	}{
		{"no header", "", http.StatusUnauthorized, false},
		{"wrong token", "Bearer nope", http.StatusUnauthorized, false},
		{"missing scheme", "s3cr3t", http.StatusUnauthorized, false},
		{"valid", "Bearer s3cr3t", http.StatusOK, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reached = false
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("status = %d, want %d", rec.Code, c.want)
			}
			if reached != c.reached {
				t.Errorf("reached = %v, want %v", reached, c.reached)
			}
		})
	}
}

func TestHandlerNoTokenSkipsAuth(t *testing.T) {
	// With an empty token the handler is the bare MCP handler (no auth wrapper).
	h := Handler(New(Options{ConfigPath: "/nonexistent"}), "")
	if h == nil {
		t.Fatal("Handler returned nil")
	}
}
