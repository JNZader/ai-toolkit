import { spawn } from 'child_process';
import { logger } from '../logger.js';

/**
 * Sanitizes a filename to prevent command injection attacks.
 * Only allows alphanumeric characters, underscores, hyphens, dots, and forward slashes.
 * @throws Error if filename contains invalid characters
 */
function sanitizeFilename(filename: string): string {
  // Allow only safe characters: alphanumeric, underscore, hyphen, dot, forward slash
  // This prevents shell metacharacters and path traversal attempts
  if (!/^[\w\-./]+$/.test(filename)) {
    throw new Error(`Invalid filename contains dangerous characters: ${filename}`);
  }

  // Prevent path traversal
  if (filename.includes('..')) {
    throw new Error(`Path traversal attempt detected: ${filename}`);
  }

  // Prevent absolute paths
  if (filename.startsWith('/') || /^[a-zA-Z]:/.test(filename)) {
    throw new Error(`Absolute path not allowed: ${filename}`);
  }

  return filename;
}

export interface GoReviewResult {
  total_issues: number;
  duration: number;
  files: Array<{
    file: string;
    response?: {
      issues: Array<{
        id: string;
        type: string;
        severity: string;
        message: string;
        suggestion?: string;
        location?: {
          file: string;
          start_line: number;
        };
      }>;
    };
    error?: string;
  }>;
}

export class GoReviewService {
  /**
   * Run GoReview CLI on a list of files
   */
  async runReview(files: string[], workDir: string): Promise<GoReviewResult> {
    return new Promise((resolve, reject) => {
      logger.info({ files, workDir }, 'Starting GoReview CLI');

      // Ensure we have files to review
      if (files.length === 0) {
        return resolve({ total_issues: 0, duration: 0, files: [] });
      }

      // Sanitize all filenames to prevent command injection
      const sanitizedFiles = files.map(f => {
        try {
          return sanitizeFilename(f);
        } catch (error) {
          logger.warn({ file: f, error }, 'Skipping file with invalid name');
          return null;
        }
      }).filter((f): f is string => f !== null);

      if (sanitizedFiles.length === 0) {
        logger.warn('All files were filtered out due to invalid names');
        return resolve({ total_issues: 0, duration: 0, files: [] });
      }

      // Construct command: goreview review file1 file2 ... --format json
      const args = ['review', ...sanitizedFiles, '--format', 'json'];

      const child = spawn('goreview', args, {
        cwd: workDir,
        env: { ...process.env }, // Pass environment variables
      });

      let stdout = '';
      let stderr = '';

      child.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      child.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      child.on('close', (code) => {
        if (code !== 0) {
          logger.error({ code, stderr, stdout }, 'GoReview CLI failed');
          return reject(new Error(`GoReview CLI exited with code ${code}: ${stderr}`));
        }

        try {
          // Parse JSON output
          const result: GoReviewResult = JSON.parse(stdout);
          logger.info({ issues: result.total_issues }, 'GoReview CLI completed');
          resolve(result);
        } catch (error) {
          logger.error({ error, stdout }, 'Failed to parse GoReview output');
          reject(new Error('Failed to parse GoReview output JSON'));
        }
      });

      child.on('error', (err) => {
        logger.error({ err }, 'Failed to spawn GoReview CLI');
        reject(err);
      });
    });
  }
}

export const goreviewService = new GoReviewService();
