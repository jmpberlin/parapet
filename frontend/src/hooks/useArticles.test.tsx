import { describe, test, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useArticlesPaginated } from './useArticles';
import * as api from '../api';

vi.mock('../api');

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useArticlesPaginated', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('fetches paginated articles with page parameter', async () => {
    vi.mocked(api.getArticlesPaginated).mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 2,
      limit: 10,
      total_pages: 1,
    });

    const { result } = renderHook(() => useArticlesPaginated(7, 2), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.getArticlesPaginated).toHaveBeenCalledWith(7, 2);
  });

  test('keeps previous page data while next page is loading', async () => {
    const page1 = {
      items: [
        {
          id: 'a-1',
          source_url: 'https://example.com/1',
          host_domain: 'bleepingcomputer.com' as const,
          published_at: '2024-01-01T00:00:00Z',
          headline: 'First',
          author: '',
          content_html: '',
          content_cleaned: '',
          crawled_at: '2024-01-01T00:00:00Z',
        },
      ],
      total: 30,
      page: 1,
      limit: 10,
      total_pages: 3,
    };

    vi.mocked(api.getArticlesPaginated).mockImplementation((_days, page) => {
      if (page === 1) return Promise.resolve(page1);
      return new Promise(() => {}); // page 2 never resolves
    });

    const { result, rerender } = renderHook(
      ({ page }: { page: number }) => useArticlesPaginated(7, page),
      { wrapper: createWrapper(), initialProps: { page: 1 } },
    );

    await waitFor(() => expect(result.current.data?.page).toBe(1));

    rerender({ page: 2 });

    expect(result.current.data?.page).toBe(1);
    expect(result.current.isPlaceholderData).toBe(true);
    expect(result.current.isFetching).toBe(true);
    expect(result.current.isLoading).toBe(false);
  });
});
