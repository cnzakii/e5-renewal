// @vitest-environment jsdom
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createMemoryHistory, createRouter, type RouteRecordRaw } from 'vue-router'
import { useAuth } from '../stores/auth'

// Mock all view/component imports that buildRoutes pulls in,
// since some transitively import CSS which Node cannot handle.
vi.mock('../views/LoginView.vue', () => ({ default: { template: '<div>Login</div>' } }))
vi.mock('../views/DashboardView.vue', () => ({ default: { template: '<div>Dashboard</div>' } }))
vi.mock('../views/AccountsView.vue', () => ({ default: { template: '<div>Accounts</div>' } }))
vi.mock('../views/LogsView.vue', () => ({ default: { template: '<div>Logs</div>' } }))
vi.mock('../views/SettingsView.vue', () => ({ default: { template: '<div>Settings</div>' } }))
vi.mock('../components/AppLayout.vue', () => ({ default: { template: '<div><router-view /></div>' } }))

import { buildRoutes } from '../router'

function makeRouter(routes?: RouteRecordRaw[]) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: routes ?? buildRoutes(),
  })

  // Replicate the same guard from the real router
  router.beforeEach((to) => {
    const { isAuthenticated } = useAuth()
    if (!to.meta.guest && !isAuthenticated.value) {
      return '/login'
    }
    if (to.meta.guest && isAuthenticated.value) {
      return '/dashboard'
    }
  })

  return router
}

describe('router', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    const auth = useAuth()
    auth.clearAuth()
  })

  describe('buildRoutes', () => {
    it('returns route array with /login and / paths', () => {
      const routes = buildRoutes()
      const paths = routes.map((r) => r.path)
      expect(paths).toContain('/login')
      expect(paths).toContain('/')
    })

    it('/login route has guest meta', () => {
      const routes = buildRoutes()
      const login = routes.find((r) => r.path === '/login')
      expect(login?.meta?.guest).toBe(true)
    })

    it('/ layout route has child routes for dashboard, accounts, logs, settings', () => {
      const routes = buildRoutes()
      const layout = routes.find((r) => r.path === '/' && r.children)
      const childPaths = layout?.children?.map((c) => c.path) ?? []
      expect(childPaths).toContain('/dashboard')
      expect(childPaths).toContain('/accounts')
      expect(childPaths).toContain('/logs')
      expect(childPaths).toContain('/settings')
    })

    it('/ has a redirect to /dashboard', () => {
      const routes = buildRoutes()
      const redirect = routes.find((r) => r.path === '/' && r.redirect)
      expect(redirect?.redirect).toBe('/dashboard')
    })

    it('has a catch-all route redirecting to /dashboard', () => {
      const routes = buildRoutes()
      const catchAll = routes.find((r) => r.path === '/:pathMatch(.*)*')
      expect(catchAll?.redirect).toBe('/dashboard')
    })
  })

  describe('auth guard', () => {
    it('redirects unauthenticated users to /login for protected routes', async () => {
      const router = makeRouter()
      await router.push('/dashboard')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/login')
    })

    it('redirects unauthenticated users to /login for /accounts', async () => {
      const router = makeRouter()
      await router.push('/accounts')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/login')
    })

    it('allows authenticated users to access protected routes', async () => {
      const auth = useAuth()
      auth.setAuth('valid-token', false)

      const router = makeRouter()
      await router.push('/dashboard')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })
  })

  describe('guest guard', () => {
    it('redirects authenticated users from /login to /dashboard', async () => {
      const auth = useAuth()
      auth.setAuth('valid-token', false)

      const router = makeRouter()
      await router.push('/login')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })

    it('allows unauthenticated users to access /login', async () => {
      const router = makeRouter()
      await router.push('/login')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/login')
    })
  })

  describe('root redirect', () => {
    it('/ redirects to /dashboard (then guard may redirect to /login)', async () => {
      const router = makeRouter()
      await router.push('/')
      await router.isReady()
      // Unauthenticated: / -> /dashboard -> /login
      expect(router.currentRoute.value.path).toBe('/login')
    })

    it('/ redirects to /dashboard for authenticated users', async () => {
      const auth = useAuth()
      auth.setAuth('valid-token', false)

      const router = makeRouter()
      await router.push('/')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })
  })
})
