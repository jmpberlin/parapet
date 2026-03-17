import { describe, test, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { usePipelineStatus, useRunPipeline } from './usePipeline';
import * as api from '../api';

vi.mock('../api');

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('usePipelineStatus', () => {
  beforeEach(() => vi.clearAllMocks());

  test('returns pipeline result on success', async () => {
    vi.mocked(api.getPipelineStatus).mockResolvedValueOnce({
      ran_at: '2024-03-13T10:00:00Z',
      articles_harvested: 12,
      vulnerabilities_extracted: 3,
      deps_added: 5,
      deps_removed: 1,
      matches_found: 2,
      run_in_progress: false,
      errors: [],
    });

    const { result } = renderHook(() => usePipelineStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.articles_harvested).toBe(12);
    expect(result.current.data?.run_in_progress).toBe(false);
  });
});

describe('useRunPipeline', () => {
  test('calls runPipeline on mutate', async () => {
    vi.mocked(api.runPipeline).mockResolvedValueOnce({
      status: 'pipeline started',
    });

    const { result } = renderHook(() => useRunPipeline(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      result.current.mutate();
    });

    expect(api.runPipeline).toHaveBeenCalledOnce();
  });
});
