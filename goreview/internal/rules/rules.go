package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// RuleSet conjunto de reglas
type RuleSet struct {
	Rules []Rule `yaml:"rules"`
}

// Rule definicion de una regla
type Rule struct {
	ID            string   `yaml:"id"`
	Description   string   `yaml:"description"`
	Severity      string   `yaml:"severity"`
	Category      string   `yaml:"category"`
	Languages     []string `yaml:"languages"`
	Tags          []string `yaml:"tags"`
	Patterns      []string `yaml:"patterns"`       // Regex patterns to match files/content
	AntiPatterns  []string `yaml:"anti_patterns"`  // Regex to exclude
	PromptContext string   `yaml:"prompt_context"` // Contexto extra para el LLM
}

// Loader carga reglas desde archivos
type Loader struct {
	rulesDir string
}

// NewLoader crea un nuevo loader
func NewLoader(rulesDir string) *Loader {
	return &Loader{rulesDir: rulesDir}
}

// Load carga todas las reglas del directorio configurado
func (l *Loader) Load() ([]Rule, error) {
	var rules []Rule

	err := filepath.Walk(l.rulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}

		loadedRules, err := l.loadFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load rules from %s: %w", path, err)
		}
		rules = append(rules, loadedRules...)
		return nil
	})

	return rules, err
}

// loadFromFile carga reglas de un archivo especifico
func (l *Loader) loadFromFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ruleSet RuleSet
	if err := yaml.Unmarshal(data, &ruleSet); err != nil {
		return nil, err
	}

	return ruleSet.Rules, nil
}

// Filter filtra reglas aplicables a un archivo/lenguaje
func Filter(rules []Rule, language string, filePath string) []Rule {
	var applicable []Rule

	for _, rule := range rules {
		// Filtro por lenguaje
		langMatch := false
		for _, lang := range rule.Languages {
			if lang == "*" || lang == language {
				langMatch = true
				break
			}
		}
		if !langMatch {
			continue
		}

		// Filtro por patterns (si existen)
		if len(rule.Patterns) > 0 {
			patternMatch := false
			for _, pattern := range rule.Patterns {
				matched, _ := regexp.MatchString(pattern, filePath)
				if matched {
					patternMatch = true
					break
				}
			}
			if !patternMatch {
				continue
			}
		}

		applicable = append(applicable, rule)
	}

	return applicable
}
