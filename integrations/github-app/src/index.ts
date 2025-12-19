import express, { Express, Request, Response, NextFunction } from 'express';
import { Webhooks, createNodeMiddleware } from '@octokit/webhooks';
import { config } from './config.js';
import { logger } from './logger.js';

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

// Error handling
app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
  logger.error(err, 'Unhandled error');
  res.status(500).json({ error: 'Internal Server Error' });
});

// Event listeners
webhooks.on('pull_request.opened', async ({ payload }) => {
  logger.info({ 
    repo: payload.repository.full_name,
    pr: payload.pull_request.number,
    action: 'opened' 
  }, 'PR Opened');
});

webhooks.on('pull_request.synchronize', async ({ payload }) => {
  logger.info({ 
    repo: payload.repository.full_name,
    pr: payload.pull_request.number,
    action: 'synchronize' 
  }, 'PR Synchronized');
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