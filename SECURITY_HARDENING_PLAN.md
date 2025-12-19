# Plan de Hardening de Seguridad - AI Toolkit

**Fecha:** 2025-12-19
**Fuente:** Análisis automático con GoReview (Ollama / qwen2.5-coder)

---

## Resumen de Issues Críticos y de Seguridad

Este documento resume los hallazgos más importantes del último análisis de `GoReview` y propone un plan de acción para solucionarlos.

---

### 🔴 Hallazgos Críticos

#### 1. **Command Injection en `goreview commit`**
- **Archivo:** `goreview/cmd/goreview/commands/commit.go`
- **Riesgo:** El uso de `exec.Command("sh", "-c", "...")` para calcular el hash del diff es vulnerable. Un nombre de archivo malicioso podría ejecutar código arbitrario.
- **Acción:** **(YA RESUELTO)** Se reemplazó con la implementación nativa en Go usando `crypto/sha256`.

#### 2. **Command Injection en `goreview.service.ts`**
- **Archivo:** `integrations/github-app/src/services/goreview.service.ts`
- **Riesgo:** Los nombres de archivo se pasan como argumentos al `spawn` del CLI `goreview`.
- **Acción:** **(YA RESUELTO)** Se implementó la función `sanitizeFilename` para validar los nombres de archivo antes de pasarlos.

#### 3. **Vulnerabilidad de Dependencia en `go.mod`**
- **Archivo:** `goreview/go.mod`
- **Riesgo:** La versión actual de `github.com/hashicorp/golang-lru` (`v2.0.7`) tiene una vulnerabilidad conocida (potencial memory leak).
- **Acción:** **Actualizar la dependencia.** Ejecutar `go get github.com/hashicorp/golang-lru/v2@latest` para obtener la última versión parcheada.

---

### 🟡 Hallazgos de Seguridad (Error / Warning)

#### 1. **Open Redirect en Logs**
- **Archivo:** `integrations/github-app/src/services/checks.service.ts`
- **Riesgo:** Loggear `owner` y `repo` directamente en un error podría, en teoría, usarse para ataques de Open Redirect si los logs son accesibles externamente.
- **Acción:** Considerar sanitizar o loggear solo IDs en lugar de nombres completos en mensajes de error públicos. Para logs internos, el riesgo es bajo.

#### 2. **XSS Potencial en Dashboard**
- **Archivo:** `integrations/github-app/src/index.ts`
- **Riesgo:** La directiva `unsafe-inline` en la Content Security Policy (CSP) debilita la protección contra XSS.
- **Acción:** **(YA RESUELTO)** Se implementó una función `escapeHtml` para sanitizar todos los datos antes de renderizarlos en el dashboard.

#### 3. **Validación de Payload en `push`**
- **Archivo:** `integrations/github-app/src/index.ts`
- **Riesgo:** El payload del evento `push` no se valida con Zod como se hace con el de `pull_request`.
- **Acción:** Crear un `pushPayloadSchema` con Zod y validar el payload entrante para asegurar la integridad de los datos.

---

### 🔵 Próximos Pasos Recomendados

1.  **(Prioridad Alta)** Actualizar la dependencia `golang-lru` para mitigar la vulnerabilidad.
2.  Implementar la validación Zod para los eventos `push`.
3.  Revisar y sanitizar los logs de errores para no exponer información sensible.
4.  Refinar el prompt de la IA para reducir falsos positivos en temas de estilo y mejorar la precisión en la detección de bugs.
