import { Octokit } from '@octokit/rest';
import { createAppAuth } from '@octokit/auth-app';
import { config } from '../config.js';
import { logger } from '../logger.js';
import { readFileSync } from 'fs';

export class GitHubService {
  private privateKey: string;

  constructor() {
    try {
      this.privateKey = readFileSync(config.GITHUB_PRIVATE_KEY_PATH, 'utf-8');
    } catch (error) {
      logger.error({ error, path: config.GITHUB_PRIVATE_KEY_PATH }, 'Failed to read private key');
      this.privateKey = ''; // Will fail on auth
    }
  }

  /**
   * Get an authenticated Octokit instance for a specific installation
   */
  async getInstallationClient(installationId: number): Promise<Octokit> {
    return new Octokit({
      authStrategy: createAppAuth,
      auth: {
        appId: config.GITHUB_APP_ID,
        privateKey: this.privateKey,
        installationId: installationId,
      },
    });
  }

  /**
   * Get the diff of a Pull Request
   */
  async getPullRequestDiff(
    octokit: Octokit,
    owner: string,
    repo: string,
    pullNumber: number
  ): Promise<string> {
    try {
      const { data } = await octokit.pulls.get({
        owner,
        repo,
        pull_number: pullNumber,
        mediaType: {
          format: 'diff',
        },
      });
      // Octokit types generic data as any/unknown for custom media types sometimes, 
      // but 'diff' format returns a string.
      return data as unknown as string;
    } catch (error) {
      logger.error({ error, owner, repo, pullNumber }, 'Failed to get PR diff');
      throw error;
    }
  }

  /**
   * Get changed files in a PR with pagination support
   * PERF-005: Handles PRs with more than 100 files
   */
  async getChangedFiles(
    octokit: Octokit,
    owner: string,
    repo: string,
    pullNumber: number
  ): Promise<string[]> {
    try {
      const files: string[] = [];
      let page = 1;
      let hasMore = true;

      while (hasMore) {
        const { data } = await octokit.pulls.listFiles({
          owner,
          repo,
          pull_number: pullNumber,
          per_page: 100,
          page,
        });

        files.push(...data.map((f) => f.filename));

        // If we got less than 100 files, we've reached the end
        hasMore = data.length === 100;
        page++;

        // Safety limit to prevent infinite loops (max 3000 files)
        if (page > 30) {
          logger.warn({ owner, repo, pullNumber, totalFiles: files.length }, 'Reached file limit for PR');
          break;
        }
      }

      return files;
    } catch (error) {
      logger.error({ error, owner, repo, pullNumber }, 'Failed to get changed files');
      throw error;
    }
  }
}

export const githubService = new GitHubService();
