import { describe, test, expect, vi, beforeEach } from 'vitest';
import { getArticlesPaginated, getArticleByID } from './articles';
import { ApiError } from './client';

describe('getArticlesPaginated', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  test('calls correct URL with default days, page and limit', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 1, limit: 10, total_pages: 1 }),
    } as Response);

    await getArticlesPaginated();

    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/articles\?days=7&page=1&limit=10/),
      expect.any(Object),
    );
  });

  test('calls correct URL with custom page and limit', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 3, limit: 25, total_pages: 1 }),
    } as Response);

    await getArticlesPaginated(7, 3, 25);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/articles\?days=7&page=3&limit=25/),
      expect.any(Object),
    );
  });

  test('clamps days to 365 maximum', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 1, limit: 10, total_pages: 1 }),
    } as Response);

    await getArticlesPaginated(999);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/articles\?days=365&page=1&limit=10/),
      expect.any(Object),
    );
  });

  test('clamps days below 1 up to 1', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 1, limit: 10, total_pages: 1 }),
    } as Response);

    await getArticlesPaginated(0);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/articles\?days=1&page=1&limit=10/),
      expect.any(Object),
    );
  });

  test('floors fractional days', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 1, limit: 10, total_pages: 1 }),
    } as Response);

    await getArticlesPaginated(7.9);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/articles\?days=7&page=1&limit=10/),
      expect.any(Object),
    );
  });

  test('rejects with ApiError for NaN days without calling fetch', async () => {
    await expect(getArticlesPaginated(NaN)).rejects.toBeInstanceOf(ApiError);
    expect(fetch).not.toHaveBeenCalled();
  });

  test('throws ApiError on 400', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({ error: 'invalid pagination parameters' }),
    } as Response);

    await expect(getArticlesPaginated(0)).rejects.toBeInstanceOf(ApiError);
  });
});

describe('getArticleByID', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  test('calls correct URL with id', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ id: 'abc-123' }),
    } as Response);

    await getArticleByID('abc-123');

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/articles/abc-123'),
      expect.any(Object),
    );
  });

  test('throws ApiError with 404 when article not found', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: async () => ({ error: 'article not found' }),
    } as Response);

    try {
      await getArticleByID('bad-id');
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(404);
      expect((e as ApiError).message).toBe('article not found');
    }
  });
});
