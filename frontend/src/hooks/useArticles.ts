import {
  useQueries,
  useQuery,
  keepPreviousData,
  type UseQueryResult,
} from '@tanstack/react-query';
import type { components } from '../types/api';
import { getArticlesPaginated, getArticleByID } from '../api';

type Article = components['schemas']['Article'];
type PaginatedArticles = components['schemas']['PaginatedArticles'];

const MAX_SOURCE_ARTICLES = 20;

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

export function useSourceArticles(articleIDs: string[]): UseQueryResult<Article>[] {
  // fetched per-ID; the cap bounds the fan-out if a poisoned vulnerability
  // record carries a very large source_article_ids list
  return useQueries({
    queries: articleIDs.slice(0, MAX_SOURCE_ARTICLES).map((articleId) => ({
      queryKey: ['articles', articleId],
      queryFn: () => getArticleByID(articleId),
      enabled: !!articleId,
    })),
  });
}
