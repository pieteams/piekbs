package mcp

import (
	"log"
	"net/http"
	"strings"
)

// withProtocolVersion is a no-op — MCP protocol version negotiation happens
// via the JSON-RPC initialize request/response, not HTTP headers.
// The mcp-go SDK handles this correctly in the server.
func withProtocolVersion(h http.Handler) http.Handler {
	return h
}

// withAPIKey rejects requests missing a valid x-api-key header.
// Deprecated: Use withAuth instead. Kept for backward compatibility.
func withAPIKey(key string, h http.Handler) http.Handler {
	if key == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != key {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// withAuth implements unified authentication middleware.
// Priority: Authorization: Bearer header (HTTP/MCP standard)
// Fallback: x-api-key header (deprecated, for backward compatibility)
// If key is empty, all requests are allowed (auth disabled).
func withAuth(key string, h http.Handler) http.Handler {
	if key == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization: Bearer header (preferred)
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == key {
					h.ServeHTTP(w, r)
					return
				}
			}
		}

		// Fallback: x-api-key header (deprecated)
		if apiKey := r.Header.Get("x-api-key"); apiKey != "" {
			log.Printf("WARNING: x-api-key header is deprecated. Use Authorization: Bearer instead.")
			if apiKey == key {
				h.ServeHTTP(w, r)
				return
			}
		}

		// Authentication failed
		w.Header().Set("WWW-Authenticate", `Bearer realm="MCP Server"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// withCORS adds permissive CORS headers for local MCP clients.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id, x-api-key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
