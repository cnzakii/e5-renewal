import { shallowMount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockPush = vi.fn()
const mockSetAuth = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => ({ path: '/login' }),
}))

vi.mock('../stores/auth', () => ({
  useAuth: () => ({
    setAuth: mockSetAuth,
    token: { value: '' },
    isAuthenticated: { value: false },
  }),
}))

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../config', () => ({
  pathPrefix: '',
}))

import LoginView from '../views/LoginView.vue'
import { apiClient } from '../api/client'

const mountOptions = {
  global: {
    stubs: {
      AppLogo: { template: '<div class="app-logo-stub" />' },
      Transition: { template: '<div><slot /></div>' },
      transition: { template: '<div><slot /></div>' },
    },
  },
}

describe('LoginView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('renders login form with key input', () => {
    const wrapper = shallowMount(LoginView, mountOptions)
    const input = wrapper.find('input')
    expect(input.exists()).toBe(true)
    // Has a submit button
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.exists()).toBe(true)
  })

  it('submit button is disabled when key is empty', () => {
    const wrapper = shallowMount(LoginView, mountOptions)
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.attributes('disabled')).toBeDefined()
  })

  it('submit button is enabled when key is filled', async () => {
    const wrapper = shallowMount(LoginView, mountOptions)
    const input = wrapper.find('input')
    await input.setValue('my-secret-key')
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.attributes('disabled')).toBeUndefined()
  })

  it('shows error on wrong key (401)', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce({
      response: { status: 401, data: {} },
    })

    const wrapper = shallowMount(LoginView, mountOptions)
    const input = wrapper.find('input')
    await input.setValue('wrong-key')

    const form = wrapper.find('form')
    await form.trigger('submit')
    await flushPromises()

    // Error message should be visible
    expect(wrapper.text()).toMatch(/密钥无效|Invalid key/i)
  })

  it('shows backend error message when provided', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce({
      response: { status: 401, data: { error: 'Custom backend error' } },
    })

    const wrapper = shallowMount(LoginView, mountOptions)
    await wrapper.find('input').setValue('bad-key')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('Custom backend error')
  })

  it('shows network error when no response', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('Network Error'))

    const wrapper = shallowMount(LoginView, mountOptions)
    await wrapper.find('input').setValue('some-key')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    // Network error message
    expect(wrapper.text()).toMatch(/网络|Network/i)
  })

  it('successful login stores token and redirects', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: { token: 'jwt-token-123' },
    })

    const wrapper = shallowMount(LoginView, mountOptions)
    await wrapper.find('input').setValue('correct-key')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith('/login', { key: 'correct-key' })
    expect(mockSetAuth).toHaveBeenCalledWith('jwt-token-123', false)
    expect(mockPush).toHaveBeenCalledWith('/dashboard')
  })

  it('remember me toggle changes setAuth second argument', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: { token: 'jwt-token-456' },
    })

    const wrapper = shallowMount(LoginView, mountOptions)
    await wrapper.find('input').setValue('correct-key')

    // Click the remember me switch (role="switch")
    const toggle = wrapper.find('button[role="switch"]')
    expect(toggle.exists()).toBe(true)
    await toggle.trigger('click')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mockSetAuth).toHaveBeenCalledWith('jwt-token-456', true)
  })

  it('renders shared brand logo above the title', () => {
    const wrapper = shallowMount(LoginView, mountOptions)
    expect(wrapper.find('.app-logo-stub').exists()).toBe(true)
  })

  it('language toggle button exists and shows locale label', () => {
    const wrapper = shallowMount(LoginView, mountOptions)
    // The language toggle button is the first button in the component (absolute top-right)
    const buttons = wrapper.findAll('button[type="button"]')
    // At least the language toggle and the show/hide key toggle exist
    expect(buttons.length).toBeGreaterThanOrEqual(1)
    // Text should contain EN or 中 (the locale label)
    const langBtn = buttons[0]
    expect(langBtn.text()).toMatch(/EN|中/)
  })

  it('clears field error on input', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce({
      response: { status: 401, data: {} },
    })

    const wrapper = shallowMount(LoginView, mountOptions)
    await wrapper.find('input').setValue('bad-key')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    // Error shows
    expect(wrapper.text()).toMatch(/密钥无效|Invalid key/)

    // Type something to clear error
    await wrapper.find('input').trigger('input')
    await flushPromises()

    // Error should be cleared
    expect(wrapper.text()).not.toMatch(/密钥无效|Invalid key/)
  })
})
