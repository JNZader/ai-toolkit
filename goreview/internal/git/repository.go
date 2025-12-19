package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRepository implementa Repository usando comandos git
type GitRepository struct {
	path           string
	ignorePatterns []string
}

// NewRepository crea un nuevo GitRepository
func NewRepository(path string, ignorePatterns []string) (*GitRepository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verificar que es un repo git
	// Usamos runGitHelper porque runGit ya existe como funcion helper abajo
	if _, err := runGitHelper(absPath, "rev-parse", "--git-dir"); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", absPath)
	}

	return &GitRepository{
		path:           absPath,
		ignorePatterns: ignorePatterns,
	}, nil
}

// GetStagedDiff obtiene el diff de cambios staged
func (r *GitRepository) GetStagedDiff(ctx context.Context) (*Diff, error) {
	output, err := r.runGitCtx(ctx, "diff", "--cached", "--unified=3")
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %w", err)
	}

	return r.parseDiff(output)
}

// GetCommitDiff obtiene el diff de un commit especifico
func (r *GitRepository) GetCommitDiff(ctx context.Context, commitHash string) (*Diff, error) {
	output, err := r.runGitCtx(ctx, "show", commitHash, "--unified=3", "--format=")
	if err != nil {
		return nil, fmt.Errorf("failed to get commit diff: %w", err)
	}

	return r.parseDiff(output)
}

// GetBranchDiff obtiene el diff entre la branch actual y una base
func (r *GitRepository) GetBranchDiff(ctx context.Context, baseBranch string) (*Diff, error) {
	// Obtener merge-base
	mergeBase, err := r.runGitCtx(ctx, "merge-base", baseBranch, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge-base: %w", err)
	}

	output, err := r.runGitCtx(ctx, "diff", strings.TrimSpace(mergeBase), "HEAD", "--unified=3")
	if err != nil {
		return nil, fmt.Errorf("failed to get branch diff: %w", err)
	}

	return r.parseDiff(output)
}

// GetFileDiff obtiene el diff de archivos especificos
func (r *GitRepository) GetFileDiff(ctx context.Context, files []string) (*Diff, error) {
	args := append([]string{"diff", "--unified=3", "--"}, files...)
	output, err := r.runGitCtx(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get file diff: %w", err)
	}

	return r.parseDiff(output)
}

// GetCurrentBranch retorna el nombre de la branch actual
func (r *GitRepository) GetCurrentBranch(ctx context.Context) (string, error) {
	output, err := r.runGitCtx(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// GetHeadCommit retorna el hash del commit HEAD
func (r *GitRepository) GetHeadCommit(ctx context.Context) (string, error) {
	output, err := r.runGitCtx(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD commit: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// IsClean verifica si el working directory esta limpio
func (r *GitRepository) IsClean(ctx context.Context) (bool, error) {
	output, err := r.runGitCtx(ctx, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to get git status: %w", err)
	}
	return strings.TrimSpace(output) == "", nil
}

// runGitCtx ejecuta un comando git con contexto
func (r *GitRepository) runGitCtx(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// runGitHelper ejecuta un comando git (renombrado para evitar conflictos)
func runGitHelper(path string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// shouldIgnore verifica si un archivo debe ser ignorado
func (r *GitRepository) shouldIgnore(path string) bool {
	for _, pattern := range r.ignorePatterns {
		matched, _ := filepath.Match(pattern, path)
		if matched {
			return true
		}
		// Tambien verificar contra el basename
		matched, _ = filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
	}
	return false
}
