import type { components } from '../types/api';
import { apiFetch } from './client';

type RepositoryListItem = components['schemas']['RepositoryListItem'];
type RepositoryDetail = components['schemas']['RepositoryDetail'];
type CreateRepositoryRequest = components['schemas']['CreateRepositoryRequest'];

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
