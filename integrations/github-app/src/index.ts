import express, { Express, Request, Response, NextFunction } from 'express';
import { Webhooks, createNodeMiddleware } from '@octokit/webhooks';
import { config } from './config.js';
import { logger } from './logger.js';
import { orchestratorService } from './services/orchestrator.service.js';

const app: Express = express();

// Inicializar Webhooks handler
const webhooks = new Webhooks({
  secret: config.GITHUB_WEBHOOK_SECRET,
});

// Middleware para loggear requests
app.use((req, _res, next) => {
  if (req.path !== '/health') {
    logger.info({ method: req.method, path: req.path }, 'Incoming request');
  }
  next();
});

// GitHub Webhooks Endpoint
// createNodeMiddleware maneja la verificacion de firma automaticamente
app.use('/api/github/webhooks', createNodeMiddleware(webhooks, { path: '/' }));

// Health check
app.get('/health', (_req, res) => {
  res.json({ 
    status: 'ok', 
    env: config.NODE_ENV,
    timestamp: new Date().toISOString() 
  });
});

// Dashboard
app.get('/dashboard', (_req, res) => {
  const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoReview Dashboard</title>
    <style>
        body { font-family: -apple-system, system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; background: #f4f4f5; }
        .card { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 1rem; }
        .status { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-weight: bold; font-size: 0.875rem; }
        .status.ok { background: #dcfce7; color: #166534; }
        h1 { color: #1f2937; }
        code { background: #f3f4f6; padding: 0.2rem 0.4rem; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="card">
        <h1>🚀 GoReview AI Toolkit</h1>
        <p>Status: <span class="status ok">OPERATIONAL</span></p>
        <p>Environment: <code>${config.NODE_ENV}</code></p>
        <p>Server Time: ${new Date().toLocaleString()}</p>
    </div>
    <div class="card">
        <h2>🤖 AI Provider</h2>
        <p>Host: <code>${config.OLLAMA_HOST}</code></p>
        <p>Model: <code>${config.OLLAMA_MODEL}</code></p>
    </div>
    <div class="card">
        <h2>📊 Statistics</h2>
        <p>Reviews Processed: <strong>(Coming soon with Redis)</strong></p>
        <p><a href="/health">View JSON Health Check</a></p>
    </div>
</body>
</html>
  `;
  res.send(html);
});

// Error handling
app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
  logger.error(err, 'Unhandled error');
  res.status(500).json({ error: 'Internal Server Error' });
});

// Event listeners
webhooks.on(['pull_request.opened', 'pull_request.synchronize'], async ({ payload }) => {
  const { repository, pull_request, installation } = payload;
  
  if (!installation) {
    logger.warn('No installation ID found in payload');
    return;
  }

  logger.info({ 
    repo: repository.full_name,
    pr: pull_request.number,
    action: payload.action 
  }, 'Triggering review orchestration');

  // Trigger async review (fire and forget for webhook response speed)
  orchestratorService.handlePullRequest(
    installation.id,
    repository.owner.login,
    repository.name,
    pull_request.number,
    pull_request.head.sha
  ).catch(err => {
    logger.error({ err }, 'Orchestration error (async)');
  });
});

// Iniciar servidor
if (require.main === module) {
  app.listen(config.PORT, () => {
    logger.info(
      { port: config.PORT, env: config.NODE_ENV },
      'GitHub App server started'
    );
  });
}

export { app, webhooks };