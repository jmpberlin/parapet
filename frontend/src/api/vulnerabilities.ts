import type { components } from '../types/api';
import { apiFetch } from './client';

type Vulnerability = components['schemas']['Vulnerability'];

export function getVulnerabilities(): Promise<Vulnerability[]> {
  return apiFetch('/vulnerabilities');
}
