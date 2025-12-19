package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JNZader/ai-toolkit/goreview/internal/config"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
)

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

// FileCache implementacion de cache en sistema de archivos
type FileCache struct {
	dir string
	ttl time.Duration
}

// NewFileCache crea un nuevo FileCache
func NewFileCache(cfg config.CacheConfig) (*FileCache, error) {
	if !cfg.Enabled {
		return nil, nil // Cache deshabilitado
	}

	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	return &FileCache{
		dir: cfg.Dir,
		ttl: cfg.TTL,
	}, nil
}

// Get implementacion
func (c *FileCache) Get(key string) (*providers.ReviewResponse, bool, error) {
	if c == nil {
		return nil, false, nil
	}

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

	return &response, true, nil
}

// Set implementacion
func (c *FileCache) Set(key string, response *providers.ReviewResponse) error {
	if c == nil {
		return nil
	}

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

// Clear limpia todo el cache
func (c *FileCache) Clear() error {
	if c == nil {
		return nil
	}
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
