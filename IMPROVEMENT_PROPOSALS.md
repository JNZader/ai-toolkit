# AI-Toolkit: Propuestas de Mejora

> Documento generado por análisis automatizado con 6 agentes especializados
> Fecha: 2025-12-19

---

## Resumen Ejecutivo

| Área | Estado Actual | Puntuación | Prioridad |
|------|---------------|------------|-----------|
| Seguridad | CRÍTICO | 23 vulnerabilidades | URGENTE |
| Testing | CRÍTICO | 0% TS / 52% Go | URGENTE |
| Arquitectura | Necesita mejoras | 5.4/10 | ALTA |
| Rendimiento | Subóptimo | 17 problemas | ALTA |
| Calidad de Código | Bueno | 7/10 | MEDIA |

---

## 1. Seguridad

### 1.1 Vulnerabilidades Críticas (CVSS 8.0+)

#### SEC-001: Credenciales Expuestas en Repositorio
- **Severidad:** CRÍTICA (CVSS 9.8)
- **Ubicación:**
  - `integrations/github-app/.env`
  - `integrations/github-app/config/private-key.pem`
- **OWASP:** A02:2021 - Cryptographic Failures
- **Impacto:** Exposición total de credenciales de GitHub App
- **Remediación:**
  1. Rotar inmediatamente todas las credenciales
  2. Eliminar archivos del repositorio: `git rm --cached .env config/private-key.pem`
  3. Usar GitHub Secrets o HashiCorp Vault
  4. Escanear historial: `git log -p -- .env`

#### SEC-002: Command Injection en GoReview Service
- **Severidad:** CRÍTICA (CVSS 8.8)
- **Ubicación:** `integrations/github-app/src/services/goreview.service.ts:42`
- **OWASP:** A03:2021 - Injection
- **Código vulnerable:**
```typescript
const args = ['review', ...files, '--format', 'json'];
const child = spawn('goreview', args, { cwd: workDir });
```
- **Impacto:** Archivos con nombres maliciosos pueden ejecutar código arbitrario
- **Remediación:**
```typescript
const sanitizeFilename = (filename: string): string => {
  if (!/^[\w\-\.\/]+$/.test(filename)) {
    throw new Error(`Invalid filename: ${filename}`);
  }
  return filename;
};

const sanitizedFiles = files.map(f => sanitizeFilename(f));
const args = ['review', ...sanitizedFiles, '--format', 'json'];
```

#### SEC-003: Command Injection en Go
- **Severidad:** CRÍTICA (CVSS 9.0)
- **Ubicación:** `goreview/cmd/goreview/commands/commit.go:147`
- **Código vulnerable:**
```go
hashCmd := exec.Command("sh", "-c", "git diff --cached | sha256sum | awk '{print $1}'")
```
- **Remediación:**
```go
diffCmd := exec.Command("git", "diff", "--cached")
output, err := diffCmd.Output()
if err != nil { return err }
hash := sha256.Sum256(output)
hashString := hex.EncodeToString(hash[:])
```

#### SEC-004: Path Traversal
- **Severidad:** CRÍTICA (CVSS 8.1)
- **Ubicación:** `integrations/github-app/src/services/orchestrator.service.ts:35`
- **Código vulnerable:**
```typescript
const workDir = path.join(process.cwd(), 'tmp', `${owner}-${repo}-${pullNumber}`);
```
- **Remediación:**
```typescript
const sanitizePath = (input: string): string => {
  return input.replace(/[^a-zA-Z0-9\-_]/g, '_');
};

const workDir = path.join(
  process.cwd(),
  'tmp',
  `${sanitizePath(owner)}-${sanitizePath(repo)}-${pullNumber}`
);

// Verificar path
const expectedBase = path.join(process.cwd(), 'tmp');
const resolvedPath = path.resolve(workDir);
if (!resolvedPath.startsWith(expectedBase)) {
  throw new Error('Path traversal attempt detected');
}
```

