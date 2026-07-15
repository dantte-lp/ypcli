package mcpserver

import (
	"crypto/subtle"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler serves the MCP server over Streamable HTTP. When token is non-empty
// (required for any network-exposed deployment) every request must carry
// "Authorization: Bearer <token>".
func Handler(srv *mcp.Server, token string) http.Handler {
	base := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	if token == "" {
		return base
	}
	return bearerAuth(token, base)
}

func bearerAuth(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
