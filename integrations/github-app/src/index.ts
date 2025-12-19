import express, { Express, Request, Response, NextFunction } from 'express';
import { Webhooks, createNodeMiddleware } from '@octokit/webhooks';
import helmet from 'helmet';
import rateLimit from 'express-rate-limit';
import { z } from 'zod';
import { config } from './config.js';
import { logger } from './logger.js';
import { orchestratorService } from './services/orchestrator.service.js';
import { docsService } from './services/docs.service.js';
import { githubService } from './services/github.service.js';

// Webhook payload validation schemas
const pullRequestPayloadSchema = z.object({
  repository: z.object({
    owner: z.object({ login: z.string() }),
    name: z.string(),
    full_name: z.string(),
  }),
  pull_request: z.object({
    number: z.number(),
    head: z.object({ sha: z.string() }),
  }),
  installation: z.object({ id: z.number() }).optional(),
  action: z.string(),
});

const pushPayloadSchema = z.object({
  repository: z.object({
    owner: z.object({ login: z.string() }).nullable(),
    name: z.string(),
    full_name: z.string(),
  }),
  ref: z.string(),
  commits: z.array(z.any()),
  installation: z.object({ id: z.number() }).optional(),
});

/**
 * Escapes HTML entities to prevent XSS attacks.
 */
function escapeHtml(unsafe: string | number | undefined | null): string {
  if (unsafe === undefined || unsafe === null) return '';
  const str = String(unsafe);
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

const app: Express = express();

// SEC-010: Security headers with helmet
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      scriptSrc: ["'self'", "'unsafe-inline'"], // Needed for inline scripts in dashboard
      styleSrc: ["'self'", "'unsafe-inline'"], // Needed for inline styles in dashboard
      imgSrc: ["'self'", "data:"],
    },
  },
  crossOriginEmbedderPolicy: false,
}));

// Inicializar Webhooks handler
const webhooks = new Webhooks({
  secret: config.GITHUB_WEBHOOK_SECRET,
});

// SEC-006: Rate limiting for webhooks
const webhookLimiter = rateLimit({
  windowMs: 1 * 60 * 1000, // 1 minute
  max: 30, // Limit each IP to 30 requests per minute
  message: { error: 'Too many webhook requests, please try again later' },
  standardHeaders: true,
  legacyHeaders: false,
});

// Middleware para loggear requests
app.use((req, _res, next) => {
  if (req.path !== '/health') {
    logger.info({ method: req.method, path: req.path }, 'Incoming request');
  }
  next();
});