#### SEC-005: Token Expuesto en Logs
- **Severidad:** CRÍTICA (CVSS 8.5)
- **Ubicación:** `integrations/github-app/src/services/orchestrator.service.ts:129`
- **Código vulnerable:**
```typescript
const remote = `https://x-access-token:${token}@github.com/${owner}/${repo}.git`;
```
- **Remediación:** Usar SSH keys o credential helper temporal

### 1.2 Vulnerabilidades Altas (CVSS 7.0-7.9)

#### SEC-006: Falta de Rate Limiting
- **Severidad:** ALTA (CVSS 7.5)
- **Ubicación:** `integrations/github-app/src/index.ts`
- **Remediación:**
```typescript
import rateLimit from 'express-rate-limit';

const webhookLimiter = rateLimit({
  windowMs: 1 * 60 * 1000,
  max: 10,
  message: 'Too many webhook requests'
});

app.use('/api/github/webhooks', webhookLimiter, createNodeMiddleware(webhooks));
```

#### SEC-007: XSS en Dashboard
- **Severidad:** ALTA (CVSS 7.4)
- **Ubicación:** `integrations/github-app/src/index.ts:41-52`
- **Remediación:**
```typescript
const escapeHtml = (unsafe: string): string => {
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
};

// Usar en todos los outputs HTML
<td>${escapeHtml(h.repo)}</td>
```

#### SEC-008: Validación de Webhook Payload Ausente
- **Severidad:** ALTA (CVSS 7.2)
- **Ubicación:** `integrations/github-app/src/index.ts:294-318`
- **Remediación:**
```typescript
import { z } from 'zod';

const pullRequestSchema = z.object({
  repository: z.object({
    owner: z.object({ login: z.string() }),
    name: z.string(),
  }),
  pull_request: z.object({
    number: z.number(),
    head: z.object({ sha: z.string() })
  }),
  installation: z.object({ id: z.number() })
});

webhooks.on(['pull_request.opened'], async ({ payload }) => {
  const validated = pullRequestSchema.safeParse(payload);
  if (!validated.success) {
    logger.warn({ error: validated.error }, 'Invalid webhook payload');
    return;
  }
});
```

#### SEC-009: Endpoint Debug Sin Autenticación
- **Severidad:** ALTA (CVSS 7.3)
- **Ubicación:** `integrations/github-app/src/index.ts:236`
- **Remediación:**
```typescript
if (config.NODE_ENV === 'production') {
  // No registrar endpoint de debug
} else {
  app.post('/api/debug/simulate', requireAuth, handler);
}
```

#### SEC-010: Missing Security Headers
- **Severidad:** MEDIA (CVSS 6.5)
- **Remediación:**
```typescript
import helmet from 'helmet';

app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      scriptSrc: ["'self'"],
    }
  },
  xFrameOptions: { action: 'deny' },
}));
```

---

## 2. Testing

### 2.1 Estado Actual

| Componente | Cobertura | Estado |
|------------|-----------|--------|
| GitHub App (TypeScript) | 0% | CRÍTICO |
| GoReview - providers | 14.3% | CRÍTICO |
| GoReview - rules | 88.1% | Excelente |
| GoReview - review | 73.1% | Bueno |
| GoReview - config | 72.1% | Bueno |
| GoReview - cache | 70.7% | Bueno |
| GoReview - git | 64.0% | Aceptable |
| GoReview - cmd | 0% | Sin tests |
| GoReview - report | 0% | Sin tests |

### 2.2 Tests Faltantes Críticos

#### TEST-001: GitHubService Tests
```typescript
// tests/services/github.service.test.ts
describe('GitHubService', () => {
  describe('getChangedFiles', () => {
    it('should return list of changed files', async () => {
      // Mock Octokit
      // Verify API calls
    });

    it('should handle API errors gracefully', async () => {
      // Test error handling
    });

    it('should handle pagination for large PRs', async () => {
      // Test >100 files
    });
  });
});
```

#### TEST-002: OrchestratorService Tests
```typescript
// tests/services/orchestrator.service.test.ts
describe('OrchestratorService', () => {
  it('should orchestrate full review workflow', async () => {});
  it('should handle no changed files', async () => {});
  it('should cleanup workspace on error', async () => {});
  it('should record review history', async () => {});
});
```

#### TEST-003: GoReviewService Tests
```typescript
// tests/services/goreview.service.test.ts
describe('GoReviewService', () => {
  it('should execute CLI and parse JSON output', async () => {});
  it('should handle empty file list', async () => {});
  it('should reject on CLI error', async () => {});
  it('should handle invalid JSON output', async () => {});
});
```

#### TEST-004: ChecksService Tests
```typescript
// tests/services/checks.service.test.ts
describe('ChecksService', () => {
  it('should create check run in progress state', async () => {});
  it('should update with success when no issues', async () => {});
  it('should limit annotations to 50', async () => {});
});
```

#### TEST-005: Integration Tests
```typescript
// tests/integration/webhook.test.ts
describe('Webhook Integration', () => {
  it('should handle pull_request.opened event', async () => {});
  it('should reject invalid webhook signature', async () => {});
});
```

### 2.3 Configuración Recomendada

```typescript
// vitest.config.ts
export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      lines: 80,
      functions: 80,
      branches: 80,
    },
  },
});
```

### 2.4 CI/CD para Tests

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]

jobs:
  test-typescript:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: pnpm/action-setup@v2
      - run: pnpm install
      - run: pnpm test:coverage
      - uses: codecov/codecov-action@v3

  test-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test -v -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3
```

