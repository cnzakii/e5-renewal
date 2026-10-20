import { basePath } from './lib/base-path'

export interface Session {
  authenticated: boolean
  csrf: string
}

export type AuthType = 'delegated' | 'application'
export type RunStatus =
  'running' | 'success' | 'partial' | 'failed' | 'interrupted'
export type RunTrigger = 'manual' | 'scheduled'

export interface Account {
  id: number
  name: string
  auth_type: AuthType
  tenant_id: string
  client_id: string
  enabled: boolean
  min_minutes: number
  max_minutes: number
  revision: number
  next_run_at: string | null
  last_run_at: string | null
  last_error: string
  last_error_code: string
  has_client_secret: boolean
  has_refresh_token: boolean
  statistics?: AccountStatistics
}

export interface AccountStatistics {
  requests: number
  successes: number
  daily: { day: string; requests: number; successes: number }[]
}

export interface Run {
  id: number
  account_id: number
  account_name: string
  trigger: RunTrigger
  status: RunStatus
  error: string
  error_code: string
  request_count: number
  success_count: number
  started_at: string
  finished_at: string | null
  results: {
    id: number
    started_at: string
    duration_ms: number
    attempts: number
    endpoint: string
    http_status: number
    error: string
    error_code: string
  }[]
}

export interface RunPage {
  runs: Run[]
  busy: boolean
  total: number
  page: number
  page_size: number
}

export interface AccountInput {
  name: string
  auth_type: AuthType
  client_id: string
  tenant_id: string
  enabled: boolean
  min_minutes: number
  max_minutes: number
  revision: number
  client_secret?: string
  refresh_token?: string
}

let csrf = ''

export class APIError extends Error {
  constructor(
    public code: string,
    message: string,
  ) {
    super(message)
  }
}

export function errorKey(error: unknown): string {
  return 'error.' + (error instanceof APIError ? error.code : 'request_failed')
}

export async function request<T>(
  path: string,
  method = 'GET',
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const response = await fetch(basePath() + '/api' + path, {
    method,
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      'X-Time-Zone': Intl.DateTimeFormat().resolvedOptions().timeZone,
      ...(method === 'GET' ? {} : { 'X-CSRF-Token': csrf }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  }).catch((error: unknown) => {
    if (signal?.aborted) {
      throw error
    }
    throw new APIError(
      'network_unavailable',
      'Could not connect to the server.',
    )
  })

  if (!response.ok) {
    const value = (await response.json().catch(() => ({}))) as {
      code?: string
      error?: string
    }
    if (
      (response.status === 401 ||
        value.code === 'csrf_invalid' ||
        value.code === 'authentication_changed') &&
      path !== '/login'
    ) {
      window.dispatchEvent(new Event('e5:unauthorized'))
    }
    throw new APIError(
      value.code || 'request_failed',
      value.error || 'The request failed.',
    )
  }
  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export async function loadSession(signal?: AbortSignal) {
  const session = await request<Session>('/session', 'GET', undefined, signal)
  if (!signal?.aborted) {
    csrf = session.csrf
  }

  return session
}

export function accountInput(account: Account): AccountInput {
  return {
    name: account.name,
    auth_type: account.auth_type,
    client_id: account.client_id,
    tenant_id: account.tenant_id,
    enabled: account.enabled,
    min_minutes: account.min_minutes,
    max_minutes: account.max_minutes,
    revision: account.revision,
  }
}

export function authorizationRequired(account?: Account): boolean {
  return (
    !!account &&
    account.auth_type === 'delegated' &&
    (!account.has_refresh_token ||
      [
        'authorization_invalid_grant',
        'authorization_invalid_client',
        'authorization_unauthorized_client',
        'authorization_missing_refresh_token',
        'authorization_invalid_scope',
        'authorization_consent_required',
        'authorization_interaction_required',
      ].includes(account.last_error_code))
  )
}
