package review

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/JNZader/ai-toolkit/goreview/internal/cache"
	"github.com/JNZader/ai-toolkit/goreview/internal/config"
	"github.com/JNZader/ai-toolkit/goreview/internal/git"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
	"github.com/JNZader/ai-toolkit/goreview/internal/rules"
)

// Engine orquesta el proceso de code review
type Engine struct {
	cfg      *config.Config
	gitRepo  git.Repository
	provider providers.Provider
	cache    cache.Cache
	rules    []rules.Rule
}

// NewEngine crea un nuevo engine
func NewEngine(
	cfg *config.Config,
	gitRepo git.Repository,
	provider providers.Provider,
	cache cache.Cache,
	ruleSet []rules.Rule,
) *Engine {
	return &Engine{
		cfg:      cfg,
		gitRepo:  gitRepo,
		provider: provider,
		cache:    cache,
		rules:    ruleSet,
	}
}

// Run ejecuta el review
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	// 1. Obtener diff segun el modo configurado
	diff, err := e.getDiff(ctx)
	if err != nil {
		return nil, err
	}

	if len(diff.Files) == 0 {
		return &Result{Summary: "No changes found to review."}, nil
	}

	// 2. Procesar archivos (paralelizable)
	var wg sync.WaitGroup
	resultsChan := make(chan *FileResult, len(diff.Files))
	semaphore := make(chan struct{}, 5) // Max 5 concurrencia para no saturar LLM local

	start := time.Now()

	for _, file := range diff.Files {
		// Ignorar archivos borrados o binarios
		if file.Status == git.FileDeleted || file.IsBinary {
			continue
		}

		wg.Add(1)
		go func(f git.FileDiff) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			res, err := e.reviewFile(ctx, f)
			if err != nil {
				// Log error but continue
				// fmt.Printf("Error reviewing %s: %v\n", f.Path, err)
				resultsChan <- &FileResult{File: f.Path, Error: err}
			} else {
				resultsChan <- res
			}
		}(file)
	}

	wg.Wait()
	close(resultsChan)

	// 3. Agregar resultados
	finalResult := &Result{
		Stats: diff.Stats,
		Files: make([]FileResult, 0),
	}

	for res := range resultsChan {
		if res != nil {
			finalResult.Files = append(finalResult.Files, *res)
			if res.Response != nil {
				finalResult.TotalIssues += len(res.Response.Issues)
			}
		}
	}

	finalResult.Duration = time.Since(start)
	return finalResult, nil
}

func (e *Engine) getDiff(ctx context.Context) (*git.Diff, error) {
	switch e.cfg.Review.Mode {
	case "staged":
		return e.gitRepo.GetStagedDiff(ctx)
	case "commit":
		return e.gitRepo.GetCommitDiff(ctx, e.cfg.Review.Commit)
	case "branch":
		return e.gitRepo.GetBranchDiff(ctx, e.cfg.Git.BaseBranch)
	case "files":
		return e.gitRepo.GetFileDiff(ctx, e.cfg.Review.Files)
	default:
		return nil, fmt.Errorf("unknown review mode: %s", e.cfg.Review.Mode)
	}
}

func (e *Engine) reviewFile(ctx context.Context, file git.FileDiff) (*FileResult, error) {
	// Filtrar reglas aplicables
	applicableRules := rules.Filter(e.rules, file.Language, file.Path)

	// Construir contexto de reglas
	var ruleDescriptions []string
	for _, r := range applicableRules {
		ruleDescriptions = append(ruleDescriptions, fmt.Sprintf("- %s: %s", r.ID, r.Description))
	}

	// Preparar request
	req := &providers.ReviewRequest{
		Diff:     formatDiff(file),
		Language: file.Language,
		FilePath: file.Path,
		Rules:    ruleDescriptions,
		Context:  e.cfg.Review.Context,
	}

	// Consultar Cache
	if e.cache != nil {
		key := e.cache.ComputeKey(req)
		if cached, found, _ := e.cache.Get(key); found {
			return &FileResult{File: file.Path, Response: cached, Cached: true}, nil
		}
	}

	// Llamar Provider
	resp, err := e.provider.Review(ctx, req)
	if err != nil {
		return nil, err
	}

	// Guardar Cache
	if e.cache != nil {
		key := e.cache.ComputeKey(req)
		_ = e.cache.Set(key, resp)
	}

	return &FileResult{File: file.Path, Response: resp, Cached: false}, nil
}

func formatDiff(file git.FileDiff) string {
	// Reconstruir diff legible para el LLM
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("File: %s\n", file.Path))
	for _, hunk := range file.Hunks {
		sb.WriteString(fmt.Sprintf("%s\n", hunk.Header))
		for _, line := range hunk.Lines {
			prefix := " "
			if line.Type == git.LineAddition {
				prefix = "+"
			} else if line.Type == git.LineDeletion {
				prefix = "-"
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", prefix, line.Content))
		}
	}
	return sb.String()
}

// Result resultado global del review
type Result struct {
	TotalIssues int           `json:"total_issues"`
	Duration    time.Duration `json:"duration"`
	Files       []FileResult  `json:"files"`
	Stats       git.DiffStats `json:"stats"`
	Summary     string        `json:"summary,omitempty"`
}

// FileResult resultado por archivo
type FileResult struct {
	File     string                    `json:"file"`
	Response *providers.ReviewResponse `json:"response,omitempty"`
	Error    error                     `json:"error,omitempty"`
	Cached   bool                      `json:"cached"`
}
