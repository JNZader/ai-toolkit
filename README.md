# AI Toolkit

Suite de herramientas de IA para desarrollo de software.

## Componentes

| Componente | Descripcion | Estado |
|------------|-------------|--------|
| [goreview](./goreview) | CLI de code review con IA | En desarrollo |
| [goreview-rules](./goreview-rules) | Reglas de analisis | En desarrollo |
| [github-app](./integrations/github-app) | Integracion GitHub | En desarrollo |

## Quick Start

### Prerequisitos

- Go 1.23+
- Node.js 20+
- pnpm 9+
- Docker & Docker Compose

### Setup

```bash
# Clonar repositorio
git clone https://github.com/tu-usuario/ai-toolkit.git
cd ai-toolkit

# Setup completo
make setup

# Iniciar servicios Docker
make docker-up

# Descargar modelo de IA
make ollama-pull
```

### Desarrollo

```bash
# Correr tests
make test

# Correr linters
make lint

# Build
make build
```

## Estructura del Proyecto

```
ai-toolkit/
├── goreview/              # CLI principal
├── goreview-rules/        # Reglas YAML
├── integrations/
│   └── github-app/        # GitHub App
├── shared/                # Codigo compartido
├── docs/                  # Documentacion
└── scripts/               # Scripts utilitarios
```

## Documentacion

- [Guia de Desarrollo](./docs/solo-dev-guides/)
- [API Reference](./docs/API.md)
- [Troubleshooting](./docs/TROUBLESHOOTING.md)

## Licencia

MIT License