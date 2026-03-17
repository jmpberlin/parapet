import { useQuery } from '@tanstack/react-query';
import { getArticles, getArticleByID } from '../api';

export function useArticles(days = 7) {
  return useQuery({
    queryKey: ['articles', days],
    queryFn: () => getArticles(days),
  });
}

export function useArticle(id: string) {
  return useQuery({
    queryKey: ['articles', id],
    queryFn: () => getArticleByID(id),
    enabled: !!id,
  });
}