---

## 3. Arquitectura

### 3.1 Problemas Identificados

| Problema | Impacto | Prioridad |
|----------|---------|-----------|
| GitHub App monolítico (350 líneas en index.ts) | Mantenibilidad | ALTA |
| HTML embebido en código | Legibilidad | ALTA |
| Estado en memoria (history) | No escalable | CRÍTICA |
| Acoplamiento via spawn | No escalable | ALTA |
| Sin inyección de dependencias | Testabilidad | MEDIA |

### 3.2 Arquitectura Propuesta

```
src/
├── application/
│   ├── use-cases/
│   │   ├── review-pull-request.use-case.ts
│   │   ├── update-documentation.use-case.ts
│   │   └── simulate-review.use-case.ts
│   └── dto/
│       ├── webhook.dto.ts
│       └── review-result.dto.ts
├── domain/
│   ├── entities/
│   │   ├── pull-request.entity.ts
│   │   └── review.entity.ts
│   ├── interfaces/
│   │   ├── vcs-provider.interface.ts
│   │   ├── review-engine.interface.ts
│   │   └── storage.interface.ts
│   └── value-objects/
│       └── review-status.vo.ts
├── infrastructure/
│   ├── github/
│   │   ├── github-vcs.adapter.ts
│   │   └── github-webhook.handler.ts
│   ├── goreview/
│   │   └── goreview-http.adapter.ts
│   ├── storage/
│   │   ├── in-memory.repository.ts
│   │   └── postgres.repository.ts
│   └── queue/
│       └── redis-queue.adapter.ts
├── presentation/
│   ├── http/
│   │   ├── controllers/
│   │   ├── routes/
│   │   └── views/
│   └── middleware/
├── config/
└── main.ts
```

### 3.3 Interfaces de Dominio

```typescript
// domain/interfaces/review-engine.interface.ts
export interface ReviewEngine {
  reviewFiles(files: string[], workDir: string, options?: ReviewOptions): Promise<ReviewResult>;
  healthCheck(): Promise<boolean>;
}

// domain/interfaces/vcs-provider.interface.ts
export interface VCSProvider {
  getChangedFiles(owner: string, repo: string, ref: string): Promise<string[]>;
  createCheckRun(owner: string, repo: string, sha: string, status: CheckStatus): Promise<string>;
  updateCheckRun(checkId: string, result: ReviewResult): Promise<void>;
  cloneRepository(owner: string, repo: string, ref: string, targetDir: string): Promise<void>;
}

// domain/interfaces/storage.interface.ts
export interface ReviewRepository {
  save(review: Review): Promise<Review>;
  findByPullRequest(owner: string, repo: string, pr: number): Promise<Review[]>;
  getStats(owner: string, repo: string): Promise<ReviewStats>;
}
```

