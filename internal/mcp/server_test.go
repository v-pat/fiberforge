package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestMCPVerifyProject(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("FIBERFORGE_WORKSPACE_ROOT", tmpDir)

	goMod := "module example.com/mcpverify\n\ngo 1.23\n"
	if err := os.WriteFile(tmpDir+"/go.mod", []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	mainGo := `package main

func main() {}
`
	if err := os.WriteFile(tmpDir+"/main.go", []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	reqData, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "verify_project",
			"arguments": map[string]any{
				"targetDir": tmpDir,
				"skipTests": true,
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
			Success bool `json:"success"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !resp.Result.Success {
		t.Errorf("expected verify_project to succeed, got %v", out.String())
	}
}

func TestMCPIntrospectDatabase(t *testing.T) {
	sqlDDL := `
CREATE TABLE products (
	id BIGSERIAL PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	price NUMERIC(10, 2) NOT NULL,
	active BOOLEAN DEFAULT true
);
`
	reqData, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "introspect_database",
			"arguments": map[string]any{
				"sql":     sqlDDL,
				"appName": "store-api",
				"driver":  "postgres",
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
			AppName    string `json:"appName"`
			ModelCount int    `json:"modelCount"`
			Schema     string `json:"schema"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Result.AppName != "store-api" {
		t.Errorf("expected appName 'store-api', got %q", resp.Result.AppName)
	}
	if resp.Result.ModelCount != 1 {
		t.Errorf("expected 1 model, got %d", resp.Result.ModelCount)
	}
	if !strings.Contains(resp.Result.Schema, "name: Product") {
		t.Errorf("expected schema to contain 'name: Product', got:\n%s", resp.Result.Schema)
	}
}

func TestMCPSearchBlueprints(t *testing.T) {
	reqData, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "search_blueprints",
			"arguments": map[string]any{
				"query": "stripe",
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
			Count      int `json:"count"`
			Blueprints []struct {
				ID string `json:"id"`
			} `json:"blueprints"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Result.Count == 0 || len(resp.Result.Blueprints) == 0 {
		t.Fatalf("expected at least 1 blueprint matching 'stripe'")
	}
	if resp.Result.Blueprints[0].ID != "stripe-billing" {
		t.Errorf("expected stripe-billing, got %s", resp.Result.Blueprints[0].ID)
	}
}

func TestMCPInspectBlueprint(t *testing.T) {
	reqData, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "inspect_blueprint",
			"arguments": map[string]any{
				"blueprintId": "stripe-billing",
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
			Blueprint struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Category string `json:"category"`
			} `json:"blueprint"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Result.Blueprint.ID != "stripe-billing" {
		t.Errorf("expected stripe-billing, got %s", resp.Result.Blueprint.ID)
	}
}

func TestMCPApplyBlueprintDryRun(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("FIBERFORGE_WORKSPACE_ROOT", tempDir)
	_ = os.WriteFile(tempDir+"/fiberforge.yaml", []byte("appName: testapi\ndatabase: postgres\n"), 0644)
	reqData, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "apply_blueprint",
			"arguments": map[string]any{
				"blueprint": "stripe-billing",
				"targetDir": tempDir,
				"dryRun":    true,
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
			Blueprint   string   `json:"blueprint"`
			AddedModels []string `json:"addedModels"`
			DryRun      bool     `json:"dryRun"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Result.Blueprint != "stripe-billing" {
		t.Errorf("expected stripe-billing, got %s", resp.Result.Blueprint)
	}
	if len(resp.Result.AddedModels) == 0 {
		t.Errorf("expected added models in dry-run result")
	}
}

