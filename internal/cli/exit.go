package cli

import (
	"errors"
	"fmt"

	"github.com/dantte-lp/ypcli/internal/api"
	"github.com/dantte-lp/ypcli/internal/crypto"
)

// Sentinel errors that steer the exit code without a network round-trip.
var (
	errUsage  = errors.New("usage error")
	errConfig = errors.New("configuration error")
)

// exitCode maps an error to a stable process exit code. The scheme lets CI
// distinguish auth failures from missing secrets from decryption failures.
func exitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errUsage):
		return 2
	case errors.Is(err, errConfig):
		return 3
	case errors.Is(err, api.ErrNetwork):
		return 4
	case errors.Is(err, api.ErrUnauthorized):
		return 5
	case errors.Is(err, api.ErrNotFound):
		return 6
	case errors.Is(err, crypto.ErrInvalidKey),
		errors.Is(err, crypto.ErrInvalidMessage),
		errors.Is(err, crypto.ErrEmptyKey):
		return 7
	default:
		return 1
	}
}

// isUsageError reports whether the error should trigger usage output.
func isUsageError(err error) bool {
	return errors.Is(err, errUsage)
}

// usage wraps msg as a usage error (exit code 2).
func usage(format string, args ...any) error {
	return errors.Join(errUsage, fmt.Errorf(format, args...))
}
