import type { components } from '../types/api';
import { apiFetch } from './client';
import { ApiError } from './client';

type Article = components['schemas']['Article'];
type PaginatedArticles = components['schemas']['PaginatedArticles'];

function sanitizeDays(days: number): number {
  return Math.max(1, Math.min(365, Math.floor(Number(days))));
}

export function getArticlesPaginated(
  days = 7,
  page = 1,
  limit = 10,
): Promise<PaginatedArticles> {
  const sanitized = sanitizeDays(days);
  if (isNaN(sanitized)) {
    return Promise.reject(new ApiError(400, 'invalid days parameter'));
  }
  const params = new URLSearchParams({
    days: String(sanitized),
    page: String(page),
    limit: String(limit),
  });
  return apiFetch(`/articles?${params}`);
}

export function getArticleByID(id: string): Promise<Article> {
  return apiFetch(`/articles/${id}`);
}
