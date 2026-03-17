import { describe, test, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useRepositories, useRepository } from './useRepositories';
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
      dependencies: [],
      matches: [],
    });

    const { result } = renderHook(() => useRepository('repo-1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getRepositoryByID).toHaveBeenCalledWith('repo-1');
  });
});
