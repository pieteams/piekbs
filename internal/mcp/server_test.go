//go:build fts5

package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pieteams/piekbs/internal/kb"
)

func TestWithProtocolVersion(t *testing.T) {
	// 创建测试处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 包装中间件
	wrappedHandler := withProtocolVersion(handler)

	// 测试1: 不发送版本 header（应该通过）
	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 测试2: 发送正确的版本（应该通过）
	req = httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 测试3: 发送错误的版本（应该返回 400）
	req = httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("MCP-Protocol-Version", "2024-11-05")
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWithAuth(t *testing.T) {
	// 创建测试处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 包装中间件
	wrappedHandler := withAuth("test-key", handler)

	// 测试1: 不发送任何 header（应该返回 401）
	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	// 测试2: 发送正确的 Authorization: Bearer（应该通过）
	req = httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 测试3: 发送正确的 x-api-key（应该通过，兼容性）
	req = httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("x-api-key", "test-key")
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 测试4: 发送错误的 token（应该返回 401）
	req = httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMCPIntegration(t *testing.T) {
	// 创建临时目录
	dir := t.TempDir()
	t.Cleanup(kb.CloseGlobalDB)

	// 初始化数据库
	if _, err := kb.GlobalDB(dir); err != nil {
		t.Fatalf("GlobalDB failed: %v", err)
	}

	// 创建测试服务器
	mux := http.NewServeMux()
	RegisterRoutes(mux, dir)

	// 有效的 JSON-RPC 请求体
	validJSONRPC := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`

	// 测试1: 不发送任何 header（应该通过）
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(validJSONRPC))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 测试2: 发送正确的协议版本（应该通过）
	req = httptest.NewRequest("POST", "/mcp", strings.NewReader(validJSONRPC))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 测试3: 发送错误的协议版本（应该返回 400）
	req = httptest.NewRequest("POST", "/mcp", strings.NewReader(validJSONRPC))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Protocol-Version", "2024-11-05")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
