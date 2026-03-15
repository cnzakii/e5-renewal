import { mount } from '@vue/test-utils'
import { describe, it, expect, beforeEach } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppSidebar from '../components/AppSidebar.vue'
import { useAuth } from '../stores/auth'
import { THEME_STORAGE_KEY } from '../utils/theme'

function makeRouter(initialRoute = '/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
      { path: '/accounts', component: { template: '<div>Accounts</div>' } },
      { path: '/logs', component: { template: '<div>Logs</div>' } },
      { path: '/settings', component: { template: '<div>Settings</div>' } },
      { path: '/login', component: { template: '<div>Login</div>' } },
    ],
  })
  router.push(initialRoute)
  return router
}

describe('AppSidebar', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    const auth = useAuth()
    auth.clearAuth()
  })

  it('renders shared brand logo component instead of inline badge', async () => {
    const router = makeRouter()
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    // Should use the AppLogo component (renders SVG with data-testid)
    expect(wrapper.find('[data-testid="app-logo"]').exists()).toBe(true)
    // Should NOT have the old inline E5 text badge
    expect(wrapper.find('.bg-apple-blue.rounded-lg').exists()).toBe(false)
  })

  it('renders navigation links for dashboard, accounts, logs, settings', async () => {
    const router = makeRouter()
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    const text = wrapper.text()
    // i18n defaults to zh
    expect(text).toContain('仪表盘')
    expect(text).toContain('账号管理')
    expect(text).toContain('执行日志')
    expect(text).toContain('设置')
  })

  it('renders logout button', async () => {
    const router = makeRouter()
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    expect(wrapper.text()).toContain('退出登录')
  })

  it('emits update:collapsed when collapse toggle is clicked', async () => {
    const router = makeRouter()
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    // The collapse button is in the bottom actions area, first button there
    // It contains the double-chevron SVG
    const bottomButtons = wrapper.findAll('button')
    // First button in bottom section is collapse toggle
    const collapseBtn = bottomButtons[0]
    await collapseBtn.trigger('click')

    expect(wrapper.emitted('update:collapsed')).toBeTruthy()
    expect(wrapper.emitted('update:collapsed')![0]).toEqual([true])
  })

  it('emits update:collapsed with false when already collapsed', async () => {
    const router = makeRouter()
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: true },
      global: {
        plugins: [router],
      },
    })

    const bottomButtons = wrapper.findAll('button')
    const collapseBtn = bottomButtons[0]
    await collapseBtn.trigger('click')

    expect(wrapper.emitted('update:collapsed')![0]).toEqual([false])
  })

  it('applies active styling to current route link', async () => {
    const router = makeRouter('/dashboard')
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    const links = wrapper.findAll('a')
    // Dashboard link should have the active class
    const dashboardLink = links.find((l) => l.text().includes('仪表盘'))
    expect(dashboardLink).toBeTruthy()
    expect(dashboardLink!.classes().some((c) => c.includes('bg-apple-blue/12'))).toBe(true)

    // Other links should not have the active class
    const accountsLink = links.find((l) => l.text().includes('账号管理'))
    expect(accountsLink!.classes().some((c) => c.includes('bg-apple-blue/12'))).toBe(false)
  })

  it('highlights accounts link when on /accounts route', async () => {
    const router = makeRouter('/accounts')
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    const links = wrapper.findAll('a')
    const accountsLink = links.find((l) => l.text().includes('账号管理'))
    expect(accountsLink!.classes().some((c) => c.includes('bg-apple-blue/12'))).toBe(true)
  })

  it('hides text labels when collapsed', async () => {
    const router = makeRouter()
    await router.isReady()

    const wrapper = mount(AppSidebar, {
      props: { collapsed: true },
      global: {
        plugins: [router],
      },
    })

    // The sidebar should have width class w-16 when collapsed
    const aside = wrapper.find('aside')
    expect(aside.classes()).toContain('w-16')
  })

  it('toggles dark mode when dark mode button is clicked', async () => {
    const router = makeRouter()
    await router.isReady()

    // Ensure not dark initially
    document.documentElement.classList.remove('dark')

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    // Find the dark mode toggle button (has sun/moon icon)
    const buttons = wrapper.findAll('button')
    // The dark toggle is the second button (after collapse)
    const darkBtn = buttons[1]
    await darkBtn.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)

    // Toggle back
    await darkBtn.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('theme toggle persists preference via shared storage key and updates classList', async () => {
    const router = makeRouter()
    await router.isReady()

    document.documentElement.classList.remove('dark')
    localStorage.removeItem(THEME_STORAGE_KEY)

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    const buttons = wrapper.findAll('button')
    const darkBtn = buttons[1]

    // Toggle to dark
    await darkBtn.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')

    // Toggle back to light
    await darkBtn.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('auto')
  })

  it('logout button clears auth and navigates to login', async () => {
    const router = makeRouter()
    await router.isReady()

    // Set auth first
    const auth = useAuth()
    auth.setAuth('test-token', false)

    const wrapper = mount(AppSidebar, {
      props: { collapsed: false },
      global: {
        plugins: [router],
      },
    })

    // Find the logout button (contains text "退出登录")
    const logoutBtn = wrapper.findAll('button').find(b => b.text().includes('退出登录'))
    expect(logoutBtn).toBeDefined()
    await logoutBtn!.trigger('click')
    // Wait for navigation to complete
    await new Promise(r => setTimeout(r, 50))
    await router.isReady()

    // Auth should be cleared
    expect(auth.token.value).toBe('')
    // Should navigate to /login
    expect(router.currentRoute.value.path).toBe('/login')
  })
})
