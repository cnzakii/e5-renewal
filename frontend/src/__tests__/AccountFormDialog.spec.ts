import { mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import AccountFormDialog from '../components/AccountFormDialog.vue'
import { apiClient } from '../api/client'

vi.mock('../api/client', () => ({
  apiClient: {
    post: vi.fn(),
    get: vi.fn(),
  },
}))

const mockedApiPost = vi.mocked(apiClient.post)
const mockedApiGet = vi.mocked((apiClient as any).get)

async function flushPromises() {
  await Promise.resolve()
  await Promise.resolve()
}

function getSaveButton(wrapper: VueWrapper<any>) {
  const buttons = wrapper.findAll('button')
  const button = buttons.find(candidate => candidate.text().includes('保存') || candidate.text().includes('Save'))
  expect(button).toBeTruthy()
  return button!
}

async function mountDialog(account?: any) {
  const wrapper = mount(AccountFormDialog, {
    props: { visible: false, account: account ?? null },
    global: {
      stubs: {
        Teleport: true,
        Transition: false,
      },
    },
  })
  // Trigger the watch by setting visible to true
  await wrapper.setProps({ visible: true })
  return wrapper
}

function getInputByPlaceholder(wrapper: VueWrapper<any>, placeholder: string) {
  return wrapper.find(`input[placeholder*="${placeholder}"]`)
}

function getTextareaByPlaceholder(wrapper: VueWrapper<any>, placeholder: string | RegExp) {
  const textareas = wrapper.findAll('textarea')
  return textareas.find(ta => {
    const ph = ta.attributes('placeholder') || ''
    return typeof placeholder === 'string' ? ph.includes(placeholder) : placeholder.test(ph)
  })
}

async function fillBaseFields(wrapper: VueWrapper<any>) {
  const nameInput = getInputByPlaceholder(wrapper, 'Contoso')
  const tenantInput = wrapper.findAll('input').find(i => {
    const ph = i.attributes('placeholder') || ''
    return ph.includes('Tenant')
  })
  const clientIdInput = wrapper.findAll('input').find(i => {
    const ph = i.attributes('placeholder') || ''
    return ph.includes('Client ID')
  })
  const clientSecretInput = wrapper.findAll('input').find(i => {
    const ph = i.attributes('placeholder') || ''
    return ph.includes('Client Secret')
  })

  if (nameInput.exists()) await nameInput.setValue('Contoso Dev')
  if (tenantInput) await tenantInput.setValue('tenant-123')
  if (clientIdInput) await clientIdInput.setValue('client-123')
  if (clientSecretInput) await clientSecretInput.setValue('secret-123')
}

describe('AccountFormDialog OAuth UX', () => {
  const windowOpen = vi.fn()

  beforeEach(() => {
    mockedApiPost.mockReset()
    mockedApiGet.mockReset()
    windowOpen.mockReset()
    vi.useRealTimers()

    vi.stubGlobal('open', windowOpen)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('opens in edit mode for new accounts (no account prop)', async () => {
    const wrapper = await mountDialog()
    // New account mode -> shows edit form (not preview)
    expect(wrapper.text()).toContain('添加账号')
  })

  it('defaults auth_type to auth_code and shows refresh token area', async () => {
    const wrapper = await mountDialog()
    // auth_code is selected by default, so Refresh Token section is visible
    expect(wrapper.text()).toContain('Refresh Token')
  })

  it('shows auto authorize button for getting refresh token', async () => {
    const wrapper = await mountDialog()
    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    expect(authorizeBtn.exists()).toBe(true)
  })

  it('auto authorize button is disabled when client_id and tenant_id are empty', async () => {
    const wrapper = await mountDialog()
    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    expect(authorizeBtn.exists()).toBe(true)
    expect((authorizeBtn.element as HTMLButtonElement).disabled).toBe(true)
  })

  it('auto authorize button is enabled when client_id and tenant_id are filled', async () => {
    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)
    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    expect(authorizeBtn.exists()).toBe(true)
    expect((authorizeBtn.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('calls /oauth/authorize via POST when auto authorize is clicked', async () => {
    mockedApiPost.mockResolvedValueOnce({ data: { authorize_url: 'https://login.example/authorize' } } as any)
    windowOpen.mockReturnValue({ closed: false, close: vi.fn() })

    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    await authorizeBtn.trigger('click')
    await flushPromises()

    expect(mockedApiPost).toHaveBeenCalledWith('/oauth/authorize', {
      client_id: 'client-123',
      client_secret: 'secret-123',
      tenant_id: 'tenant-123',
    })
  })

  it('opens popup with authorize URL', async () => {
    mockedApiPost.mockResolvedValueOnce({ data: { authorize_url: 'https://login.example/authorize' } } as any)
    windowOpen.mockReturnValue({ closed: false, close: vi.fn() })

    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    await authorizeBtn.trigger('click')
    await flushPromises()

    expect(windowOpen).toHaveBeenCalledWith(
      'https://login.example/authorize',
      'e5-oauth',
      'width=600,height=700,scrollbars=yes'
    )
  })

  it('stores refresh_token from postMessage success', async () => {
    mockedApiPost.mockResolvedValueOnce({ data: { authorize_url: 'https://login.example/authorize' } } as any)
    const popupClose = vi.fn()
    const popup = { closed: false, close: popupClose }
    windowOpen.mockReturnValue(popup)

    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    await authorizeBtn.trigger('click')
    await flushPromises()

    window.dispatchEvent(new MessageEvent('message', {
      origin: window.location.origin,
      source: popup as any,
      data: {
        type: 'e5-oauth-result',
        data: {
          status: 'success',
          payload: JSON.stringify({ refresh_token: 'rt-123' }),
        },
      },
    }))
    await flushPromises()

    await getSaveButton(wrapper).trigger('click')
    const saveEvents = wrapper.emitted('save')
    expect(saveEvents).toBeTruthy()
    expect(saveEvents?.[0]?.[0]).toMatchObject({ refresh_token: 'rt-123' })
  })

  it('shows error when manual OAuth authorize fails', async () => {
    mockedApiPost.mockRejectedValueOnce({ response: { data: { error: 'invalid client' } } })

    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    // Switch to manual mode
    const manualBtn = wrapper.findAll('button').find(b => b.text() === '手动获取')
    await manualBtn!.trigger('click')

    const redirectInput = wrapper.find('input[placeholder*="localhost"]')
    await redirectInput.setValue('http://localhost:3000/api/oauth/callback')

    const generateBtn = wrapper.findAll('button').find(b => b.text() === '生成授权链接')
    await generateBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toMatch(/失败|failed/i)
  })

  it('shows verify button and calls /accounts/verify with correct payload', async () => {
    mockedApiPost.mockResolvedValueOnce({ data: {} } as any)

    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    // Switch to direct mode to set refresh token
    const directBtn = wrapper.findAll('button').find(b => b.text() === '直接填写')
    await directBtn!.trigger('click')

    const textarea = wrapper.find('[data-testid="direct-textarea"]')
    await textarea.setValue('rt-verify')

    const verifyBtn = wrapper.findAll('button').find(b =>
      b.text().includes('验证') || b.text().includes('Verify')
    )
    expect(verifyBtn).toBeTruthy()

    await verifyBtn!.trigger('click')
    await flushPromises()

    expect(mockedApiPost).toHaveBeenCalledWith('/accounts/verify', {
      auth_type: 'auth_code',
      client_id: 'client-123',
      client_secret: 'secret-123',
      tenant_id: 'tenant-123',
      refresh_token: 'rt-verify',
    })
  })

  it('emits save with form data on submit', async () => {
    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    // Switch to direct mode to set refresh token
    const directBtn = wrapper.findAll('button').find(b => b.text() === '直接填写')
    await directBtn!.trigger('click')

    const textarea = wrapper.find('[data-testid="direct-textarea"]')
    await textarea.setValue('rt-test')

    await getSaveButton(wrapper).trigger('click')

    const saveEvents = wrapper.emitted('save')
    expect(saveEvents).toBeTruthy()
    expect(saveEvents?.[0]?.[0]).toMatchObject({
      name: 'Contoso Dev',
      client_id: 'client-123',
      client_secret: 'secret-123',
      tenant_id: 'tenant-123',
      refresh_token: 'rt-test',
      auth_type: 'auth_code',
    })
  })

  it('opens in preview mode for existing accounts', async () => {
    const wrapper = await mountDialog({
      id: 1,
      name: 'Test Account',
      auth_type: 'auth_code',
      client_id: 'cid',
      client_secret: 'csec',
      tenant_id: 'tid',
      refresh_token: 'rt',
      notify_enabled: false,
      auth_expires_at: '',
    })
    // Preview mode shows account details title
    expect(wrapper.text()).toContain('账号详情')
  })

  it('validates required fields before saving', async () => {
    const wrapper = await mountDialog()
    // Don't fill any fields, just click save
    await getSaveButton(wrapper).trigger('click')

    // Should not emit save
    expect(wrapper.emitted('save')).toBeFalsy()
  })

  it('emits save without refresh_token for client_credentials type', async () => {
    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    // Switch to client_credentials
    const credBtn = wrapper.findAll('button').find(b => b.text().includes('凭据'))
    expect(credBtn).toBeTruthy()
    await credBtn!.trigger('click')

    // Save should succeed without refresh_token
    await getSaveButton(wrapper).trigger('click')
    const saveEvents = wrapper.emitted('save')
    expect(saveEvents).toBeTruthy()
    expect(saveEvents?.[0]?.[0]).toMatchObject({
      auth_type: 'client_credentials',
      client_id: 'client-123',
    })
  })

  it('emits update:visible false when close button is clicked', async () => {
    const wrapper = await mountDialog()

    // Find close button (X button or cancel)
    const cancelBtn = wrapper.findAll('button').find(b => b.text() === '取消')
    expect(cancelBtn).toBeTruthy()
    await cancelBtn!.trigger('click')

    expect(wrapper.emitted('update:visible')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })

  it('clicking edit for existing account triggers GET request to load secrets', async () => {
    mockedApiGet.mockResolvedValueOnce({
      data: {
        id: 1, name: 'Test Account', auth_type: 'auth_code',
        client_id: 'cid', client_secret: 'plaintext-secret-value',
        tenant_id: 'tid', refresh_token: 'plaintext-refresh-token',
      },
    })

    const wrapper = await mountDialog({
      id: 1,
      name: 'Test Account',
      auth_type: 'auth_code',
      client_id: 'cid',
      client_secret: 'mask********cret',
      tenant_id: 'tid',
      refresh_token: 'mask********oken',
      notify_enabled: false,
      auth_expires_at: '',
    })

    // Should be in preview mode
    expect(wrapper.text()).toContain('账号详情')

    // Click Edit button
    const editBtn = wrapper.findAll('button').find(b => b.text().includes('编辑'))
    expect(editBtn).toBeTruthy()
    await editBtn!.trigger('click')
    await flushPromises()

    // Should have called GET /accounts/1
    expect(mockedApiGet).toHaveBeenCalledWith('/accounts/1')
  })

  it('form shows plaintext values after successful secret load', async () => {
    mockedApiGet.mockResolvedValueOnce({
      data: {
        id: 1, name: 'Test Account', auth_type: 'auth_code',
        client_id: 'cid', client_secret: 'plaintext-secret-value',
        tenant_id: 'tid', refresh_token: 'plaintext-refresh-token',
      },
    })

    const wrapper = await mountDialog({
      id: 1,
      name: 'Test Account',
      auth_type: 'auth_code',
      client_id: 'cid',
      client_secret: 'mask********cret',
      tenant_id: 'tid',
      refresh_token: 'mask********oken',
      notify_enabled: false,
      auth_expires_at: '',
    })

    // Click Edit button
    const editBtn = wrapper.findAll('button').find(b => b.text().includes('编辑'))
    await editBtn!.trigger('click')
    await flushPromises()

    // Should now be in edit mode
    expect(wrapper.text()).toContain('编辑账号')

    // Submit the form - all fields already populated from account prop + GET response
    await getSaveButton(wrapper).trigger('click')
    const saveEvents = wrapper.emitted('save')
    expect(saveEvents).toBeTruthy()
    expect(saveEvents?.[0]?.[0]).toMatchObject({
      client_secret: 'plaintext-secret-value',
      refresh_token: 'plaintext-refresh-token',
    })
  })

  it('failed secret load shows error and stays in preview mode', async () => {
    mockedApiGet.mockRejectedValueOnce({
      response: { data: { error: 'account not found' } },
    })

    const wrapper = await mountDialog({
      id: 99,
      name: 'Missing Account',
      auth_type: 'auth_code',
      client_id: 'cid',
      client_secret: 'mask********cret',
      tenant_id: 'tid',
      refresh_token: 'mask********oken',
      notify_enabled: false,
      auth_expires_at: '',
    })

    // Should be in preview mode
    expect(wrapper.text()).toContain('账号详情')

    // Click Edit button
    const editBtn = wrapper.findAll('button').find(b => b.text().includes('编辑'))
    await editBtn!.trigger('click')
    await flushPromises()

    // Should still be in preview mode (not edit)
    expect(wrapper.text()).toContain('账号详情')
    // Should show error
    expect(wrapper.text()).toContain('account not found')
  })

  it('failed secret load uses fallback i18n message when no response error', async () => {
    mockedApiGet.mockRejectedValueOnce(new Error('Network error'))

    const wrapper = await mountDialog({
      id: 99,
      name: 'Missing Account',
      auth_type: 'auth_code',
      client_id: 'cid',
      client_secret: 'mask********cret',
      tenant_id: 'tid',
      refresh_token: 'mask********oken',
      notify_enabled: false,
      auth_expires_at: '',
    })

    const editBtn = wrapper.findAll('button').find(b => b.text().includes('编辑'))
    await editBtn!.trigger('click')
    await flushPromises()

    // Should show fallback error message
    expect(wrapper.text()).toContain('加载密钥失败')
  })
})

describe('AccountFormDialog RefreshTokenModePanel integration', () => {
  beforeEach(() => {
    mockedApiPost.mockReset()
    mockedApiGet.mockReset()
    vi.useRealTimers()
  })

  it('auth-code accounts default to auto mode', async () => {
    const wrapper = await mountDialog()
    // Default auth_type is auth_code, so RefreshTokenModePanel should render
    // Auto mode is the default, so the auto authorize button should be visible
    expect(wrapper.text()).toContain('Refresh Token')
    // Check for mode selector buttons - auto should be present
    const autoBtn = wrapper.findAll('button').find(b => b.text() === '自动获取')
    expect(autoBtn).toBeTruthy()
  })

  it('direct mode shows textarea', async () => {
    const wrapper = await mountDialog()
    // Click the direct mode button
    const directBtn = wrapper.findAll('button').find(b => b.text() === '直接填写')
    expect(directBtn).toBeTruthy()
    await directBtn!.trigger('click')

    // Should show a textarea for direct input
    const textarea = wrapper.find('[data-testid="direct-textarea"]')
    expect(textarea.exists()).toBe(true)
  })

  it('auto mode shows authorize button', async () => {
    const wrapper = await mountDialog()
    // Auto mode is the default
    const authorizeBtn = wrapper.find('[data-testid="auto-authorize-btn"]')
    expect(authorizeBtn.exists()).toBe(true)
    expect(authorizeBtn.text()).toContain('授权获取')
  })

  it('manual mode shows exchange button after generating auth link', async () => {
    mockedApiPost.mockResolvedValueOnce({ data: { authorize_url: 'https://login.example/authorize' } } as any)

    const wrapper = await mountDialog()
    await fillBaseFields(wrapper)

    // Click the manual mode button
    const manualBtn = wrapper.findAll('button').find(b => b.text() === '手动获取')
    expect(manualBtn).toBeTruthy()
    await manualBtn!.trigger('click')

    // Fill in custom redirect URI
    const redirectInput = wrapper.find('input[placeholder*="localhost"]')
    expect(redirectInput.exists()).toBe(true)
    await redirectInput.setValue('http://localhost:3000/api/oauth/callback')

    // Click generate auth link
    const generateBtn = wrapper.findAll('button').find(b => b.text() === '生成授权链接')
    expect(generateBtn).toBeTruthy()
    await generateBtn!.trigger('click')
    await flushPromises()

    // Exchange button should now be visible
    const exchangeBtn = wrapper.find('[data-testid="manual-exchange-btn"]')
    expect(exchangeBtn.exists()).toBe(true)
    expect(exchangeBtn.text()).toContain('交换令牌')
  })
})
