import type { components } from '../types/api';
import { apiFetch } from './client';
import { ApiError } from './client';

type Article = components['schemas']['Article'];

export function getArticles(days = 7): Promise<Article[]> {
  const sanitized = Math.max(1, Math.min(365, Math.floor(Number(days))));
  if (isNaN(sanitized)) {
    return Promise.reject(new ApiError(400, 'invalid days parameter'));
  }
  return apiFetch(`/articles?days=${sanitized}`);
}

export function getArticleByID(id: string): Promise<Article> {
  return apiFetch(`/articles/${id}`);
}