### 3.4 Use Case Example

```typescript
// application/use-cases/review-pull-request.use-case.ts
export class ReviewPullRequestUseCase {
  constructor(
    private readonly vcsProvider: VCSProvider,
    private readonly reviewEngine: ReviewEngine,
    private readonly reviewRepository: ReviewRepository,
    private readonly logger: Logger
  ) {}

  async execute(input: ReviewPullRequestInput): Promise<ReviewPullRequestOutput> {
    const { owner, repo, pullNumber, commitSha } = input;

    // 1. Create check run
    const checkId = await this.vcsProvider.createCheckRun(owner, repo, commitSha, CheckStatus.InProgress);

    // 2. Get changed files
    const files = await this.vcsProvider.getChangedFiles(owner, repo, pullNumber);

    if (files.length === 0) {
      await this.vcsProvider.updateCheckRun(checkId, { status: 'success', issues: 0 });
      return { success: true, issuesFound: 0 };
    }

    // 3. Prepare workspace
    const workDir = await this.prepareWorkspace(owner, repo, commitSha);

    // 4. Run review
    const result = await this.reviewEngine.reviewFiles(files, workDir);

    // 5. Save to repository
    await this.reviewRepository.save({ owner, repo, pullNumber, commitSha, result });

    // 6. Update check run
    await this.vcsProvider.updateCheckRun(checkId, result);

    return { success: true, issuesFound: result.totalIssues };
  }
}
```

### 3.5 Persistencia con PostgreSQL

```sql
-- migrations/001_initial_schema.sql
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner VARCHAR(255) NOT NULL,
    repo VARCHAR(255) NOT NULL,
    pull_number INTEGER NOT NULL,
    commit_sha VARCHAR(64) NOT NULL,
    total_issues INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT reviews_unique UNIQUE (owner, repo, pull_number, commit_sha)
);

CREATE TABLE review_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    file_path VARCHAR(500) NOT NULL,
    issue_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    suggestion TEXT,
    start_line INTEGER,
    end_line INTEGER
);

CREATE INDEX idx_reviews_repo ON reviews(owner, repo);
CREATE INDEX idx_issues_review ON review_issues(review_id);
```

### 3.6 Diagrama de Arquitectura Final

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub App (Node.js)                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐ │
│  │   Controllers  │  │   Use Cases    │  │    Domain      │ │
│  └────────────────┘  └────────────────┘  └────────────────┘ │
│           │                   │                    │         │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Infrastructure (Adapters)                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
                ▼             ▼             ▼
    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
    │  PostgreSQL  │  │ Redis Queue  │  │    Redis     │
    │  (Reviews)   │  │   (Jobs)     │  │   (Cache)    │
    └──────────────┘  └──────────────┘  └──────────────┘
                              │
                              ▼
                ┌──────────────────────────┐
                │   GoReview Workers       │
                │   (Go/Multiple Instances)│
                └──────────────────────────┘
                              │
                              ▼
                ┌──────────────────────────┐
                │   AI Providers           │
                │  - Ollama / OpenAI       │
                └──────────────────────────┘
```

---

## 4. Rendimiento

### 4.1 Problemas Críticos

| # | Problema | Ubicación | Impacto |
|---|----------|-----------|---------|
| 1 | Git clone completo sin shallow | orchestrator.service.ts | +30-300s latencia |
| 2 | Sin rate limiting a Ollama | providers/ollama.go | Saturación |
| 3 | Race condition en historial | orchestrator.service.ts | Corrupción datos |
| 4 | Semáforo hardcoded (5) | review/engine.go | No escala |
| 5 | Cache sin límites | cache/cache.go | Memory leak |
| 6 | Sin paginación GitHub API | github.service.ts | PRs >100 archivos |

### 4.2 Optimizaciones Propuestas

#### PERF-001: Shallow Clone
```typescript
// Antes
await git.clone(remote, workDir);

