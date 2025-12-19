package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/JNZader/ai-toolkit/goreview/internal/config"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
)

// PERF-004: Default LRU cache size (number of entries)
const DefaultLRUCacheSize = 1000

// Cache interface para guardar resultados de reviews
type Cache interface {
	// Get obtiene un resultado del cache
	Get(key string) (*providers.ReviewResponse, bool, error)

	// Set guarda un resultado en el cache
	Set(key string, response *providers.ReviewResponse) error

	// ComputeKey genera una key unica para un request
	ComputeKey(req *providers.ReviewRequest) string

	// Clear limpia el cache
	Clear() error
}

// cacheEntry wraps a response with its timestamp for TTL checking
type cacheEntry struct {
	Response  *providers.ReviewResponse
	Timestamp time.Time
}

// FileCache implementacion de cache en sistema de archivos con LRU en memoria
type FileCache struct {
	dir      string
	ttl      time.Duration
	lruCache *lru.Cache[string, *cacheEntry] // PERF-004: In-memory LRU cache
	mu       sync.RWMutex
}

// NewFileCache crea un nuevo FileCache con LRU en memoria
func NewFileCache(cfg config.CacheConfig) (*FileCache, error) {
	if !cfg.Enabled {
		return nil, nil // Cache deshabilitado
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	// PERF-004: Calculate LRU cache size from MaxSizeMB
	// Assume average entry size of ~4KB, so 1MB ≈ 256 entries
	lruSize := DefaultLRUCacheSize
	if cfg.MaxSizeMB > 0 {
		lruSize = cfg.MaxSizeMB * 256
		if lruSize > 10000 {
			lruSize = 10000 // Cap at 10k entries to prevent excessive memory usage
		}
	}

	lruCache, err := lru.New[string, *cacheEntry](lruSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	return &FileCache{
		dir:      cfg.Dir,
		ttl:      cfg.TTL,
		lruCache: lruCache,
	}, nil
}

// Get implementacion - PERF-004: Check LRU cache first, then file cache
func (c *FileCache) Get(key string) (*providers.ReviewResponse, bool, error) {
	if c == nil {
		return nil, false, nil
	}

	// PERF-004: Check in-memory LRU cache first
	c.mu.RLock()
	if entry, ok := c.lruCache.Get(key); ok {
		c.mu.RUnlock()
		// Check TTL
		if time.Since(entry.Timestamp) <= c.ttl {
			return entry.Response, true, nil
		}
		// Expired in memory, remove it
		c.mu.Lock()
		c.lruCache.Remove(key)
		c.mu.Unlock()
	} else {
		c.mu.RUnlock()
	}

	// Fall back to file cache
	path := c.getPath(key)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	// Verificar TTL
	if time.Since(info.ModTime()) > c.ttl {
		_ = os.Remove(path) // Expired
		return nil, false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	var response providers.ReviewResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, false, err
	}

	// PERF-004: Populate LRU cache with file data for faster future access
	c.mu.Lock()
	c.lruCache.Add(key, &cacheEntry{
		Response:  &response,
		Timestamp: info.ModTime(),
	})
	c.mu.Unlock()

	return &response, true, nil
}

// Set implementacion - PERF-004: Store in both LRU and file cache
func (c *FileCache) Set(key string, response *providers.ReviewResponse) error {
	if c == nil {
		return nil
	}

	now := time.Now()

	// PERF-004: Store in LRU cache first (fast)
	c.mu.Lock()
	c.lruCache.Add(key, &cacheEntry{
		Response:  response,
		Timestamp: now,
	})
	c.mu.Unlock()

	// Then persist to file (slower but durable)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	path := c.getPath(key)
	return os.WriteFile(path, data, 0644)
}

// ComputeKey genera SHA-256 del request
func (c *FileCache) ComputeKey(req *providers.ReviewRequest) string {
	// Incluimos todos los campos relevantes en el hash
	input := fmt.Sprintf("%s|%s|%s|%v|%s",
		req.Diff,
		req.Language,
		req.Context,
		req.Rules,
		req.FilePath,
	)

	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// Clear limpia todo el cache (memoria y archivos)
func (c *FileCache) Clear() error {
	if c == nil {
		return nil
	}

	// PERF-004: Clear LRU cache
	c.mu.Lock()
	c.lruCache.Purge()
	c.mu.Unlock()

	return os.RemoveAll(c.dir)
}

func (c *FileCache) getPath(key string) string {
	// Usamos los primeros 2 caracteres como subdirectorio para no llenar un solo folder
	if len(key) < 2 {
		return filepath.Join(c.dir, key)
	}

	subdir := filepath.Join(c.dir, key[:2])
	_ = os.MkdirAll(subdir, 0755)

	return filepath.Join(subdir, key)
}
