package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewRepository(t *testing.T) {
	// Crear repo temporal
	tmpDir := t.TempDir()

	// Inicializar git
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git: %v", err)
	}

	// Configurar git
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()

	repo, err := NewRepository(tmpDir, nil)
	if err != nil {
		t.Fatalf("NewRepository failed: %v", err)
	}

	if repo.path != tmpDir {
		// En Windows, los paths pueden diferir ligeramente en formato (backslash vs slash) o capitalizacion
		// Usamos filepath.Clean para comparar
		if filepath.Clean(repo.path) != filepath.Clean(tmpDir) {
			t.Errorf("expected path %s, got %s", tmpDir, repo.path)
		}
	}
}

func TestNewRepository_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := NewRepository(tmpDir, nil)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestParseDiff(t *testing.T) {
	repo := &GitRepository{ignorePatterns: []string{}}

	rawDiff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

+import "fmt"
+
 func main() {
-    println("hello")
+    fmt.Println("hello")
 }
`
	diff, err := repo.parseDiff(rawDiff)
	if err != nil {
		t.Fatalf("parseDiff failed: %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(diff.Files))
	}

	file := diff.Files[0]
	if file.Path != "main.go" {
		t.Errorf("expected path main.go, got %s", file.Path)
	}

	if file.Language != "go" {
		t.Errorf("expected language go, got %s", file.Language)
	}

	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}

	if file.Stats.Additions != 3 {
		t.Errorf("expected 3 additions, got %d", file.Stats.Additions)
	}

	if file.Stats.Deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", file.Stats.Deletions)
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.ts", "typescript"},
		{"style.css", "css"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			lang := detectLanguage(tt.path)
			if lang != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, lang)
			}
		})
	}
}

func TestStagedDiff(t *testing.T) {
	// Crear repo temporal
	tmpDir := t.TempDir()

	// Setup git
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		cmd.Run()
	}

	// Crear archivo
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)

	// Stage archivo
	exec.Command("git", "-C", tmpDir, "add", "test.go").Run()

	repo, _ := NewRepository(tmpDir, nil)
	diff, err := repo.GetStagedDiff(context.Background())

	if err != nil {
		t.Fatalf("GetStagedDiff failed: %v", err)
	}

	if len(diff.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(diff.Files))
	}
}
