const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://192.168.2.156:8080';
// const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://127.94.0.1:8080';
// const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://10.5.0.2:8080';

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export async function apiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown error' }));
    throw new ApiError(res.status, body.error ?? 'unknown error');
  }

  return res.json();
}
