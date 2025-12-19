import { Octokit } from '@octokit/rest';
import { githubService } from './github.service.js';
import { goreviewService } from './goreview.service.js';
import { logger } from '../logger.js';
import * as fs from 'fs';
import * as path from 'path';
import { simpleGit } from 'simple-git';

export class OrchestratorService {
  async handlePullRequest(
    installationId: number,
    owner: string,
    repo: string,
    pullNumber: number,
    commitSha: string
  ): Promise<void> {
    const workDir = path.join(process.cwd(), 'tmp', `${owner}-${repo}-${pullNumber}`);
    
    try {
      logger.info({ owner, repo, pullNumber }, 'Starting review orchestration');

      // 1. Authenticate
      const octokit = await githubService.getInstallationClient(installationId);

      // 2. Get changed files
      const files = await githubService.getChangedFiles(octokit, owner, repo, pullNumber);
      if (files.length === 0) {
        logger.info('No files changed, skipping review');
        return;
      }

      // 3. Clone/Checkout code (Simplified: git clone)
      // In production, you'd use a more robust caching/cloning strategy
      await this.prepareWorkspace(workDir, owner, repo, commitSha, octokit);

      // 4. Run GoReview
      const result = await goreviewService.runReview(files, workDir);

      // 5. Report results (Log for now, later Checks API)
      logger.info(
        { 
          totalIssues: result.total_issues,
          duration: result.duration 
        }, 
        'Review completed'
      );

      // Log details
      for (const file of result.files) {
        if (file.response && file.response.issues.length > 0) {
          logger.info({ file: file.file, issues: file.response.issues }, 'Issues found');
        }
      }

    } catch (error) {
      logger.error({ error, owner, repo, pullNumber }, 'Orchestration failed');
    } finally {
      // Cleanup
      try {
        fs.rmSync(workDir, { recursive: true, force: true });
      } catch (e) {
        logger.warn({ error: e, workDir }, 'Failed to cleanup workspace');
      }
    }
  }

  private async prepareWorkspace(
    workDir: string,
    owner: string,
    repo: string,
    sha: string,
    octokit: Octokit
  ): Promise<void> {
    // Clean directory
    fs.rmSync(workDir, { recursive: true, force: true });
    fs.mkdirSync(workDir, { recursive: true });

    // Get auth token
    const { token } = await (octokit.auth({ type: 'installation' }) as Promise<{ token: string }>);
    const remote = `https://x-access-token:${token}@github.com/${owner}/${repo}.git`;

    const git = simpleGit(workDir);
    await git.init();
    await git.addRemote('origin', remote);
    await git.fetch('origin', sha);
    await git.checkout(sha);
  }
}

export const orchestratorService = new OrchestratorService();
