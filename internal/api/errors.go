package api

import (
	"errors"
	"fmt"
)

// Sentinel error kinds. Callers classify failures with errors.Is; the cli
// layer maps each kind to a process exit code.
var (
	// ErrNetwork indicates a transport-level failure (connection, timeout).
	ErrNetwork = errors.New("network error")
	// ErrUnauthorized indicates a 401/403 response (auth required or denied).
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNotFound indicates a 404/410 response (missing or already consumed).
	ErrNotFound = errors.New("not found")
	// ErrServer indicates a 5xx or otherwise unexpected server response.
	ErrServer = errors.New("server error")
	// ErrUnsupported indicates the server lacks the requested endpoint.
	ErrUnsupported = errors.New("unsupported by server")
)

// Error is a server-side failure carrying the HTTP status and the server's
// message. It unwraps to one of the sentinel kinds above.
type Error struct {
	Status  int
	Message string
	kind    error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s (status %d): %s", e.kind, e.Status, e.Message)
	}
	return fmt.Sprintf("%s (status %d)", e.kind, e.Status)
}

// Unwrap exposes the sentinel kind for errors.Is.
func (e *Error) Unwrap() error { return e.kind }

// classify maps an HTTP status code to its sentinel kind.
func classify(status int) error {
	switch {
	case status == 401 || status == 403:
		return ErrUnauthorized
	case status == 404 || status == 410:
		return ErrNotFound
	default:
		return ErrServer
	}
}
