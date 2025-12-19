package providers

import (
	"context"
)

// Provider interface para diferentes backends de IA
type Provider interface {
	// Name retorna el nombre del provider
	Name() string

	// Review realiza un code review
	Review(ctx context.Context, request *ReviewRequest) (*ReviewResponse, error)

	// HealthCheck verifica que el provider esta disponible
	HealthCheck(ctx context.Context) error

	// Close cierra conexiones
	Close() error
}

// ReviewRequest representa una solicitud de review
type ReviewRequest struct {
	// Diff del codigo a revisar
	Diff string `json:"diff"`

	// Lenguaje del codigo
	Language string `json:"language"`

	// Contexto adicional
	Context string `json:"context,omitempty"`

	// Reglas a aplicar
	Rules []string `json:"rules,omitempty"`

	// Path del archivo
	FilePath string `json:"file_path,omitempty"`

	// Contenido completo del archivo (para contexto)
	FileContent string `json:"file_content,omitempty"`
}

// ReviewResponse representa la respuesta del review
type ReviewResponse struct {
	// Issues encontrados
	Issues []Issue `json:"issues"`

	// Resumen general
	Summary string `json:"summary"`

	// Puntuacion general (0-100)
	Score int `json:"score"`

	// Tokens utilizados
	TokensUsed int `json:"tokens_used"`

	// Tiempo de procesamiento en ms
	ProcessingTime int64 `json:"processing_time_ms"`
}

// Issue representa un problema encontrado
type Issue struct {
	// ID unico del issue
	ID string `json:"id"`

	// Tipo de issue: bug, security, performance, style, etc.
	Type IssueType `json:"type"`

	// Severidad: info, warning, error, critical
	Severity Severity `json:"severity"`

	// Mensaje descriptivo
	Message string `json:"message"`

	// Sugerencia de fix
	Suggestion string `json:"suggestion,omitempty"`

	// Ubicacion en el archivo
	Location *Location `json:"location,omitempty"`

	// Regla que detecto el issue
	RuleID string `json:"rule_id,omitempty"`

	// Codigo de ejemplo corregido
	FixedCode string `json:"fixed_code,omitempty"`
}

// Location representa la ubicacion de un issue
type Location struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line,omitempty"`
	StartCol  int    `json:"start_col,omitempty"`
	EndCol    int    `json:"end_col,omitempty"`
}

// IssueType tipos de issues
type IssueType string

const (
	IssueTypeBug           IssueType = "bug"
	IssueTypeSecurity      IssueType = "security"
	IssueTypePerformance   IssueType = "performance"
	IssueTypeStyle         IssueType = "style"
	IssueTypeMaintenance   IssueType = "maintenance"
	IssueTypeDocumentation IssueType = "documentation"
	IssueTypeBestPractice  IssueType = "best_practice"
)

// Severity niveles de severidad
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// SeverityLevel retorna el nivel numerico de severidad
func (s Severity) Level() int {
	levels := map[Severity]int{
		SeverityInfo:     1,
		SeverityWarning:  2,
		SeverityError:    3,
		SeverityCritical: 4,
	}
	return levels[s]
}