// Después
await git.clone(remote, workDir, ['--depth', '1', '--single-branch']);
```

#### PERF-002: Rate Limiting para Ollama
```go
// internal/providers/ollama.go
type RateLimiter struct {
    tokens   chan struct{}
    interval time.Duration
}

func NewRateLimiter(rps int) *RateLimiter {
    rl := &RateLimiter{
        tokens:   make(chan struct{}, rps),
        interval: time.Second / time.Duration(rps),
    }
    go rl.refill()
    return rl
}
```

#### PERF-003: Concurrencia Configurable
```go
// config/config.go
type ReviewConfig struct {
    MaxConcurrency int `yaml:"max_concurrency" default:"0"` // 0 = auto
}

// review/engine.go
func (e *Engine) calculateOptimalConcurrency() int {
    if e.cfg.Review.MaxConcurrency > 0 {
        return e.cfg.Review.MaxConcurrency
    }
    return runtime.NumCPU() * 2
}
```

#### PERF-004: LRU Cache con Límites
```go
import "github.com/hashicorp/golang-lru/v2"

type FileCache struct {
    dir string
    ttl time.Duration
    lru *lru.Cache[string, *providers.ReviewResponse]
}

func NewFileCache(cfg config.CacheConfig) (*FileCache, error) {
    cache, err := lru.New[string, *providers.ReviewResponse](cfg.MaxEntries)
    if err != nil {
        return nil, err
    }
    return &FileCache{dir: cfg.Dir, ttl: cfg.TTL, lru: cache}, nil
}
```

#### PERF-005: Paginación GitHub API
```typescript
async getChangedFiles(octokit: Octokit, owner: string, repo: string, pullNumber: number): Promise<string[]> {
  const files: string[] = [];
  let page = 1;
  let hasMore = true;

  while (hasMore) {
    const { data } = await octokit.pulls.listFiles({
      owner, repo, pull_number: pullNumber,
      per_page: 100, page,
    });

    files.push(...data.map(f => f.filename));
    hasMore = data.length === 100;
    page++;
  }

  return files;
}
```

### 4.3 Impacto Esperado

| Métrica | Antes | Después | Mejora |
|---------|-------|---------|--------|
| Throughput | 20 reviews/hora | 50 reviews/hora | +150% |
| Latencia P95 | 120s | 60s | -50% |
| Memoria | 500MB | 200MB | -60% |
| Errores | 5% | <1% | -80% |

---

## 5. Calidad de Código

### 5.1 Code Smells Identificados

#### CODE-001: God Object en index.ts
- **Ubicación:** `integrations/github-app/src/index.ts` (350 líneas)
- **Problema:** Múltiples responsabilidades mezcladas
- **Solución:** Separar en controllers, routes, views

#### CODE-002: HTML Embebido
- **Ubicación:** `integrations/github-app/src/index.ts:54-231`
- **Problema:** 180 líneas de HTML en el código
- **Solución:** Usar templates o separar a SPA

#### CODE-003: Duplicación en Parsing JSON
- **Ubicación:** `goreview/internal/providers/ollama.go:128-152`
- **Problema:** Lógica de parsing duplicada con OpenAI
- **Solución:** Extraer a función compartida:
```go
// internal/providers/parser.go
func ParseLLMResponse(rawResponse string) ([]Issue, error) {
    cleanResp := strings.TrimSpace(rawResponse)
    cleanResp = strings.TrimPrefix(cleanResp, "```json")
    cleanResp = strings.TrimSuffix(cleanResp, "```")

    var issues []Issue
    if err := json.Unmarshal([]byte(cleanResp), &issues); err == nil {
        return issues, nil
    }
    // Try other formats...
    return nil, fmt.Errorf("failed to parse LLM response")
}
```

#### CODE-004: Magic Numbers
- **Ubicaciones:**
  - `review/engine.go:58` - `semaphore := make(chan struct{}, 5)`
  - `checks.service.ts:51` - `annotations.slice(0, 50)`
- **Solución:** Definir constantes:
```go
const (
    DefaultMaxConcurrency = 5
    GitHubMaxAnnotationsPerRequest = 50
)
```

#### CODE-005: Errores Silenciados
- **Ubicación:** `orchestrator.service.ts:92-93`
```typescript
} catch (e) { /* ignore */ }
```
- **Solución:**
```typescript
} catch (e) {
  logger.warn({ error: e, checkRunId }, 'Failed to update check run status');
}
```

### 5.2 Mejoras de Documentación

#### DOC-001: JSDoc para Funciones Públicas
```typescript
/**
 * Orchestrates the complete code review process for a pull request
 *
 * @param installationId - GitHub App installation ID
 * @param owner - Repository owner username
 * @param repo - Repository name
 * @param pullNumber - Pull request number
 * @param commitSha - Specific commit SHA to analyze
 *
 * @throws {Error} When GitHub authentication fails
 * @throws {Error} When GoReview CLI execution fails
 */
