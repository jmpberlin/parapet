import { describe, test, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useRepositories, useRepository, useRepositoryMatches, useRepositoryDependencies } from './useRepositories';
import * as api from '../api';

vi.mock('../api');

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false, // don't retry on failure in tests
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useRepositories', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('returns data on success', async () => {
    vi.mocked(api.getRepositories).mockResolvedValueOnce([
      {
        id: 'repo-1',
        owner_name: 'jmp',
        repository_name: 'nightwatch',
        git_provider: 'Github.com',
      },
    ]);

    const { result } = renderHook(() => useRepositories(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].repository_name).toBe('nightwatch');
  });

  test('returns error on failure', async () => {
    vi.mocked(api.getRepositories).mockRejectedValueOnce(
      new Error('failed to fetch repositories'),
    );

    const { result } = renderHook(() => useRepositories(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('failed to fetch repositories');
  });

  test('isLoading is true while fetching', async () => {
    vi.mocked(api.getRepositories).mockImplementation(
      () => new Promise(() => {}), // never resolves
    );

    const { result } = renderHook(() => useRepositories(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
  });
});

describe('useRepository', () => {
  test('does not fetch when id is empty', async () => {
    const { result } = renderHook(() => useRepository(''), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(api.getRepositoryByID).not.toHaveBeenCalled();
  });

  test('fetches when id is provided', async () => {
    vi.mocked(api.getRepositoryByID).mockResolvedValueOnce({
      id: 'repo-1',
      owner_name: 'jmp',
      repository_name: 'nightwatch',
      git_provider: 'Github.com',
      dependency_count: 0,
      match_count: 0,
    });

    const { result } = renderHook(() => useRepository('repo-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getRepositoryByID).toHaveBeenCalledWith('repo-1');
  });
});

describe('useRepositoryMatches', () => {
  test('does not fetch when id is empty', async () => {
    const { result } = renderHook(() => useRepositoryMatches('', 1), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(api.getRepositoryMatches).not.toHaveBeenCalled();
  });

  test('fetches matches with correct page parameter', async () => {
    vi.mocked(api.getRepositoryMatches).mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      limit: 10,
      total_pages: 1,
    });

    const { result } = renderHook(() => useRepositoryMatches('repo-1', 1), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getRepositoryMatches).toHaveBeenCalledWith('repo-1', 1);
  });

  test('fetches matches with custom page number', async () => {
    vi.mocked(api.getRepositoryMatches).mockResolvedValueOnce({
      items: [],
      total: 50,
      page: 3,
      limit: 10,
      total_pages: 5,
    });

    const { result } = renderHook(() => useRepositoryMatches('repo-1', 3), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getRepositoryMatches).toHaveBeenCalledWith('repo-1', 3);
  });

  test('returns paginated matches data', async () => {
    const mockMatches = {
      items: [
        {
          id: 'match-1',
          vulnerability_id: 'vuln-1',
          repository_id: 'repo-1',
          component_purl: 'pkg:npm/lodash@4.17.0',
          matched_component: 'lodash',
          matched_version: '4.17.0',
          status: 'CONFIRMED' as const,
          created_at: '2024-01-01T00:00:00Z',
        },
      ],
      total: 12,
      page: 1,
      limit: 10,
      total_pages: 2,
    };

    vi.mocked(api.getRepositoryMatches).mockResolvedValueOnce(mockMatches);

    const { result } = renderHook(() => useRepositoryMatches('repo-1', 1), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockMatches);
    expect(result.current.data?.total_pages).toBe(2);
  });
});

describe('useRepositoryDependencies', () => {
  test('does not fetch when id is empty', async () => {
    const { result } = renderHook(() => useRepositoryDependencies('', 1), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(api.getRepositoryDependencies).not.toHaveBeenCalled();
  });

  test('fetches dependencies with correct page parameter', async () => {
    vi.mocked(api.getRepositoryDependencies).mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      limit: 10,
      total_pages: 1,
    });

    const { result } = renderHook(() => useRepositoryDependencies('repo-1', 1), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getRepositoryDependencies).toHaveBeenCalledWith('repo-1', 1);
  });

  test('fetches dependencies with custom page number', async () => {
    vi.mocked(api.getRepositoryDependencies).mockResolvedValueOnce({
      items: [],
      total: 35,
      page: 2,
      limit: 10,
      total_pages: 4,
    });

    const { result } = renderHook(() => useRepositoryDependencies('repo-1', 2), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getRepositoryDependencies).toHaveBeenCalledWith('repo-1', 2);
  });

  test('returns paginated dependencies data', async () => {
    const mockDependencies = {
      items: [
        {
          id: 'dep-1',
          repository_id: 'repo-1',
          name: 'lodash',
          version: '4.17.21',
          purl: 'pkg:npm/lodash@4.17.21',
          created_at: '2024-01-01T00:00:00Z',
        },
      ],
      total: 25,
      page: 1,
      limit: 10,
      total_pages: 3,
    };

    vi.mocked(api.getRepositoryDependencies).mockResolvedValueOnce(mockDependencies);

    const { result } = renderHook(() => useRepositoryDependencies('repo-1', 1), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockDependencies);
    expect(result.current.data?.total_pages).toBe(3);
  });
});
