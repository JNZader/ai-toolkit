import * as fs from 'fs/promises';
import * as path from 'path';
import { logger } from '../logger.js';
import { ReviewRecord } from './orchestrator.service.js';

const logFilePath = path.join(process.cwd(), 'logs', 'reviews.jsonl');

export class AuditService {
  constructor() {
    this.ensureLogFile();
  }

  private async ensureLogFile() {
    try {
      await fs.mkdir(path.dirname(logFilePath), { recursive: true });
    } catch (error) {
      logger.error({ error }, 'Failed to create log directory');
    }
  }

  async recordReview(data: ReviewRecord) {
    try {
      const logEntry = JSON.stringify(data) + '\n';
      await fs.appendFile(logFilePath, logEntry);
    } catch (error) {
      logger.error({ error, data }, 'Failed to write to audit log');
    }
  }
}

export const auditService = new AuditService();
