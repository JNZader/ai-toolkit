package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/JNZader/ai-toolkit/goreview/internal/git"
	"github.com/JNZader/ai-toolkit/goreview/internal/providers"
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate documentation for changes",
	Long: `Analyzes the git diff and generates a human-readable summary/changelog using AI.
Useful for commit messages or updating CHANGELOG.md.`,
	RunE: runDoc,
}

func init() {
	rootCmd.AddCommand(docCmd)
	docCmd.Flags().BoolVar(&staged, "staged", false, "document staged changes")
	docCmd.Flags().StringVar(&commit, "commit", "", "document specific commit")
}

func runDoc(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// 1. Setup Git
	gitRepo, err := git.NewRepository(cfg.Git.RepoPath, cfg.Git.IgnorePatterns)
	if err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// 2. Setup Provider
	providerFactory := providers.NewFactory(&cfg.Provider)
	provider, err := providerFactory.Create()
	if err != nil {
		return fmt.Errorf("provider creation failed: %w", err)
	}

	// 3. Get Diff
	var diff *git.Diff
	if staged {
		diff, err = gitRepo.GetStagedDiff(context.Background())
	} else if commit != "" {
		diff, err = gitRepo.GetCommitDiff(context.Background(), commit)
	} else {
		// Default to staged if nothing specified, or error? Let's default to staged for ease.
		diff, err = gitRepo.GetStagedDiff(context.Background())
	}

	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if len(diff.Files) == 0 {
		fmt.Println("No changes to document.")
		return nil
	}

	// 4. Construct Diff String
	var sb strings.Builder
	for _, f := range diff.Files {
		// Ignorar binarios o borrados
		if f.IsBinary {
			continue
		}
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

	// 5. Call AI
	fmt.Fprintf(cmd.ErrOrStderr(), "📝 Generating documentation with %s...\n", cfg.Provider.Name)
	
	doc, err := provider.GenerateDocumentation(context.Background(), sb.String(), "")
	if err != nil {
		return err
	}

	// 6. Output
	fmt.Println(doc)
	return nil
}
