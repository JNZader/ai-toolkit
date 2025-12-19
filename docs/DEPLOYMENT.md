# Deployment Guide

## Prerequisites

- Docker and Docker Compose
- GitHub App registered
- Domain with HTTPS (recommended)
- Server with access to Ollama (local or remote)

## Deployment Steps

### 1. Configure Environment

Create a `.env` file in `integrations/github-app/` based on `.env.example`.

### 2. Prepare Private Key

Place your GitHub App private key in `integrations/github-app/config/private-key.pem`.

### 3. Build and Start

Using Docker Compose:

```bash
docker compose -f docker-compose.production.yml up -d --build
```

### 4. Verify

Check logs:
```bash
docker compose -f docker-compose.production.yml logs -f
```

## Updating

To update to the latest version:

```bash
git pull origin main
docker compose -f docker-compose.production.yml up -d --build
```
