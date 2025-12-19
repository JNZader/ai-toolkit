package git

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)
	fileStatusRegex = regexp.MustCompile(`^(new file|deleted file|rename from|rename to|similarity index|index)`)
)

// parseDiff parsea un diff de git
func (r *GitRepository) parseDiff(rawDiff string) (*Diff, error) {
	diff := &Diff{
		Files:   []FileDiff{},
		RawDiff: rawDiff,
	}

	if strings.TrimSpace(rawDiff) == "" {
		return diff, nil
	}

	lines := strings.Split(rawDiff, "\n")
	var currentFile *FileDiff
	var currentHunk *Hunk
	oldLineNum, newLineNum := 0, 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Nuevo archivo
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Guardar archivo anterior si existe
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				if !r.shouldIgnore(currentFile.Path) {
					diff.Files = append(diff.Files, *currentFile)
				}
			}

			currentFile = &FileDiff{
				Path:     matches[2],
				OldPath:  matches[1],
				Status:   FileModified,
				Language: detectLanguage(matches[2]),
				Hunks:    []Hunk{},
			}
			currentHunk = nil
			continue
		}

		if currentFile == nil {
			continue
		}

		// Estado del archivo
		if fileStatusRegex.MatchString(line) {
			if strings.HasPrefix(line, "new file") {
				currentFile.Status = FileAdded
			} else if strings.HasPrefix(line, "deleted file") {
				currentFile.Status = FileDeleted
			} else if strings.HasPrefix(line, "rename from") {
				currentFile.Status = FileRenamed
			}
			continue
		}

		// Binary file
		if strings.HasPrefix(line, "Binary files") {
			currentFile.IsBinary = true
			continue
		}

		// Hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Guardar hunk anterior
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(matches[1])
			oldLines := 1
			if matches[2] != "" {
				oldLines, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newLines := 1
			if matches[4] != "" {
				newLines, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
				Header:   strings.TrimSpace(matches[5]),
				Lines:    []Line{},
			}
			oldLineNum = oldStart
			newLineNum = newStart
			continue
		}

		// Lineas de diff
		if currentHunk != nil && len(line) > 0 {
			var diffLine Line
			switch line[0] {
			case '+' :
				diffLine = Line{
					Type:    LineAddition,
					Content: line[1:],
					NewNum:  newLineNum,
				}
				newLineNum++
				currentFile.Stats.Additions++
				diff.Stats.Additions++
			case '-' :
				diffLine = Line{
					Type:    LineDeletion,
					Content: line[1:],
					OldNum:  oldLineNum,
				}
				oldLineNum++
				currentFile.Stats.Deletions++
				diff.Stats.Deletions++
			case ' ' :
				diffLine = Line{
					Type:    LineContext,
					Content: line[1:],
					OldNum:  oldLineNum,
					NewNum:  newLineNum,
				}
				oldLineNum++
				newLineNum++
			default:
				continue
			}
			currentHunk.Lines = append(currentHunk.Lines, diffLine)
		}
	}

	// Guardar ultimo archivo
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		if !r.shouldIgnore(currentFile.Path) {
			diff.Files = append(diff.Files, *currentFile)
		}
	}

	// Calcular total de cambios
	diff.Stats.Changes = diff.Stats.Additions + diff.Stats.Deletions

	return diff, nil
}

// detectLanguage detecta el lenguaje por extension
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	languages := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "typescript",
		".jsx":   "javascript",
		".java":  "java",
		".rs":    "rust",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".rb":    "ruby",
		".php":   "php",
		".cs":    "csharp",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".sh":    "bash",
		".bash":  "bash",
		".zsh":   "bash",
		".sql":   "sql",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".less":  "less",
		".md":    "markdown",
		".tf":    "terraform",
		".hcl":   "hcl",
	}

	if lang, ok := languages[ext]; ok {
		return lang
	}
	return "unknown"
}
