import { describe, test, expect, vi, beforeEach } from 'vitest';
import { getArticles, getArticleByID } from './articles';
import { ApiError } from './client';

describe('getArticles', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  test('calls correct URL with default days', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => [],
    } as Response);

    await getArticles();

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/articles?days=7'),
      expect.any(Object),
    );
  });

  test('calls correct URL with provided days', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => [],
    } as Response);

    await getArticles(30);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/articles?days=30'),
      expect.any(Object),
    );
  });

  test('clamps days to 365 maximum', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => [],
    } as Response);

    await getArticles(999);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/articles?days=365'),
      expect.any(Object),
    );
  });

  test('clamps days to 1 minimum', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => [],
    } as Response);

    await getArticles(-5);

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/articles?days=1'),
      expect.any(Object),
    );
  });

  test('throws ApiError on 400', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: async () => ({ error: 'invalid days parameter' }),
    } as Response);

    await expect(getArticles(0)).rejects.toBeInstanceOf(ApiError);
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
