package report

import (
	"encoding/json"
	"io"

	"github.com/JNZader/ai-toolkit/goreview/internal/review"
)

// SARIFReporter genera reportes en formato SARIF (Static Analysis Results Interchange Format)
type SARIFReporter struct{}

// Generate escribe el resultado en formato SARIF
func (r *SARIFReporter) Generate(result *review.Result, w io.Writer) error {
	sarif := map[string]interface{}{
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":           "GoReview",
						"informationUri": "https://github.com/JNZader/ai-toolkit",
						"rules":          []interface{}{}, // TODO: Populate if rules meta is available
					},
				},
				"results": r.buildResults(result),
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarif)
}

func (r *SARIFReporter) buildResults(result *review.Result) []map[string]interface{} {
	results := []map[string]interface{}{}

	for _, fileRes := range result.Files {
		if fileRes.Response == nil {
			continue
		}

		for _, issue := range fileRes.Response.Issues {
			sarifIssue := map[string]interface{}{
				"ruleId": issue.ID,
				"message": map[string]interface{}{
					"text": issue.Message,
				},
				"level": r.mapSeverity(issue.Severity),
				"locations": []map[string]interface{}{
					{
						"physicalLocation": map[string]interface{}{
							"artifactLocation": map[string]interface{}{
								"uri": fileRes.File,
							},
							"region": map[string]interface{}{
								"startLine": issue.Location.StartLine,
							},
						},
					},
				},
			}
			results = append(results, sarifIssue)
		}
	}

	return results
}

func (r *SARIFReporter) mapSeverity(sev string) string {
	switch sev {
	case "critical", "error":
		return "error"
	case "warning":
		return "warning"
	case "info":
		return "note"
	default:
		return "none"
	}
}
