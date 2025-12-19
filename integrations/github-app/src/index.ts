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
  const history = orchestratorService.getHistory();
  
  const historyRows = history.map(h => `
    <tr>
        <td>${new Date(h.timestamp).toLocaleTimeString()}</td>
        <td>${h.repo}</td>
        <td>#${h.pr}</td>
        <td><code>${h.commit}</code></td>
        <td><strong>${h.issues}</strong></td>
        <td>${h.duration}s</td>
        <td><span class="status ${h.status === 'success' ? 'ok' : 'error'}">${h.status.toUpperCase()}</span></td>
    </tr>
  `).join('');

  const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoReview Dashboard</title>
    <style>
        body { font-family: -apple-system, system-ui, sans-serif; max-width: 1000px; margin: 0 auto; padding: 2rem; background: #f4f4f5; color: #1f2937; }
        .card { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 1rem; }
        .status { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-weight: bold; font-size: 0.75rem; }
        .status.ok { background: #dcfce7; color: #166534; }
        .status.error { background: #fee2e2; color: #991b1b; }
        h1 { margin-top: 0; }
        code { background: #f3f4f6; padding: 0.2rem 0.4rem; border-radius: 4px; font-family: monospace; }
        table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
        th, td { text-align: left; padding: 0.75rem; border-bottom: 1px solid #e5e7eb; }
        th { background: #f9fafb; font-weight: 600; font-size: 0.875rem; text-transform: uppercase; color: #6b7280; }
        tr:last-child td { border-bottom: none; }
        .btn { padding: 0.5rem 1rem; cursor: pointer; background: #fff; border: 1px solid #d1d5db; border-radius: 4px; font-size: 0.875rem; }
        .btn:hover { background: #f9fafb; }
        .btn-primary { background: #2563eb; color: white; border-color: #1d4ed8; }
        .btn-primary:hover { background: #1d4ed8; }
    </style>
</head>
<body>
    <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center;">
            <h1>🚀 GoReview AI Toolkit</h1>
            <div>
                <span class="status ok">OPERATIONAL</span>
                <span style="margin-left: 10px; font-size: 0.875rem; color: #6b7280;">Env: ${config.NODE_ENV}</span>
            </div>
        </div>
    </div>

    <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom: 1rem;">
            <h2>🤖 Configuration</h2>
            <button onclick="simulateReview()" class="btn btn-primary">🧪 Simulate Review</button>
        </div>
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
            <div><strong>Provider Host:</strong> <br><code>${config.OLLAMA_HOST}</code></div>
            <div><strong>Model:</strong> <br><code>${config.OLLAMA_MODEL}</code></div>
            <div><strong>Port:</strong> <br><code>${config.PORT}</code></div>
        </div>
    </div>

    <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center;">
            <h2>📊 Review History (Session)</h2>
            <button onclick="window.location.reload()" class="btn">🔄 Refresh</button>
        </div>
        
        ${history.length === 0 ? '<p style="color:#6b7280; font-style:italic;">No reviews processed in this session yet.</p>' : `
        <table>
            <thead>
                <tr>
                    <th>Time</th>
                    <th>Repo</th>
                    <th>PR</th>
                    <th>Commit</th>
                    <th>Issues</th>
                    <th>Duration</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>
                ${historyRows}
            </tbody>
        </table>
        `}
    </div>

    <script>
        async function simulateReview() {
            const btn = document.querySelector('.btn-primary');
            btn.disabled = true;
            btn.innerText = 'Simulating...';
            try {
                await fetch('/api/debug/simulate', { method: 'POST' });
                window.location.reload();
            } catch (err) {
                alert('Simulation failed');
                btn.disabled = false;
                btn.innerText = '🧪 Simulate Review';
            }
        }
    </script>
</body>
</html>
  `;
  res.send(html);
});

// Debug endpoint to simulate a review
app.post('/api/debug/simulate', (_req, res) => {
  // Access private history through any means or add a public method
  // For now, I'll add a mock entry directly if I had access, but 
  // better to use the orchestrator instance.
  
  // We'll use a hacky way since history is private, or better, 
  // I should have made a method in the service.
  // I will add a 'addMockRecord' method to OrchestratorService.
  (orchestratorService as any).history.push({
    timestamp: new Date().toISOString(),
    repo: 'JNZader/ai-toolkit',
    pr: Math.floor(Math.random() * 100) + 1,
    commit: Math.random().toString(36).substring(7),
    issues: Math.floor(Math.random() * 5),
    duration: (Math.random() * 5).toFixed(2),
    status: 'success'
  });
  
  res.json({ status: 'ok' });
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
app.listen(config.PORT, () => {
  logger.info(
    { port: config.PORT, env: config.NODE_ENV },
    'GitHub App server started'
  );
});

export { app, webhooks };