import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ChecksService } from '../../src/services/checks.service.js';
import type { GoReviewResult } from '../../src/services/goreview.service.js';

// Mock logger
vi.mock('../../src/logger.js', () => ({
  logger: {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  },
}));

describe('ChecksService', () => {
  let service: ChecksService;

  beforeEach(() => {
    vi.clearAllMocks();
    service = new ChecksService();
  });

  describe('createCheckRun', () => {
    it('should create check run in progress state', async () => {
      const mockOctokit = {
        checks: {
          create: vi.fn().mockResolvedValue({ data: { id: 123 } }),
        },
      };

      const checkRunId = await service.createCheckRun(
        mockOctokit as any,
        'owner',
        'repo',
        'abc123'
      );

      expect(checkRunId).toBe(123);
      expect(mockOctokit.checks.create).toHaveBeenCalledWith({
        owner: 'owner',
        repo: 'repo',
        name: 'GoReview AI Analysis',
        head_sha: 'abc123',
        status: 'in_progress',
        started_at: expect.any(String),
        output: {
          title: 'AI Analysis Running',
          summary: 'GoReview is analyzing your changes...',
        },
      });
    });

    it('should throw on API error', async () => {
      const mockOctokit = {
        checks: {
          create: vi.fn().mockRejectedValue(new Error('API Error')),
        },
      };

      await expect(
        service.createCheckRun(mockOctokit as any, 'owner', 'repo', 'abc123')
      ).rejects.toThrow('API Error');
    });
  });

  describe('updateCheckRun', () => {
    it('should update with success when no issues', async () => {
      const mockOctokit = {
        checks: {
          update: vi.fn().mockResolvedValue({}),
        },
      };

      const result: GoReviewResult = {
        total_issues: 0,
        duration: 1.5,
        files: [],
      };

      await service.updateCheckRun(
        mockOctokit as any,
        'owner',
        'repo',
        123,
        result
      );

      expect(mockOctokit.checks.update).toHaveBeenCalledWith(
        expect.objectContaining({
          owner: 'owner',
          repo: 'repo',
          check_run_id: 123,
          status: 'completed',
          conclusion: 'success',
        })
      );
    });

    it('should update with failure when issues found', async () => {
      const mockOctokit = {
        checks: {
          update: vi.fn().mockResolvedValue({}),
        },
      };

      const result: GoReviewResult = {
        total_issues: 2,
        duration: 1.5,
        files: [
          {
            file: 'test.go',
            response: {
              issues: [
                {
                  id: 'SEC-001',
                  type: 'security',
                  severity: 'critical',
                  message: 'Security issue',
                  location: { file: 'test.go', start_line: 10 },
                },
              ],
            },
          },
        ],
      };

      await service.updateCheckRun(
        mockOctokit as any,
        'owner',
        'repo',
        123,
        result
      );

      expect(mockOctokit.checks.update).toHaveBeenCalledWith(
        expect.objectContaining({
          conclusion: 'failure',
          output: expect.objectContaining({
            annotations: expect.arrayContaining([
              expect.objectContaining({
                path: 'test.go',
                start_line: 10,
                annotation_level: 'failure',
              }),
            ]),
          }),
        })
      );
    });

    it('should limit annotations to 50', async () => {
      const mockOctokit = {
        checks: {
          update: vi.fn().mockResolvedValue({}),
        },
      };

      // Create 60 issues
      const issues = Array.from({ length: 60 }, (_, i) => ({
        id: `ISSUE-${i}`,
        type: 'quality',
        severity: 'warning',
        message: `Issue ${i}`,
        location: { file: 'test.go', start_line: i + 1 },
      }));

      const result: GoReviewResult = {
        total_issues: 60,
        duration: 5,
        files: [
          {
            file: 'test.go',
            response: { issues },
          },
        ],
      };

      await service.updateCheckRun(
        mockOctokit as any,
        'owner',
        'repo',
        123,
        result
      );

      // Verify only 50 annotations were sent
      const updateCall = mockOctokit.checks.update.mock.calls[0][0];
      expect(updateCall.output.annotations).toHaveLength(50);
    });

    it('should map severity correctly', async () => {
      const mockOctokit = {
        checks: {
          update: vi.fn().mockResolvedValue({}),
        },
      };

      const result: GoReviewResult = {
        total_issues: 3,
        duration: 1,
        files: [
          {
            file: 'test.go',
            response: {
              issues: [
                {
                  id: 'I1',
                  type: 'security',
                  severity: 'critical',
                  message: 'Critical',
                  location: { file: 'test.go', start_line: 1 },
                },
                {
                  id: 'I2',
                  type: 'quality',
                  severity: 'warning',
                  message: 'Warning',
                  location: { file: 'test.go', start_line: 2 },
                },
                {
                  id: 'I3',
                  type: 'style',
                  severity: 'info',
                  message: 'Info',
                  location: { file: 'test.go', start_line: 3 },
                },
              ],
            },
          },
        ],
      };

      await service.updateCheckRun(
        mockOctokit as any,
        'owner',
        'repo',
        123,
        result
      );

      const annotations = mockOctokit.checks.update.mock.calls[0][0].output.annotations;
      expect(annotations[0].annotation_level).toBe('failure'); // critical -> failure
      expect(annotations[1].annotation_level).toBe('warning'); // warning -> warning
      expect(annotations[2].annotation_level).toBe('notice'); // info -> notice
    });
  });
});
