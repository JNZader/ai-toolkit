package commands

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JNZader/ai-toolkit/goreview/internal/cache"
	"github.com/JNZader/ai-toolkit/goreview/internal/git"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
	"github.com/JNZader/ai-toolkit/goreview/internal/review"
	"github.com/JNZader/ai-toolkit/goreview/internal/rules"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Review changes and commit with AI-generated message",
	Long: `Performs a code review on staged changes. If the review passes,
it generates a conventional commit message using AI and commits the changes.`,
	RunE: runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	ctx := context.Background()

	// 1. Setup (Git, Provider, Cache, Rules)
	gitRepo, err := git.NewRepository(cfg.Git.RepoPath, cfg.Git.IgnorePatterns)
	if err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Verificar si hay cambios staged
	diff, err := gitRepo.GetStagedDiff(ctx)
	if err != nil {
		return err
	}
	if len(diff.Files) == 0 {
		return fmt.Errorf("no staged changes found")
	}

	providerFactory := providers.NewFactory(&cfg.Provider)
	provider, err := providerFactory.Create()
	if err != nil {
		return fmt.Errorf("provider creation failed: %w", err)
	}

	reviewCache, err := cache.NewFileCache(cfg.Cache)
	if err != nil {
		return fmt.Errorf("cache init failed: %w", err)
	}

	// TODO: Load rules from file system in a real scenario
	ruleSet := []rules.Rule{}

	engine := review.NewEngine(cfg, gitRepo, provider, reviewCache, ruleSet)

	// 2. Ejecutar Review
	fmt.Println("🔍 Running pre-commit review...")
	result, err := engine.Run(ctx)
	if err != nil {
		return err
	}

	if result.TotalIssues > 0 {
		fmt.Printf("\n❌ Found %d issues. Review failed.\n", result.TotalIssues)
		// Mostrar issues brevemente
		for _, f := range result.Files {
			if f.Response != nil && len(f.Response.Issues) > 0 {
				fmt.Printf("  %s:\n", f.File)
				for _, i := range f.Response.Issues {
					fmt.Printf("    - [%s] %s\n", i.Severity, i.Message)
				}
			}
		}
		return fmt.Errorf("review failed")
	}

	fmt.Println("✅ Review passed!")

	// 3. Generar Mensaje
	fmt.Printf("🤖 Generating commit message with %s...\n", cfg.Provider.Name)
	
	// Reconstruir diff string para el prompt (reusing logic logic from doc command would be better but keeping it simple)
	var sb strings.Builder
	for _, f := range diff.Files {
		if f.IsBinary { continue }
		sb.WriteString(fmt.Sprintf("File: %s\n", f.Path))
		for _, h := range f.Hunks {
			sb.WriteString(h.Header + "\n")
			for _, l := range h.Lines {
				prefix := " "
				if l.Type == git.LineAddition {
					prefix = "+"
				} else if l.Type == git.LineDeletion {
					prefix = "-"
				}
				sb.WriteString(prefix + l.Content + "\n")
			}
		}
	}

	msg, err := provider.GenerateCommitMessage(ctx, sb.String())
	if err != nil {
		return fmt.Errorf("failed to generate message: %w", err)
	}

	// Limpiar mensaje (quitar comillas si las hay)
	msg = strings.Trim(msg, "`")

	// 4. Prompt usuario
	fmt.Printf("\nProposed Commit Message:\n%s\n\n", msg)
	fmt.Print("Commit with this message? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" || response == "" {
		// 5. Execute Git Commit
		// Usamos --no-verify para saltar el pre-commit hook ya que acabamos de correr el review
		fmt.Println("🚀 Committing...")
		gitCmd := exec.Command("git", "commit", "-m", msg, "--no-verify")
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			return err
		}

		// Obtener la raiz del repo
		rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
		var rootPathBytes []byte
		rootPathBytes, err = rootCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get git root path: %w", err)
		}
		rootPath := strings.TrimSpace(string(rootPathBytes))

		// Generar reporte de exito tambien para que sea consistente con el hook
		var reviewReportOutputBytes []byte
		reviewReportOutputBytes, err = exec.Command(filepath.Join(rootPath, "goreview/build/goreview"), "review", "--staged", "--format", "markdown").Output()
		if err != nil {
			fmt.Printf("⚠️  Fallo al generar el reporte de exito: %v\n", err)
		}
		_ = os.MkdirAll(filepath.Join(rootPath, "logs"), 0755)
		_ = os.WriteFile(filepath.Join(rootPath, "logs", "last_successful_review.md"), reviewReportOutputBytes, 0644)
		fmt.Println("📄 Reporte detallado guardado en logs/last_successful_review.md")

		// 6. Actualizar el hash para que el hook sepa que ya revisamos esto
		if rootPath != "" {
			// Calculate hash using native Go (avoid shell command injection)
			diffCmd := exec.Command("git", "diff", "--cached")
			diffCmd.Dir = rootPath
			diffOutput, err := diffCmd.Output()
			if err == nil && len(diffOutput) > 0 {
				hashBytes := sha256.Sum256(diffOutput)
				hash := hex.EncodeToString(hashBytes[:])

				dotGoreview := filepath.Join(rootPath, ".goreview")
				_ = os.MkdirAll(dotGoreview, 0755)
				_ = os.WriteFile(filepath.Join(dotGoreview, "last_reviewed_hash"), []byte(hash), 0644)
			}
		}
		return nil
	}

	fmt.Println("❌ Commit aborted.")
	return nil
}
