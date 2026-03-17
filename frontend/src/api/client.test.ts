import { describe, test, expect, vi, beforeEach } from 'vitest';
import { apiFetch, ApiError } from './client';

describe('apiFetch', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  test('returns parsed JSON on success', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: 'ok' }),
    } as Response);

    const result = await apiFetch<{ status: string }>('/health');
    expect(result).toEqual({ status: 'ok' });
  });

  test('throws ApiError with status and message on non-ok response', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: async () => ({ error: 'article not found' }),
    } as Response);

    await expect(apiFetch('/articles/bad-id')).rejects.toThrow(
      'article not found',
    );
  });

  test('ApiError carries correct status code', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ error: 'internal server error' }),
    } as Response);

    try {
      await apiFetch('/repositories');
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(500);
    }
  });

  test('falls back to unknown error if response body is not valid JSON', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => {
        throw new Error('not json');
      },
    } as unknown as Response);

    await expect(apiFetch('/repositories')).rejects.toThrow('unknown error');
  });

  test('sends Content-Type application/json header', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => [],
    } as Response);

    await apiFetch('/repositories');

    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
      }),
    );
  });

  test('merges additional options into fetch call', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    } as Response);

    await apiFetch('/repositories', { method: 'POST', body: '{}' });

    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ method: 'POST', body: '{}' }),
    );
  });
});
