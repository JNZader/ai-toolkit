import { Octokit } from '@octokit/rest';
import { logger } from '../logger.js';
import { GoReviewResult } from './goreview.service.js';

export class ChecksService {
  /**
   * Create a new Check Run in "in_progress" state
   */
  async createCheckRun(
    octokit: Octokit,
    owner: string,
    repo: string,
    headSha: string
  ): Promise<number> {
    try {
      const { data } = await octokit.checks.create({
        owner,
        repo,
        name: 'GoReview AI Analysis',
        head_sha: headSha,
        status: 'in_progress',
        started_at: new Date().toISOString(),
        output: {
          title: 'AI Analysis Running',
          summary: 'GoReview is analyzing your changes...',
        },
      });
      return data.id;
    } catch (error) {
      logger.error({ error, owner, repo }, 'Failed to create check run');
      throw error;
    }
  }

  /**
   * Update Check Run with final results and annotations
   */
  async updateCheckRun(
    octokit: Octokit,
    owner: string,
    repo: string,
    checkRunId: number,
    result: GoReviewResult
  ): Promise<void> {
    try {
      const annotations = this.buildAnnotations(result);
      const conclusion = result.total_issues > 0 ? 'failure' : 'success';
      
      // GitHub API allows max 50 annotations per request
      // We'll take the first 50 for now (pagination logic can be added later)
      const batchAnnotations = annotations.slice(0, 50);

      await octokit.checks.update({
        owner,
        repo,
        check_run_id: checkRunId,
        status: 'completed',
        conclusion,
        completed_at: new Date().toISOString(),
        output: {
          title: `GoReview: ${result.total_issues} issues found`,
          summary: `Analysis completed in ${result.duration}s. Found ${result.total_issues} issues.`,
          text: this.buildMarkdownReport(result),
          annotations: batchAnnotations,
        },
      });
    } catch (error) {
      logger.error({ error, owner, repo, checkRunId }, 'Failed to update check run');
      throw error;
    }
  }

  private buildAnnotations(result: GoReviewResult): any[] {
    const annotations: any[] = [];

    for (const file of result.files) {
      if (!file.response || !file.response.issues) continue;

      for (const issue of file.response.issues) {
        if (issue.location && issue.location.start_line) {
          annotations.push({
            path: file.file,
            start_line: issue.location.start_line,
            end_line: issue.location.start_line, // Single line annotation for now
            annotation_level: this.mapSeverityToLevel(issue.severity),
            message: issue.message,
            title: `${issue.type} (${issue.id})`,
            raw_details: issue.suggestion ? `Suggestion: ${issue.suggestion}` : undefined,
          });
        }
      }
    }
    return annotations;
  }

  private mapSeverityToLevel(severity: string): 'notice' | 'warning' | 'failure' {
    switch (severity.toLowerCase()) {
      case 'critical':
      case 'error':
        return 'failure';
      case 'warning':
        return 'warning';
      default:
        return 'notice';
    }
  }

  private buildMarkdownReport(result: GoReviewResult): string {
    let md = `## Analysis Report\n\n`;
    md += `**Total Issues:** ${result.total_issues}\n`;
    md += `**Duration:** ${result.duration}s\n\n`;

    for (const file of result.files) {
      if (!file.response || file.response.issues.length === 0) continue;
      
      md += `### 📄 ${file.file}\n`;
      for (const issue of file.response.issues) {
        md += `- **[${issue.severity.toUpperCase()}]** ${issue.message}\n`;
        if (issue.suggestion) {
          md += `  > _Suggestion:_ ${issue.suggestion}\n`;
        }
      }
      md += `\n`;
    }
    return md;
  }
}

export const checksService = new ChecksService();
