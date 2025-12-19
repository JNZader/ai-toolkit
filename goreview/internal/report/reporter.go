package report

import (
	"io"

	"github.com/JNZader/ai-toolkit/goreview/internal/review"
)

// Reporter interface para generar reportes
type Reporter interface {
	// Generate escribe el reporte en el writer dado
	Generate(result *review.Result, w io.Writer) error
}

// Factory crea reporters
func NewReporter(format string) Reporter {
	switch format {
	case "markdown", "md":
		return &MarkdownReporter{}
	case "json":
		// return &JSONReporter{}
		return nil // TODO
	case "sarif":
		// return &SARIFReporter{}
		return nil // TODO
	default:
		return &MarkdownReporter{}
	}
}
