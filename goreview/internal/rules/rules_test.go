package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()

	// Crear archivo de reglas dummy
	content := []byte(`
rules:
  - id: TEST-001
    description: Test rule
    severity: warning
    category: test
    languages: [go]
    prompt_context: "Check for testing"
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "test.yaml"), content, 0644); err != nil {
		t.Fatalf("failed to write rule file: %v", err)
	}

	loader := NewLoader(tmpDir)
	rules, err := loader.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "TEST-001" {
		t.Errorf("expected ID TEST-001, got %s", rules[0].ID)
	}
}

func TestFilter(t *testing.T) {
	rules := []Rule{
		{ID: "GO-001", Languages: []string{"go"}},
		{ID: "JS-001", Languages: []string{"javascript"}},
		{ID: "ALL-001", Languages: []string{"*"}},
		{ID: "PAT-001", Languages: []string{"go"}, Patterns: []string{`.*_test\.go`}},
	}

	// Test Go file
	filtered := Filter(rules, "go", "main.go")
	if len(filtered) != 2 { // GO-001, ALL-001
		t.Errorf("expected 2 rules for Go, got %d", len(filtered))
	}

	// Test JS file
	filtered = Filter(rules, "javascript", "app.js")
	if len(filtered) != 2 { // JS-001, ALL-001
		t.Errorf("expected 2 rules for JS, got %d", len(filtered))
	}

	// Test Pattern match
	filtered = Filter(rules, "go", "main_test.go")
	if len(filtered) != 3 { // GO-001, ALL-001, PAT-001
		t.Errorf("expected 3 rules for Go test file, got %d", len(filtered))
	}
}
