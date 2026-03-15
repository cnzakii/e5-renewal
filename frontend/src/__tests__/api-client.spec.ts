// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest'
import { useAuth } from '../stores/auth'

describe('api client', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    const auth = useAuth()
    auth.clearAuth()
  })

  it('baseURL includes /api path', async () => {
    const { apiClient } = await import('../api/client')
    // pathPrefix is '' in test env, so baseURL should be '/api'
    expect(apiClient.defaults.baseURL).toBe('/api')
  })

  it('request interceptor adds Authorization header when token exists', async () => {
    const auth = useAuth()
    auth.setAuth('my-secret-token', false)

    const { apiClient } = await import('../api/client')

    // Simulate running the request interceptor by calling it with a mock config
    const interceptor = (apiClient.interceptors.request as any).handlers[0]
    const config = { headers: {} as Record<string, string> }
    const result = interceptor.fulfilled(config)

    expect(result.headers['Authorization']).toBe('Bearer my-secret-token')
  })

  it('request interceptor does not add Authorization header when no token', async () => {
    const { apiClient } = await import('../api/client')

    const interceptor = (apiClient.interceptors.request as any).handlers[0]
    const config = { headers: {} as Record<string, string> }
    const result = interceptor.fulfilled(config)

    expect(result.headers['Authorization']).toBeUndefined()
  })
})