// GitHub Webhooks Endpoint with rate limiting
// createNodeMiddleware maneja la verificacion de firma automaticamente
app.use('/api/github/webhooks', webhookLimiter, createNodeMiddleware(webhooks, { path: '/' }));

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
  
  // SEC-007: XSS prevention - escape all user-controlled data
  const historyRows = history.map((h, index) => `
    <tr>
        <td>${escapeHtml(new Date(h.timestamp).toLocaleTimeString())}</td>
        <td>${escapeHtml(h.repo)}</td>
        <td>#${escapeHtml(h.pr)}</td>
        <td><code>${escapeHtml(h.commit)}</code></td>
        <td><strong>${escapeHtml(h.issues)}</strong></td>
        <td>${escapeHtml(h.duration)}s</td>
        <td><span class="status ${h.status === 'success' ? 'ok' : 'error'}">${escapeHtml(h.status.toUpperCase())}</span></td>
        <td><button onclick="showDetails(${index})" class="btn btn-sm">View Details</button></td>
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
        .btn-sm { padding: 0.25rem 0.5rem; font-size: 0.75rem; }
        
        dialog { padding: 0; border: none; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.15); max-width: 800px; width: 90%; }
        dialog::backdrop { background: rgba(0,0,0,0.5); }
        .modal-header { padding: 1rem; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; align-items: center; background: #f9fafb; }
        .modal-body { padding: 1rem; max-height: 70vh; overflow-y: auto; }
        .issue-card { border: 1px solid #e5e7eb; border-radius: 6px; padding: 1rem; margin-bottom: 1rem; background: #fff; }
        .issue-header { display: flex; justify-content: space-between; margin-bottom: 0.5rem; }
        .severity { text-transform: uppercase; font-size: 0.7rem; font-weight: bold; padding: 2px 6px; border-radius: 4px; }
        .sev-critical { background: #fee2e2; color: #991b1b; }
        .sev-warning { background: #fef3c7; color: #92400e; }
        .sev-info { background: #dbeafe; color: #1e40af; }
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
                    <th>Action</th>
                </tr>
            </thead>
            <tbody>
                ${historyRows}
            </tbody>
        </table>
        `}
    </div>

    <dialog id="detailsModal">
        <div class="modal-header">
            <h3 style="margin:0">Review Details</h3>
            <button onclick="closeModal()" style="border:none; background:none; cursor:pointer; font-size:1.5rem;">&times;</button>
        </div>
        <div id="modalContent" class="modal-body"></div>
    </dialog>

    <script>
        // Inject history data safely
        const reviewData = ${JSON.stringify(history).replace(/</g, '\\u003c')};

        function showDetails(index) {
            const record = reviewData[index];
            const modal = document.getElementById('detailsModal');
            const content = document.getElementById('modalContent');
            
            if (!record.details || !record.details.files) {
                content.innerHTML = '<p>No detailed results available for this review.</p>';
                modal.showModal();
                return;
            }

            let html = '';
            
            record.details.files.forEach(file => {
                if (file.response && file.response.issues && file.response.issues.length > 0) {
                    html += '<h4>📄 ' + escapeHtml(file.file) + '</h4>';
                    
                    file.response.issues.forEach(issue => {
                        const sevClass = 'sev-' + (issue.severity || 'info').toLowerCase();
                        html += '<div class="issue-card">';
                        html += '<div class="issue-header">';
                        html += '<strong>' + escapeHtml(issue.type) + '</strong>';
                        html += '<span class="severity ' + sevClass + '">' + escapeHtml(issue.severity) + '</span>';
                        html += '</div>';
                        html += '<p>' + escapeHtml(issue.message) + '</p>';
                        
                        if (issue.suggestion) {
                            html += '<div style="background:#f8fafc; padding:0.5rem; border-radius:4px; font-size:0.9em;">';
                            html += '<strong>💡 Suggestion:</strong><br>' + escapeHtml(issue.suggestion);
                            html += '</div>';
                        }
                        
                        if (issue.location) {
                            html += '<p style="font-size:0.8rem; color:#6b7280; margin-top:0.5rem;">Line: ' + issue.location.start_line + '</p>';
                        }
                        html += '</div>';
                    });
                }
            });

            if (html === '') {
                html = '<p>✅ No issues found in analyzed files.</p>';
            }

            content.innerHTML = html;
            modal.showModal();
        }

        function closeModal() {
            document.getElementById('detailsModal').close();
        }

        function escapeHtml(unsafe) {
            if (!unsafe) return '';
            return unsafe
                 .replace(/&/g, "&amp;")
                 .replace(/</g, "&lt;")
                 .replace(/>/g, "&gt;")
                 .replace(/"/g, "&quot;")
                 .replace(/'/g, "&#039;");
        }

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

// SEC-009: Debug endpoint disabled in production
if (config.NODE_ENV !== 'production') {
  // Debug endpoint to simulate a review (only available in development)
  app.post('/api/debug/simulate', (_req, res) => {
    const mockIssues = [
      {
        id: "SEC-001",
        type: "security",
        severity: "critical",
        message: "Potential SQL Injection identified in query construction.",
        suggestion: "Use parameterized queries instead of string concatenation.",
        location: { file: "src/database.go", start_line: 42 }
      },
      {
        id: "CQ-005",
        type: "quality",
        severity: "warning",
        message: "Function complexity is too high (cyclomatic complexity > 10).",
        suggestion: "Refactor the function into smaller, more manageable pieces.",
        location: { file: "src/utils.py", start_line: 15 }
      }
    ];

    const mockResult = {
      total_issues: 2,
      duration: 1.5,
      files: [
        {
          file: "src/database.go",
          response: { issues: [mockIssues[0]] },
          cached: false
        },
        {
          file: "src/utils.py",
          response: { issues: [mockIssues[1]] },
          cached: true
        }
      ]
    };

    (orchestratorService as any).history.push({
      timestamp: new Date().toISOString(),
      repo: 'JNZader/ai-toolkit',
      pr: Math.floor(Math.random() * 100) + 1,
      commit: Math.random().toString(36).substring(7),
      issues: 2,
      duration: 1.5,
      status: 'success',
      details: mockResult
    });

    res.json({ status: 'ok' });
  });
}

// Error handling
app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
  logger.error(err, 'Unhandled error');
  res.status(500).json({ error: 'Internal Server Error' });
});

// Event listeners with SEC-008: Payload validation
webhooks.on(['pull_request.opened', 'pull_request.synchronize'], async ({ payload }) => {
  // Validate webhook payload structure
  const validated = pullRequestPayloadSchema.safeParse(payload);
  if (!validated.success) {
    logger.warn({ error: validated.error.issues }, 'Invalid pull_request webhook payload');
    return;
  }

  const { repository, pull_request, installation, action } = validated.data;

  if (!installation) {
    logger.warn('No installation ID found in payload');
    return;
  }

  logger.info({
    repo: repository.full_name,
    pr: pull_request.number,
    action: action
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

webhooks.on('push', async ({ payload }) => {
  // Validate webhook payload structure
  const validated = pushPayloadSchema.safeParse(payload);
  if (!validated.success) {
    logger.warn({ error: validated.error.issues }, 'Invalid push webhook payload');
    return;
  }

  const { repository, ref, commits, installation } = validated.data;

  if (!installation) return;

  // Only process pushes to main/develop branches to avoid noise
  if (!ref.includes('main') && !ref.includes('develop')) return;

  logger.info({ repo: repository.full_name, ref }, 'Push received');

  // Authenticate
  const octokit = await githubService.getInstallationClient(installation.id);

  // Trigger doc update
  docsService.handlePush(
    octokit,
    repository.owner?.login || '',
    repository.name,
    ref,
    commits
  ).catch(err => logger.error(err, 'Docs update failed'));
});

// Iniciar servidor
app.listen(config.PORT, () => {
  logger.info(
    { port: config.PORT, env: config.NODE_ENV },
    'GitHub App server started'
  );
});

export { app, webhooks };