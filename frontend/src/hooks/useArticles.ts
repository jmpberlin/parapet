import {
  useQuery,
  keepPreviousData,
  type UseQueryResult,
} from '@tanstack/react-query';
import type { components } from '../types/api';
import { getArticles, getArticlesPaginated, getArticleByID } from '../api';

type PaginatedArticles = components['schemas']['PaginatedArticles'];

export function useArticles(days = 7) {
  return useQuery({
    queryKey: ['articles', days],
    queryFn: () => getArticles(days),
  });
}

export function useArticlesPaginated(
  days = 7,
  page = 1,
): UseQueryResult<PaginatedArticles> {
  return useQuery({
    queryKey: ['articles', days, page],
    queryFn: () => getArticlesPaginated(days, page),
    placeholderData: keepPreviousData,
  });
}

export function useArticle(id: string) {
  return useQuery({
    queryKey: ['articles', id],
    queryFn: () => getArticleByID(id),
    enabled: !!id,
  });
}