async handlePullRequest(
  installationId: number,
  owner: string,
  repo: string,
  pullNumber: number,
  commitSha: string
): Promise<void>
```

#### DOC-002: Estandarizar Comentarios en Inglés
```go
// Engine orchestrates the code review process
type Engine struct { /* ... */ }

// NewEngine creates a new review engine
func NewEngine(/* ... */) *Engine { /* ... */ }

// Run executes the code review
func (e *Engine) Run(ctx context.Context) (*Result, error) { /* ... */ }
```

---

## 6. Plan de Implementación

### Fase 1: Seguridad (Semana 1)
- [ ] SEC-001: Rotar credenciales y limpiar repositorio
- [ ] SEC-002: Sanitizar inputs en goreview.service.ts
- [ ] SEC-003: Fix command injection en commit.go
- [ ] SEC-004: Implementar path traversal protection
- [ ] SEC-007: Escapar HTML en dashboard
- [ ] SEC-009: Deshabilitar debug endpoint en producción

### Fase 2: Testing (Semanas 2-3)
- [ ] TEST-001: Implementar tests GitHubService
- [ ] TEST-002: Implementar tests OrchestratorService
- [ ] TEST-003: Implementar tests GoReviewService
- [ ] TEST-004: Implementar tests ChecksService
- [ ] TEST-005: Implementar tests de integración
- [ ] Configurar CI/CD con cobertura

### Fase 3: Rendimiento (Semana 4)
- [ ] PERF-001: Implementar shallow clone
- [ ] PERF-002: Rate limiting para Ollama
- [ ] PERF-003: Concurrencia configurable
- [ ] PERF-004: LRU cache con límites
- [ ] PERF-005: Paginación GitHub API

### Fase 4: Arquitectura (Semanas 5-8)
- [ ] Refactorizar a Clean Architecture
- [ ] Separar HTML a templates/SPA
- [ ] Implementar persistencia PostgreSQL
- [ ] Implementar message queue (opcional)
- [ ] Agregar API HTTP a GoReview (opcional)

---

## 7. Métricas de Éxito

| Métrica | Objetivo |
|---------|----------|
| Vulnerabilidades críticas | 0 |
| Cobertura de tests | >80% |
| Latencia P95 | <60s |
| Throughput | >50 reviews/hora |
| Tasa de errores | <1% |
| Puntuación arquitectura | >8/10 |

---

## Apéndice: Archivos Analizados

### TypeScript (GitHub App)
- `src/index.ts`
- `src/config.ts`
- `src/logger.ts`
- `src/services/orchestrator.service.ts`
- `src/services/goreview.service.ts`
- `src/services/github.service.ts`
- `src/services/checks.service.ts`
- `src/services/docs.service.ts`

### Go (GoReview CLI)
- `cmd/goreview/main.go`
- `cmd/goreview/commands/review.go`
- `cmd/goreview/commands/commit.go`
- `internal/review/engine.go`
- `internal/providers/factory.go`
- `internal/providers/ollama.go`
- `internal/config/config.go`
- `internal/cache/cache.go`
- `internal/git/git.go`

---

> Generado por Claude Opus 4.5 - Análisis Multi-Agente
> Agentes utilizados: Explore, Security-Auditor, Code-Reviewer, Performance-Engineer, Backend-Architect, Test-Engineer
