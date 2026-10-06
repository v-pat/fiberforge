package mcp_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/mcp"
)

func TestMCP_TargetDirTraversalRejection(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("FIBERFORGE_WORKSPACE_ROOT", wsRoot)

	// Create an escape directory outside wsRoot
	outsideDir := t.TempDir()

	traversalPayloads := []string{
		"../../escape",
		"../../../tmp/escape",
		"/absolute/escape",
		filepath.Join(outsideDir, "malicious"),
		"sub/../../../../escape",
	}

	for _, payload := range traversalPayloads {
		t.Run("add_model_"+payload, func(t *testing.T) {
			reqJSON := map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]any{
					"name": "add_model",
					"arguments": map[string]any{
						"targetDir": payload,
						"model":     "name: item\nfields:\n  - name: title\n    type: string\n",
					},
				},
			}
			reqBytes, _ := json.Marshal(reqJSON)

			in := bytes.NewReader(append(reqBytes, '\n'))
			var out bytes.Buffer
			srv := mcp.NewServer(in, &out, "test")

			if err := srv.Serve(); err != nil {
				t.Fatalf("Serve failed: %v", err)
			}

			var resp struct {
				Result struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response failed: %v", err)
			}

			if !resp.Result.IsError {
				t.Fatalf("expected isError=true for targetDir %q, got: %+v", payload, resp.Result)
			}
			if len(resp.Result.Content) == 0 || !strings.Contains(resp.Result.Content[0].Text, "outside workspace") {
				t.Fatalf("expected 'outside workspace' error message for %q, got: %+v", payload, resp.Result.Content)
			}
		})

		t.Run("apply_module_"+payload, func(t *testing.T) {
			reqJSON := map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "tools/call",
				"params": map[string]any{
					"name": "apply_module",
					"arguments": map[string]any{
						"targetDir": payload,
						"module":    "rbac",
					},
				},
			}
			reqBytes, _ := json.Marshal(reqJSON)

			in := bytes.NewReader(append(reqBytes, '\n'))
			var out bytes.Buffer
			srv := mcp.NewServer(in, &out, "test")

			if err := srv.Serve(); err != nil {
				t.Fatalf("Serve failed: %v", err)
			}

			var resp struct {
				Result struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response failed: %v", err)
			}

			if !resp.Result.IsError {
				t.Fatalf("expected isError=true for targetDir %q, got: %+v", payload, resp.Result)
			}
			if len(resp.Result.Content) == 0 || !strings.Contains(resp.Result.Content[0].Text, "outside workspace") {
				t.Fatalf("expected 'outside workspace' error message for %q, got: %+v", payload, resp.Result.Content)
			}
		})
	}
}
