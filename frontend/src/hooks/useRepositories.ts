import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  createRepository,
  getRepositories,
  getRepositoryByID,
  getRepositoryMatches,
  getRepositoryDependencies,
} from '../api';
import type { components } from '../types/api';

type CreateRepositoryRequest = components['schemas']['CreateRepositoryRequest'];

export function useRepositories() {
  return useQuery({
    queryKey: ['repositories'],
    queryFn: getRepositories,
  });
}

export function useRepository(id: string) {
  return useQuery({
    queryKey: ['repositories', id],
    queryFn: () => getRepositoryByID(id),
    enabled: !!id,
  });
}

export function useRepositoryMatches(id: string, page: number) {
  return useQuery({
    queryKey: ['repositories', id, 'matches', page],
    queryFn: () => getRepositoryMatches(id, page),
    enabled: !!id,
  });
}

export function useRepositoryDependencies(id: string, page: number) {
  return useQuery({
    queryKey: ['repositories', id, 'dependencies', page],
    queryFn: () => getRepositoryDependencies(id, page),
    enabled: !!id,
  });
}

export function useCreateRepository() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateRepositoryRequest) => createRepository(body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
    },
  });
}
