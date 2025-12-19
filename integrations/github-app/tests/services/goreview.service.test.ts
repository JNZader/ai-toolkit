import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { GoReviewService } from '../../src/services/goreview.service.js';
import { spawn, ChildProcess } from 'child_process';
import { EventEmitter } from 'events';

// Mock child_process
vi.mock('child_process', () => ({
  spawn: vi.fn(),
}));

// Mock logger
vi.mock('../../src/logger.js', () => ({
  logger: {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  },
}));

describe('GoReviewService', () => {
  let service: GoReviewService;

  beforeEach(() => {
    service = new GoReviewService();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('runReview', () => {
    it('should return empty result when no files provided', async () => {
      const result = await service.runReview([], '/tmp/workdir');

      expect(result).toEqual({
        total_issues: 0,
        duration: 0,
        files: [],
      });
      expect(spawn).not.toHaveBeenCalled();
    });

    it('should execute CLI and parse JSON output', async () => {
      const mockStdout = JSON.stringify({
        total_issues: 2,
        duration: 1.5,
        files: [
          { file: 'test.go', response: { issues: [] } },
        ],
      });

      const mockProcess = new EventEmitter() as ChildProcess & EventEmitter;
      mockProcess.stdout = new EventEmitter() as any;
      mockProcess.stderr = new EventEmitter() as any;

      vi.mocked(spawn).mockReturnValue(mockProcess as any);

      const resultPromise = service.runReview(['test.go'], '/tmp/workdir');

      // Simulate stdout data
      mockProcess.stdout.emit('data', Buffer.from(mockStdout));
      // Simulate process close with success
      mockProcess.emit('close', 0);

      const result = await resultPromise;

      expect(result.total_issues).toBe(2);
      expect(result.duration).toBe(1.5);
      expect(result.files).toHaveLength(1);
    });

    it('should reject on CLI error', async () => {
      const mockProcess = new EventEmitter() as ChildProcess & EventEmitter;
      mockProcess.stdout = new EventEmitter() as any;
      mockProcess.stderr = new EventEmitter() as any;

      vi.mocked(spawn).mockReturnValue(mockProcess as any);

      const resultPromise = service.runReview(['test.go'], '/tmp/workdir');

      // Simulate stderr data
      mockProcess.stderr.emit('data', Buffer.from('CLI error'));
      // Simulate process close with error code
      mockProcess.emit('close', 1);

      await expect(resultPromise).rejects.toThrow('GoReview CLI exited with code 1');
    });

    it('should reject on invalid JSON output', async () => {
      const mockProcess = new EventEmitter() as ChildProcess & EventEmitter;
      mockProcess.stdout = new EventEmitter() as any;
      mockProcess.stderr = new EventEmitter() as any;

      vi.mocked(spawn).mockReturnValue(mockProcess as any);

      const resultPromise = service.runReview(['test.go'], '/tmp/workdir');

      // Simulate invalid JSON output
      mockProcess.stdout.emit('data', Buffer.from('not valid json'));
      mockProcess.emit('close', 0);

      await expect(resultPromise).rejects.toThrow('Failed to parse GoReview output JSON');
    });

    it('should filter out files with invalid names', async () => {
      const mockStdout = JSON.stringify({
        total_issues: 0,
        duration: 0.5,
        files: [],
      });

      const mockProcess = new EventEmitter() as ChildProcess & EventEmitter;
      mockProcess.stdout = new EventEmitter() as any;
      mockProcess.stderr = new EventEmitter() as any;

      vi.mocked(spawn).mockReturnValue(mockProcess as any);

      // Include a file with invalid characters
      const resultPromise = service.runReview(
        ['valid.go', '../../../etc/passwd', 'another.go'],
        '/tmp/workdir'
      );

      mockProcess.stdout.emit('data', Buffer.from(mockStdout));
      mockProcess.emit('close', 0);

      const result = await resultPromise;

      // Verify spawn was called with only valid files
      expect(spawn).toHaveBeenCalledWith(
        'goreview',
        ['review', 'valid.go', 'another.go', '--format', 'json'],
        expect.any(Object)
      );
    });

    it('should return empty result when all files are invalid', async () => {
      const result = await service.runReview(
        ['../../../etc/passwd', '$(whoami)'],
        '/tmp/workdir'
      );

      expect(result).toEqual({
        total_issues: 0,
        duration: 0,
        files: [],
      });
      expect(spawn).not.toHaveBeenCalled();
    });
  });
});
