package mcp

import (
	"log"
	"net/http"
	"strings"
)

// MCP 协议版本（使用日期格式，不是语义化版本）
const mcpProtocolVersion = "2025-06-18"

// withProtocolVersion 检查 MCP-Protocol-Version header
// 如果客户端发送了不支持的版本，返回 400 Bad Request
func withProtocolVersion(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查协议版本（可选，用于调试）
		if v := r.Header.Get("MCP-Protocol-Version"); v != "" {
			if v != mcpProtocolVersion {
				http.Error(w, "unsupported protocol version", http.StatusBadRequest)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
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
