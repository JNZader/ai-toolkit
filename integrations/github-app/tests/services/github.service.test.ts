import { describe, it, expect, vi, beforeEach } from 'vitest';
import { GitHubService } from '../../src/services/github.service.js';

// Mock fs
vi.mock('fs', () => ({
  readFileSync: vi.fn().mockReturnValue('fake-private-key'),
}));

// Mock config
vi.mock('../../src/config.js', () => ({
  config: {
    GITHUB_APP_ID: '12345',
    GITHUB_PRIVATE_KEY_PATH: '/path/to/key.pem',
  },
}));

// Mock logger
vi.mock('../../src/logger.js', () => ({
  logger: {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  },
}));

describe('GitHubService', () => {
  let service: GitHubService;

  beforeEach(() => {
    vi.clearAllMocks();
    service = new GitHubService();
  });

  describe('getChangedFiles', () => {
    it('should return list of changed files', async () => {
      const mockOctokit = {
        pulls: {
          listFiles: vi.fn().mockResolvedValue({
            data: [
              { filename: 'file1.go' },
              { filename: 'file2.go' },
              { filename: 'file3.go' },
            ],
          }),
        },
      };

      const files = await service.getChangedFiles(
        mockOctokit as any,
        'owner',
        'repo',
        1
      );

      expect(files).toEqual(['file1.go', 'file2.go', 'file3.go']);
      expect(mockOctokit.pulls.listFiles).toHaveBeenCalledWith({
        owner: 'owner',
        repo: 'repo',
        pull_number: 1,
        per_page: 100,
        page: 1,
      });
    });

    it('should handle pagination for large PRs', async () => {
      // First page returns 100 files (full page)
      const page1Files = Array.from({ length: 100 }, (_, i) => ({
        filename: `file${i + 1}.go`,
      }));
      // Second page returns 50 files (partial page = last page)
      const page2Files = Array.from({ length: 50 }, (_, i) => ({
        filename: `file${i + 101}.go`,
      }));

      const mockOctokit = {
        pulls: {
          listFiles: vi
            .fn()
            .mockResolvedValueOnce({ data: page1Files })
            .mockResolvedValueOnce({ data: page2Files }),
        },
      };

      const files = await service.getChangedFiles(
        mockOctokit as any,
        'owner',
        'repo',
        1
      );

      expect(files).toHaveLength(150);
      expect(mockOctokit.pulls.listFiles).toHaveBeenCalledTimes(2);
      expect(mockOctokit.pulls.listFiles).toHaveBeenNthCalledWith(1, {
        owner: 'owner',
        repo: 'repo',
        pull_number: 1,
        per_page: 100,
        page: 1,
      });
      expect(mockOctokit.pulls.listFiles).toHaveBeenNthCalledWith(2, {
        owner: 'owner',
        repo: 'repo',
        pull_number: 1,
        per_page: 100,
        page: 2,
      });
    });

    it('should handle API errors gracefully', async () => {
      const mockOctokit = {
        pulls: {
          listFiles: vi.fn().mockRejectedValue(new Error('API Error')),
        },
      };

      await expect(
        service.getChangedFiles(mockOctokit as any, 'owner', 'repo', 1)
      ).rejects.toThrow('API Error');
    });

    it('should stop pagination after safety limit', async () => {
      // Always return 100 files to simulate infinite pagination
      const fullPage = Array.from({ length: 100 }, (_, i) => ({
        filename: `file${i}.go`,
      }));

      const mockOctokit = {
        pulls: {
          listFiles: vi.fn().mockResolvedValue({ data: fullPage }),
        },
      };

      const files = await service.getChangedFiles(
        mockOctokit as any,
        'owner',
        'repo',
        1
      );

      // Should stop at page 30 (3000 files max)
      expect(mockOctokit.pulls.listFiles).toHaveBeenCalledTimes(30);
      expect(files).toHaveLength(3000);
    });
  });

  describe('getPullRequestDiff', () => {
    it('should return diff as string', async () => {
      const mockDiff = 'diff --git a/file.go b/file.go\n...';
      const mockOctokit = {
        pulls: {
          get: vi.fn().mockResolvedValue({ data: mockDiff }),
        },
      };

      const diff = await service.getPullRequestDiff(
        mockOctokit as any,
        'owner',
        'repo',
        1
      );

      expect(diff).toBe(mockDiff);
      expect(mockOctokit.pulls.get).toHaveBeenCalledWith({
        owner: 'owner',
        repo: 'repo',
        pull_number: 1,
        mediaType: { format: 'diff' },
      });
    });
  });
});
