package git

import (
	"context"
)

// Repository representa un repositorio Git
type Repository interface {
	// GetStagedDiff obtiene el diff de cambios staged
	GetStagedDiff(ctx context.Context) (*Diff, error)

	// GetCommitDiff obtiene el diff de un commit especifico
	GetCommitDiff(ctx context.Context, commitHash string) (*Diff, error)

	// GetBranchDiff obtiene el diff entre la branch actual y una base
	GetBranchDiff(ctx context.Context, baseBranch string) (*Diff, error)

	// GetFileDiff obtiene el diff de archivos especificos
	GetFileDiff(ctx context.Context, files []string) (*Diff, error)

	// GetCurrentBranch retorna el nombre de la branch actual
	GetCurrentBranch(ctx context.Context) (string, error)

	// GetHeadCommit retorna el hash del commit HEAD
	GetHeadCommit(ctx context.Context) (string, error)

	// IsClean verifica si el working directory esta limpio
	IsClean(ctx context.Context) (bool, error)
}

// Diff representa un conjunto de cambios
type Diff struct {
	Files   []FileDiff `json:"files"`
	Stats   DiffStats  `json:"stats"`
	RawDiff string     `json:"-"`
}

// FileDiff representa los cambios en un archivo
type FileDiff struct {
	Path       string      `json:"path"`
	OldPath    string      `json:"old_path,omitempty"` // Para renames
	Status     FileStatus  `json:"status"`
	Language   string      `json:"language"`
	Hunks      []Hunk      `json:"hunks"`
	Stats      DiffStats   `json:"stats"`
	IsBinary   bool        `json:"is_binary"`
	Content    string      `json:"-"` // Contenido completo del archivo
	OldContent string      `json:"-"` // Contenido anterior (para context)
}

// FileStatus representa el estado de un archivo
type FileStatus string

const (
	FileAdded    FileStatus = "added"
	FileModified FileStatus = "modified"
	FileDeleted  FileStatus = "deleted"
	FileRenamed  FileStatus = "renamed"
	FileCopied   FileStatus = "copied"
)

// Hunk representa una seccion de cambios
type Hunk struct {
	OldStart int      `json:"old_start"`
	OldLines int      `json:"old_lines"`
	NewStart int      `json:"new_start"`
	NewLines int      `json:"new_lines"`
	Header   string   `json:"header"`
	Lines    []Line   `json:"lines"`
}

// Line representa una linea de cambio
type Line struct {
	Type    LineType `json:"type"`
	Content string   `json:"content"`
	OldNum  int      `json:"old_num,omitempty"`
	NewNum  int      `json:"new_num,omitempty"`
}

// LineType representa el tipo de linea
type LineType string

const (
	LineContext  LineType = "context"
	LineAddition LineType = "addition"
	LineDeletion LineType = "deletion"
)

// DiffStats estadisticas del diff
type DiffStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Changes   int `json:"changes"`
}
