package mcp_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/v-pat/fiberforge/internal/mcp"
)

func TestMCPServerHandshake(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"ping","params":{}}
`)
	var out bytes.Buffer
	srv := mcp.NewServer(in, &out, "test")

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 response lines, got %d: %v", len(lines), lines)
	}

	var resp1 struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &resp1); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if resp1.Result.ServerInfo.Name != "fiberforge" {
		t.Errorf("expected server name 'fiberforge', got %q", resp1.Result.ServerInfo.Name)
	}
}

func TestMCPGetSchemaTemplate(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_schema_template","arguments":{"name":"blog"}}}`)
	var out bytes.Buffer
	srv := mcp.NewServer(in, &out, "test")

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp struct {
		Result struct {
			Template string `json:"template"`
			Schema   string `json:"schema"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Result.Template != "blog" {
		t.Errorf("expected template 'blog', got %q", resp.Result.Template)
	}
	if !strings.Contains(resp.Result.Schema, "appName: blog") {
		t.Errorf("expected schema content, got %q", resp.Result.Schema)
	}
}

func TestMCPListFieldTypes(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_field_types","arguments":{}}}`)
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
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(resp.Result.Content) == 0 || !strings.Contains(resp.Result.Content[0].Text, "Supported Field Types") {
		t.Errorf("unexpected list_field_types response: %v", resp.Result)
	}
}

func TestMCPExplainProject(t *testing.T) {
	schemaJSON := `{"appName":"testapp","database":"postgres","models":[{"name":"Post","endpoint":"posts","fields":[{"name":"title","type":"string"}]}]}`
	reqData, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_project",
			"arguments": map[string]any{
				"schema": schemaJSON,
			},
		},
	})

	in := strings.NewReader(string(reqData) + "\n")
	var out bytes.Buffer
	srv := mcp.NewServer(in, &out, "test")

	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp struct {
		Result struct {
			AppName string   `json:"appName"`
			Files   []string `json:"files"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Result.AppName != "testapp" {
		t.Errorf("expected appName 'testapp', got %q", resp.Result.AppName)
	}
	if len(resp.Result.Files) == 0 {
		t.Errorf("expected generated files list, got empty")
	}
}
