import type { components } from '../types/api';
import { apiFetch } from './client';

type RepositoryListItem = components['schemas']['RepositoryListItem'];
type RepositoryDetail = components['schemas']['RepositoryDetail'];
type CreateRepositoryRequest = components['schemas']['CreateRepositoryRequest'];
type PaginatedMatches = components['schemas']['PaginatedMatches'];
type PaginatedDependencies = components['schemas']['PaginatedDependencies'];

export function getRepositories(): Promise<RepositoryListItem[]> {
  return apiFetch('/repositories');
}

export function getRepositoryByID(id: string): Promise<RepositoryDetail> {
  return apiFetch(`/repositories/${id}`);
}

export function createRepository(
  body: CreateRepositoryRequest,
): Promise<RepositoryDetail> {
  return apiFetch('/repositories', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function getRepositoryMatches(
  id: string,
  page = 1,
  limit = 20,
): Promise<PaginatedMatches> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  return apiFetch(`/repositories/${id}/matches?${params}`);
}

export function getRepositoryDependencies(
  id: string,
  page = 1,
  limit = 50,
): Promise<PaginatedDependencies> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  return apiFetch(`/repositories/${id}/dependencies?${params}`);
}
