package review

import (
	"context"
	"testing"
	"time"

	"github.com/JNZader/ai-toolkit/goreview/internal/cache"
	"github.com/JNZader/ai-toolkit/goreview/internal/config"
	"github.com/JNZader/ai-toolkit/goreview/internal/git"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
	"github.com/JNZader/ai-toolkit/goreview/internal/rules"
)

// MockProvider
type MockProvider struct{}

func (m *MockProvider) Name() string                          { return "mock" }
func (m *MockProvider) HealthCheck(ctx context.Context) error { return nil }
func (m *MockProvider) Close() error                          { return nil }
func (m *MockProvider) Review(ctx context.Context, req *providers.ReviewRequest) (*providers.ReviewResponse, error) {
	return &providers.ReviewResponse{
		Issues: []providers.Issue{
			{ID: "issue-1", Message: "Test issue"},
		},
	}, nil
}
func (m *MockProvider) GenerateDocumentation(ctx context.Context, diff string, context string) (string, error) {
	return "Mock documentation", nil
}
func (m *MockProvider) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	return "feat: mock commit", nil
}

// MockGit
type MockGit struct{}

func (m *MockGit) GetStagedDiff(ctx context.Context) (*git.Diff, error) {
	return &git.Diff{
		Files: []git.FileDiff{
			{Path: "main.go", Status: git.FileModified, Language: "go"},
		},
	}, nil
}
func (m *MockGit) GetCommitDiff(ctx context.Context, hash string) (*git.Diff, error) { return nil, nil }
func (m *MockGit) GetBranchDiff(ctx context.Context, branch string) (*git.Diff, error) {
	return nil, nil
}
func (m *MockGit) GetFileDiff(ctx context.Context, files []string) (*git.Diff, error) {
	return nil, nil
}
func (m *MockGit) GetCurrentBranch(ctx context.Context) (string, error) { return "main", nil }
func (m *MockGit) GetHeadCommit(ctx context.Context) (string, error)    { return "hash", nil }
func (m *MockGit) IsClean(ctx context.Context) (bool, error)            { return true, nil }

func TestEngine_Run(t *testing.T) {
	cfg := &config.Config{
		Review: config.ReviewConfig{Mode: "staged"},
	}

	mockGit := &MockGit{}
	mockProvider := &MockProvider{}
	tmpDir := t.TempDir()

	cacheCfg := config.CacheConfig{Enabled: true, Dir: tmpDir, TTL: time.Hour}
	fileCache, _ := cache.NewFileCache(cacheCfg)

	ruleSet := []rules.Rule{}

	engine := NewEngine(cfg, mockGit, mockProvider, fileCache, ruleSet)

	res, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Engine run failed: %v", err)
	}

	if res.TotalIssues != 1 {
		t.Errorf("expected 1 issue, got %d", res.TotalIssues)
	}

	if len(res.Files) != 1 {
		t.Errorf("expected 1 file result, got %d", len(res.Files))
	}

	if !res.Files[0].Cached {
		t.Log("First run should not be cached")
	}

	// Second run should be cached
	res2, _ := engine.Run(context.Background())
	if !res2.Files[0].Cached {
		t.Error("Second run should be cached")
	}
}
