import { Octokit } from '@octokit/rest';
import { githubService } from './github.service.js';
import { goreviewService, GoReviewResult } from './goreview.service.js';
import { checksService } from './checks.service.js';
import { logger } from '../logger.js';
import * as fs from 'fs';
import * as path from 'path';
import { simpleGit } from 'simple-git';

export interface ReviewRecord {
  timestamp: string;
  repo: string;
  pr: number;
  commit: string;
  issues: number;
  duration: number;
  status: 'success' | 'failure';
  details?: GoReviewResult;
}

export class OrchestratorService {
  private history: ReviewRecord[] = [];

  getHistory(): ReviewRecord[] {
    return this.history.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }

  async handlePullRequest(
    installationId: number,
    owner: string,
    repo: string,
    pullNumber: number,
    commitSha: string
  ): Promise<void> {
    const workDir = path.join(process.cwd(), 'tmp', `${owner}-${repo}-${pullNumber}`);
    let checkRunId: number | undefined;
    let octokit: Octokit;
    let reviewStatus: 'success' | 'failure' = 'failure'; // Default to failure until proven success
    let totalIssues = 0;
    let duration = 0;
    let reviewResult: GoReviewResult | undefined;

    try {
      logger.info({ owner, repo, pullNumber }, 'Starting review orchestration');

      // 1. Authenticate
      octokit = await githubService.getInstallationClient(installationId);

      // 2. Create Check Run
      checkRunId = await checksService.createCheckRun(octokit, owner, repo, commitSha);

      // 3. Get changed files
      const files = await githubService.getChangedFiles(octokit, owner, repo, pullNumber);
      if (files.length === 0) {
        logger.info('No files changed, skipping review');
        await checksService.updateCheckRun(octokit, owner, repo, checkRunId, { total_issues: 0, duration: 0, files: [] });
        return;
      }

      // 4. Clone/Checkout code
      await this.prepareWorkspace(workDir, owner, repo, commitSha, octokit);

      // 5. Run GoReview
      reviewResult = await goreviewService.runReview(files, workDir);
      totalIssues = reviewResult.total_issues;
      duration = reviewResult.duration;

      // 6. Report results to GitHub Checks
      await checksService.updateCheckRun(octokit, owner, repo, checkRunId, reviewResult);

      reviewStatus = 'success';

      logger.info(
        { 
          totalIssues: reviewResult.total_issues,
          duration: reviewResult.duration 
        }, 
        'Review completed'
      );

    } catch (error) {
      logger.error({ error, owner, repo, pullNumber }, 'Orchestration failed');
      
      // Update check run with failure if it exists
      if (checkRunId && octokit!) {
        try {
          await octokit.checks.update({
            owner, repo, check_run_id: checkRunId,
            status: 'completed', conclusion: 'failure',
            output: { title: 'Analysis Failed', summary: 'An internal error occurred during analysis.' }
          });
        } catch (e) { /* ignore */ }
      }
    } finally {
      // Record history
      this.history.push({
        timestamp: new Date().toISOString(),
        repo: `${owner}/${repo}`,
        pr: pullNumber,
        commit: commitSha.substring(0, 7),
        issues: totalIssues,
        duration: duration,
        status: reviewStatus,
        details: reviewResult
      });

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
