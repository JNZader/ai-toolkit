import express, { Express } from 'express';
import pino from 'pino';

const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
});

const app: Express = express();
const port = process.env.PORT || 3000;

app.use(express.json());

// Health check endpoint
app.get('/health', (_req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Webhook endpoint (placeholder)
app.post('/webhook', (req, res) => {
  logger.info({ event: req.headers['x-github-event'] }, 'Webhook received');
  res.status(200).send('OK');
});

app.listen(port, () => {
  logger.info({ port }, 'GitHub App server started');
});

export { app };
