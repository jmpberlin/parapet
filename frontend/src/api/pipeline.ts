import type { components } from '../types/api';
import { apiFetch } from './client';

type PipelineResult = components['schemas']['PipelineResult'];
type PipelineStartedResponse = components['schemas']['PipelineStartedResponse'];

export function runPipeline(): Promise<PipelineStartedResponse> {
  return apiFetch('/pipeline/run', { method: 'POST' });
}

export function getPipelineStatus(): Promise<PipelineResult> {
  return apiFetch('/pipeline/status');
}
