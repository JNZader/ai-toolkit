package report

import (
	"encoding/json"
	"io"

	"github.com/JNZader/ai-toolkit/goreview/internal/review"
)

// JSONReporter genera reportes en formato JSON
type JSONReporter struct{}

// Generate escribe el resultado en formato JSON
func (r *JSONReporter) Generate(result *review.Result, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
