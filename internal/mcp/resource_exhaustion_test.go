package mcp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/mcp"
)

type rpcTestResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func TestMCP_NormalRequest(t *testing.T) {
	in := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\",\"params\":{}}\n")
	var out bytes.Buffer
	srv := mcp.NewServer(in, &out, "test")

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp rpcTestResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v, raw: %s", err, out.String())
	}
	if resp.Error != nil {
		t.Fatalf("expected no error, got: %+v", resp.Error)
	}
}

func TestMCP_MalformedJSON(t *testing.T) {
	in := strings.NewReader("{\"jsonrpc\":\"2.0\", malformed json payload\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"ping\",\"params\":{}}\n")
	var out bytes.Buffer
	srv := mcp.NewServer(in, &out, "test")

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses (1 parse error, 1 ping response), got %d: %v", len(lines), lines)
	}

	var errResp rpcTestResponse
	if err := json.Unmarshal([]byte(lines[0]), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error == nil || errResp.Error.Code != -32700 {
		t.Fatalf("expected JSON-RPC parse error (-32700), got: %+v", errResp.Error)
	}

	var pingResp rpcTestResponse
	if err := json.Unmarshal([]byte(lines[1]), &pingResp); err != nil {
		t.Fatalf("unmarshal ping response: %v", err)
	}
	if pingResp.Error != nil {
		t.Fatalf("expected successful ping, got error: %+v", pingResp.Error)
	}
}

func TestMCP_RequestNearMaximumSize(t *testing.T) {
	// Configure server with a test-sized limit of 8192 bytes
	limit := 8192
	var in bytes.Buffer

	// Padding to create request of 8000 bytes (near max size)
	padding := strings.Repeat("x", 7800)
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":10,"method":"unknown_test","params":{"pad":"%s"}}`+"\n", padding)
	if len(req) >= limit {
		t.Fatalf("test setup error: request length %d must be < limit %d", len(req), limit)
	}
	in.WriteString(req)

	var out bytes.Buffer
	srv := mcp.NewServer(&in, &out, "test")
	srv.MaxMessageSize = limit

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp rpcTestResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v, raw: %s", err, out.String())
	}
	// It should reach dispatch and return method not found (-32601), NOT -32600 (too large)
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected method not found (-32601) for valid size request, got: %+v", resp.Error)
	}
}

func TestMCP_RequestOverMaximumSize(t *testing.T) {
	limit := 1024
	var in bytes.Buffer

	// Oversized request (2000 bytes > 1024)
	padding := strings.Repeat("a", 2000)
	in.WriteString(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"%s"}}`+"\n", padding))
	// Followed by a valid normal request to verify server is not killed/poisoned
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"ping\",\"params\":{}}\n")

	var out bytes.Buffer
	srv := mcp.NewServer(&in, &out, "test")
	srv.MaxMessageSize = limit

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses (1 oversized error, 1 ping ok), got %d: %v", len(lines), lines)
	}

	var overResp rpcTestResponse
	if err := json.Unmarshal([]byte(lines[0]), &overResp); err != nil {
		t.Fatalf("unmarshal oversized response: %v", err)
	}
	if overResp.Error == nil || overResp.Error.Code != -32600 {
		t.Fatalf("expected error code -32600 (request entity too large), got: %+v", overResp.Error)
	}

	var pingResp rpcTestResponse
	if err := json.Unmarshal([]byte(lines[1]), &pingResp); err != nil {
		t.Fatalf("unmarshal second response: %v", err)
	}
	if pingResp.Error != nil {
		t.Fatalf("subsequent request failed after oversized message: %+v", pingResp.Error)
	}
}

func TestMCP_RepeatedOversizedRequests(t *testing.T) {
	limit := 512
	var in bytes.Buffer

	// Send 3 repeated oversized requests, followed by a valid ping
	for i := 0; i < 3; i++ {
		padding := strings.Repeat("z", 1000)
		in.WriteString(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"test","params":{"data":"%s"}}`+"\n", i+1, padding))
	}
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":99,\"method\":\"ping\",\"params\":{}}\n")

	var out bytes.Buffer
	srv := mcp.NewServer(&in, &out, "test")
	srv.MaxMessageSize = limit

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 responses, got %d: %v", len(lines), lines)
	}

	for i := 0; i < 3; i++ {
		var resp rpcTestResponse
		if err := json.Unmarshal([]byte(lines[i]), &resp); err != nil {
			t.Fatalf("unmarshal response %d: %v", i, err)
		}
		if resp.Error == nil || resp.Error.Code != -32600 {
			t.Fatalf("expected error -32600 for request %d, got: %+v", i, resp.Error)
		}
	}

	var finalResp rpcTestResponse
	if err := json.Unmarshal([]byte(lines[3]), &finalResp); err != nil {
		t.Fatalf("unmarshal final response: %v", err)
	}
	if finalResp.Error != nil {
		t.Fatalf("final ping failed after repeated oversized requests: %+v", finalResp.Error)
	}
}
