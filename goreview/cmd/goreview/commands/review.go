package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/JNZader/ai-toolkit/goreview/internal/cache"
	"github.com/JNZader/ai-toolkit/goreview/internal/git"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
	"github.com/JNZader/ai-toolkit/goreview/internal/review"
	"github.com/JNZader/ai-toolkit/goreview/internal/rules"
)

var reviewCmd = &cobra.Command{
	Use:   "review [files...]",
	Short: "Review code changes",
	Long: `Review code changes using AI.

By default, reviews staged changes. Use flags to specify different targets.

Examples:
  # Review staged changes
  goreview review --staged

  # Review last commit
  goreview review --commit HEAD

  # Review specific commit
  goreview review --commit abc123

  # Review changes against a branch
  goreview review --base main

  # Review specific files
  goreview review file1.go file2.go

  # Review with specific provider
  goreview review --staged --provider claude --model claude-3-opus`,
	RunE: runReview,
}

var (
	staged   bool
	commit   string
	base     string
	noCache  bool
)

func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().BoolVar(&staged, "staged", false, "review staged changes")
	reviewCmd.Flags().StringVar(&commit, "commit", "", "review specific commit")
	reviewCmd.Flags().StringVar(&base, "base", "", "review changes against base branch")
	reviewCmd.Flags().BoolVar(&noCache, "no-cache", false, "disable cache")
}

func runReview(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// 1. Configurar modo
	if staged {
		cfg.Review.Mode = "staged"
	} else if commit != "" {
		cfg.Review.Mode = "commit"
		cfg.Review.Commit = commit
	} else if base != "" {
		cfg.Review.Mode = "branch"
		cfg.Git.BaseBranch = base
	} else if len(args) > 0 {
		cfg.Review.Mode = "files"
		cfg.Review.Files = args
	}

	if noCache {
		cfg.Cache.Enabled = false
	}

	// 2. Inicializar componentes
	// Git
	gitRepo, err := git.NewRepository(cfg.Git.RepoPath, cfg.Git.IgnorePatterns)
	if err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Provider
	providerFactory := providers.NewFactory(&cfg.Provider)
	provider, err := providerFactory.Create()
	if err != nil {
		return fmt.Errorf("provider creation failed: %w", err)
	}

	// Cache
	reviewCache, err := cache.NewFileCache(cfg.Cache)
	if err != nil {
		return fmt.Errorf("cache init failed: %w", err)
	}

	// Rules
	// TODO: Load from configured directory
	ruleSet := []rules.Rule{} 

	// 3. Crear y ejecutar Engine
	engine := review.NewEngine(cfg, gitRepo, provider, reviewCache, ruleSet)

	fmt.Printf("Starting review with %s (%s)...\n", cfg.Provider.Name, cfg.Provider.Model)
	result, err := engine.Run(context.Background())
	if err != nil {
		return err
	}

	// 4. Mostrar resultados (Simple text report por ahora)
	fmt.Printf("\nReview Complete in %s\n", result.Duration)
	fmt.Printf("Total Issues: %d\n", result.TotalIssues)
	
	for _, file := range result.Files {
		if file.Error != nil {
			fmt.Printf("  [!] %s: Error: %v\n", file.File, file.Error)
			continue
		}
		
		status := "Analyzed"
		if file.Cached {
			status = "Cached"
		}
		
		issuesCount := 0
		if file.Response != nil {
			issuesCount = len(file.Response.Issues)
		}
		
		fmt.Printf("  [%s] %s: %d issues\n", status, file.File, issuesCount)
		
		if file.Response != nil {
			for _, issue := range file.Response.Issues {
				fmt.Printf("    - [%s] %s: %s\n", issue.Severity, issue.Type, issue.Message)
			}
		}
	}

	return nil
}
