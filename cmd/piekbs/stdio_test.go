//go:build fts5

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const stdioTestKB = "PIEKBS_TEST_STDIO_KB"

func TestStdioProcess(t *testing.T) {
	if root := os.Getenv(stdioTestKB); root != "" {
		if err := runStdio(root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func TestStdioKeepsConversionLogsOffProtocolStream(t *testing.T) {
	root := t.TempDir()
	if err := ensureKBDirs(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "raw", "items.csv")
	if err := os.WriteFile(path, []byte("name,value\nStdioConversionMarker,42\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Trigger startup conversion without depending on file watcher registration.
	modified := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	writeTimestamp(filepath.Join(root, lastShutdownFile))
	t.Setenv(stdioTestKB, root)
	t.Setenv("PIEKBS_DISTILL_BASE_URL", "")
	t.Setenv("PIEKBS_API_KEY", "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStdioProcess$")
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	}()

	input := json.NewEncoder(stdin)
	output := bufio.NewScanner(stdout)
	requestID := 0
	request := func(method string, params map[string]any) json.RawMessage {
		t.Helper()
		requestID++
		message := map[string]any{"jsonrpc": "2.0", "id": requestID, "method": method, "params": params}
		if err := input.Encode(message); err != nil {
			t.Fatal(err)
		}
		if !output.Scan() {
			t.Fatalf("no reply for %s: %v", method, output.Err())
		}
		var reply struct {
			JSONRPC string
			ID      int
			Result  json.RawMessage
			Error   json.RawMessage
		}
		if err := json.Unmarshal(output.Bytes(), &reply); err != nil {
			t.Fatalf("non-JSON line on MCP stdout: %q", output.Text())
		}
		if reply.JSONRPC != "2.0" || reply.ID != requestID || len(reply.Error) > 0 {
			t.Fatalf("unexpected reply to %s: %s", method, output.Bytes())
		}
		return reply.Result
	}
	request("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]string{"name": "piekbs-stdio-test", "version": "1"},
	})
	if err := input.Encode(map[string]string{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		t.Fatal(err)
	}

	for ctx.Err() == nil {
		result := request("tools/call", map[string]any{
			"name":      "kb_search",
			"arguments": map[string]string{"query": "StdioConversionMarker"},
		})
		var call struct {
			IsError bool
			Content []struct{ Text string }
		}
		if err := json.Unmarshal(result, &call); err != nil || call.IsError || len(call.Content) != 1 {
			t.Fatalf("unexpected search response: %s (%v)", result, err)
		}
		var search struct {
			Results []struct{ ID string }
		}
		if err := json.Unmarshal([]byte(call.Content[0].Text), &search); err != nil {
			t.Fatal(err)
		}
		for _, hit := range search.Results {
			if hit.ID == "raw/converted/items.md" {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("converted document did not become searchable")
}
