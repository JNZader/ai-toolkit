package cache

import (
	"testing"
	"time"

	"github.com/JNZader/ai-toolkit/goreview/internal/config"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
)

func TestFileCache(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.CacheConfig{
		Enabled: true,
		Dir:     tmpDir,
		TTL:     1 * time.Minute,
	}

	c, err := NewFileCache(cfg)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	req := &providers.ReviewRequest{
		Diff:     "diff",
		Language: "go",
	}
	key := c.ComputeKey(req)

	// Test Get empty
	_, found, errGet := c.Get(key)
	if errGet != nil {
		t.Fatalf("Get failed: %v", errGet)
	}
	if found {
		t.Error("expected not found")
	}

	// Test Set
	resp := &providers.ReviewResponse{
		Summary: "test summary",
	}
	if err := c.Set(key, resp); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get found
	cachedResp, found, errGet2 := c.Get(key)
	if errGet2 != nil {
		t.Fatalf("Get failed: %v", errGet2)
	}
	if !found {
		t.Error("expected found")
	}
	if cachedResp.Summary != "test summary" {
		t.Errorf("expected summary 'test summary', got %s", cachedResp.Summary)
	}
}

func TestFileCache_TTL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.CacheConfig{
		Enabled: true,
		Dir:     tmpDir,
		TTL:     1 * time.Nanosecond, // Expira inmediatamente
	}

	c, err := NewFileCache(cfg)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}
	key := "test-key"
	if err := c.Set(key, &providers.ReviewResponse{}); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(1 * time.Millisecond)

	_, found, _ := c.Get(key)
	if found {
		t.Error("expected cache miss due to TTL")
	}
}

func TestComputeKey_Determinism(t *testing.T) {
	c := &FileCache{}
	req1 := &providers.ReviewRequest{Diff: "a", Language: "go"}
	req2 := &providers.ReviewRequest{Diff: "a", Language: "go"}
	req3 := &providers.ReviewRequest{Diff: "b", Language: "go"}

	key1 := c.ComputeKey(req1)
	key2 := c.ComputeKey(req2)
	key3 := c.ComputeKey(req3)

	if key1 != key2 {
		t.Error("same request should produce same key")
	}
	if key1 == key3 {
		t.Error("different request should produce different key")
	}
}
