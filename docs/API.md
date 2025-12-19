# GoReview API Documentation

## CLI Reference

### goreview review

Review code changes using AI.

**Usage:**
```bash
goreview review [flags] [files...]
```

**Flags:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| --staged | | Review staged changes | false |
| --commit | | Review specific commit | |
| --base | | Compare against base branch | main |
| --provider | -p | AI provider (ollama, claude, openai) | ollama |
| --model | -m | Model to use | qwen2.5-coder:7b |
| --format | -f | Output format (markdown, json, sarif, html) | markdown |
| --output | -o | Output file (default: stdout) | |
| --no-cache | | Disable caching | false |
| --verbose | -v | Verbose output | false |

---

## Configuration File

Create `.goreview.yaml` in your project root:

```yaml
# Provider settings
provider:
  name: ollama
  model: qwen2.5-coder:7b
  base_url: http://localhost:11434

# Review settings
review:
  min_severity: warning
  max_issues: 50

# Rules settings
rules:
  preset: standard

# Output settings
output:
  format: markdown
  color: true

# Cache settings
cache:
  enabled: true
  ttl: 24h
```
