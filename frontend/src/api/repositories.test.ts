import { describe, test, expect, vi, beforeEach } from 'vitest';
import {
  getRepositories,
  getRepositoryByID,
  createRepository,
  getRepositoryMatches,
  getRepositoryDependencies,
} from './repositories';
import * as client from './client';

vi.mock('./client', () => ({
  apiFetch: vi.fn(),
}));

describe('repositories API', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getRepositories', () => {
    test('calls apiFetch with /repositories endpoint', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce([]);

      await getRepositories();

      expect(client.apiFetch).toHaveBeenCalledWith('/repositories');
    });
  });

  describe('getRepositoryByID', () => {
    test('calls apiFetch with repository ID', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        id: 'repo-1',
        owner_name: 'owner',
        repository_name: 'repo',
        git_provider: 'GitHub',
        dependency_count: 0,
        match_count: 0,
      });

      await getRepositoryByID('repo-1');

      expect(client.apiFetch).toHaveBeenCalledWith('/repositories/repo-1');
    });
  });

  describe('createRepository', () => {
    test('calls apiFetch with POST method and body', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        id: 'repo-1',
        owner_name: 'owner',
        repository_name: 'repo',
        git_provider: 'Github.com',
        dependency_count: 0,
        match_count: 0,
      });

      const body = { owner_name: 'owner', repository_name: 'repo', git_provider: 'Github.com' as const };
      await createRepository(body);

      expect(client.apiFetch).toHaveBeenCalledWith('/repositories', {
        method: 'POST',
        body: JSON.stringify(body),
      });
    });
  });

  describe('getRepositoryMatches', () => {
    test('uses default page 1 and limit 10', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryMatches('repo-1');

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toMatch(/\/repositories\/repo-1\/matches\?/);
      expect(call[0]).toContain('page=1');
      expect(call[0]).toContain('limit=10');
    });

    test('uses custom page parameter', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryMatches('repo-1', 3);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('page=3');
      expect(call[0]).toContain('limit=10');
    });

    test('uses custom limit parameter', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryMatches('repo-1', 1, 25);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('page=1');
      expect(call[0]).toContain('limit=25');
    });

    test('uses custom page and limit parameters together', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryMatches('repo-1', 2, 20);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('page=2');
      expect(call[0]).toContain('limit=20');
    });

    test('default limit is 10 (not 20)', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryMatches('repo-1', 1);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('limit=10');
    });
  });

  describe('getRepositoryDependencies', () => {
    test('uses default page 1 and limit 10', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryDependencies('repo-1');

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toMatch(/\/repositories\/repo-1\/dependencies\?/);
      expect(call[0]).toContain('page=1');
      expect(call[0]).toContain('limit=10');
    });

    test('uses custom page parameter', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryDependencies('repo-1', 2);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('page=2');
      expect(call[0]).toContain('limit=10');
    });

    test('uses custom limit parameter', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryDependencies('repo-1', 1, 30);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('page=1');
      expect(call[0]).toContain('limit=30');
    });

    test('uses custom page and limit parameters together', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryDependencies('repo-1', 3, 15);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('page=3');
      expect(call[0]).toContain('limit=15');
    });

    test('default limit is 10 (not 50)', async () => {
      vi.mocked(client.apiFetch).mockResolvedValueOnce({
        items: [],
        total_pages: 1,
      });

      await getRepositoryDependencies('repo-1', 1);

      const call = vi.mocked(client.apiFetch).mock.calls[0];
      expect(call[0]).toContain('limit=10');
    });
  });
});
