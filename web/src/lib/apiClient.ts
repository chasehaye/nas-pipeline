import { env } from '../config/env'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface RequestOptions {
  method?: HttpMethod
  body?: unknown
  signal?: AbortSignal
  headers?: Record<string, string>
}

export interface ApiError extends Error {
  status?: number
}

export async function sendRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = 'GET', body, signal, headers } = options

  const res = await fetch(`${env.apiBase}${path}`, {
    method,
    signal,
    headers: {
      Accept: 'application/json',
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    const err = new Error(`${method} ${path} failed: ${res.status}`) as ApiError
    err.status = res.status
    throw err
  }

  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}
