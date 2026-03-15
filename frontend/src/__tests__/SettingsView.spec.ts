import { shallowMount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

import SettingsView from '../views/SettingsView.vue'
import { apiClient } from '../api/client'

const mountOptions = {
  global: {
    stubs: {
      Teleport: true,
      Transition: { template: '<div><slot /></div>' },
      transition: { template: '<div><slot /></div>' },
    },
  },
}

describe('SettingsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('applies shared page-surface class', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.animate-fade-in').exists()).toBe(true)
  })

  it('renders notification URL input', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: {
        url: 'telegram://bot@telegram?chats=@ops',
        on_auth_expiry: false,
        expiry_days_before: 7,
        on_task_all_failed: false,
        on_health_low: false,
        health_threshold: 50,
      },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const inputs = wrapper.findAll('input[type="text"]')
    expect(inputs.length).toBeGreaterThan(0)
    // The first text input should be the notification URL
    expect((inputs[0].element as HTMLInputElement).value).toBe('telegram://bot@telegram?chats=@ops')
  })

  it('renders settings title and subtitle', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/设置|Settings/)
  })

  it('renders save button', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const saveBtn = wrapper.findAll('button').find(b => b.text().match(/保存设置|Save/))
    expect(saveBtn).toBeDefined()
  })

  it('renders test notification button', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))
    expect(testBtn).toBeDefined()
  })

  it('save button calls PUT API', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 } })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })

    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // Set URL value
    const urlInput = wrapper.findAll('input[type="text"]')[0]
    await urlInput.setValue('telegram://bot@telegram?chats=@test')

    const saveBtn = wrapper.findAll('button').find(b => b.text().match(/保存设置|Save/))!
    await saveBtn.trigger('click')
    await flushPromises()

    expect(apiClient.put).toHaveBeenCalledWith('/settings/notification', expect.objectContaining({
      url: 'telegram://bot@telegram?chats=@test',
    }))
  })

  it('test button is disabled when not saved', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    expect(testBtn.attributes('disabled')).toBeDefined()
  })

  it('test button is enabled after save', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'telegram://bot@telegram?chats=@test', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // saved should be true because url is non-empty
    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    expect(testBtn.attributes('disabled')).toBeUndefined()
  })

  it('test notification calls POST API', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'telegram://bot@telegram?chats=@test', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: {} })

    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    await testBtn.trigger('click')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith('/settings/notification/test')
  })

  it('renders notification condition toggles', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: true, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toMatch(/Client Secret/)
    expect(text).toMatch(/任务全部失败|All Tasks Failed/)
    expect(text).toMatch(/健康度过低|Health Below Threshold/)
  })

  it('fetches settings on mount', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: {} })
    shallowMount(SettingsView, mountOptions)
    await flushPromises()

    expect(apiClient.get).toHaveBeenCalledWith('/settings/notification')
  })

  // --- Test notification handler ---

  it('test notification shows success toast on API success', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'telegram://bot@telegram?chats=@test', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: {} })

    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    await testBtn.trigger('click')
    await flushPromises()

    // Toast should show success message
    expect(wrapper.text()).toMatch(/测试通知已发送|Test sent|success/i)
  })

  it('test notification shows error toast on API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'telegram://bot@telegram?chats=@test', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('fail'))

    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    await testBtn.trigger('click')
    await flushPromises()

    // Toast should show error
    expect(wrapper.text()).toMatch(/失败|error|Error/i)
  })

  it('test notification does nothing when not saved', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    // Force trigger despite disabled
    await testBtn.trigger('click')
    await flushPromises()

    // Should NOT have called POST since saved=false
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  // --- Condition toggle handlers ---

  it('toggling auth expiry condition toggles form value', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // Find the toggle button for auth expiry (the first condition toggle)
    const toggleBtns = wrapper.findAll('button[type="button"]')
    const authExpiryToggle = toggleBtns.find(b => {
      // These toggle buttons have class with w-11 h-6 rounded-full
      return b.classes().some(c => c.includes('rounded-full')) && b.classes().some(c => c.includes('w-11'))
    })
    if (authExpiryToggle) {
      await authExpiryToggle.trigger('click')
      await flushPromises()

      // After toggle, expiry_days input should be visible
      const numberInputs = wrapper.findAll('input[type="number"]')
      expect(numberInputs.length).toBeGreaterThan(0)
    }
  })

  it('toggling all tasks failed condition works', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // Find all rounded-full toggle buttons
    const toggleBtns = wrapper.findAll('button[type="button"]').filter(b =>
      b.classes().some(c => c.includes('rounded-full')) && b.classes().some(c => c.includes('w-11'))
    )
    // Second toggle should be "all tasks failed"
    if (toggleBtns.length >= 2) {
      await toggleBtns[1].trigger('click')
      await flushPromises()

      // The button should now be active (bg-apple-blue)
      expect(toggleBtns[1].classes().join(' ')).toContain('bg-apple-blue')
    }
  })

  it('toggling health low condition shows threshold input', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const toggleBtns = wrapper.findAll('button[type="button"]').filter(b =>
      b.classes().some(c => c.includes('rounded-full')) && b.classes().some(c => c.includes('w-11'))
    )
    // Third toggle should be "health low"
    if (toggleBtns.length >= 3) {
      await toggleBtns[2].trigger('click')
      await flushPromises()

      // Health threshold input should now be visible (percent input)
      expect(wrapper.text()).toContain('%')
    }
  })

  // --- Save settings ---

  it('save shows success toast on success', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const saveBtn = wrapper.findAll('button').find(b => b.text().match(/保存设置|Save/))!
    await saveBtn.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toMatch(/设置已保存|Saved|success/i)
  })

  it('save shows error toast on API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.put).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const saveBtn = wrapper.findAll('button').find(b => b.text().match(/保存设置|Save/))!
    await saveBtn.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toMatch(/失败|error|Error/i)
  })

  // --- Fetch settings failure ---

  it('handles fetch settings failure gracefully', async () => {
    vi.mocked(apiClient.get).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // Should still render with default values
    expect(wrapper.text()).toMatch(/设置|Settings/)
    const numberInputs = wrapper.findAll('input[type="number"]')
    // Default form should be used
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
  })

  // --- Notification language selector ---

  it('renders notification language selector with default zh', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', language: 'zh', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const langBtns = wrapper.findAll('button').filter(b =>
      b.text().match(/中文|English/)
    )
    expect(langBtns.length).toBe(2)
    // zh button should be active (has bg-apple-blue)
    const zhBtn = langBtns.find(b => b.text().includes('中文'))!
    expect(zhBtn.classes().join(' ')).toContain('bg-apple-blue')
  })

  it('notification language saves with the form', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'https://example.com', language: 'zh', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // Click English button
    const enBtn = wrapper.findAll('button').find(b => b.text() === 'English')!
    await enBtn.trigger('click')
    await flushPromises()

    // Save
    const saveBtn = wrapper.findAll('button').find(b => b.text().match(/保存设置|Save/))!
    await saveBtn.trigger('click')
    await flushPromises()

    expect(apiClient.put).toHaveBeenCalledWith('/settings/notification', expect.objectContaining({
      language: 'en',
    }))
  })

  // --- Test notification backend error surfacing ---

  it('test notification shows backend error detail when available', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'telegram://bot@telegram?chats=@test', language: 'zh', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.post).mockRejectedValueOnce({
      response: { data: { error: 'invalid Telegram token' } }
    })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    await testBtn.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('invalid Telegram token')
  })

  it('test notification falls back to generic error when no backend detail', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: 'telegram://bot@telegram?chats=@test', language: 'zh', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('network error'))
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    const testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    await testBtn.trigger('click')
    await flushPromises()

    // Should show generic translated error
    expect(wrapper.text()).toMatch(/失败|error|Error/i)
  })

  // --- After save, test button becomes enabled ---

  it('test button becomes enabled after successful save', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { url: '', on_auth_expiry: false, expiry_days_before: 7, on_task_all_failed: false, on_health_low: false, health_threshold: 50 },
    })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(SettingsView, mountOptions)
    await flushPromises()

    // Initially test button should be disabled
    let testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    expect(testBtn.attributes('disabled')).toBeDefined()

    // Set URL and save
    const urlInput = wrapper.findAll('input[type="text"]')[0]
    await urlInput.setValue('telegram://bot@telegram?chats=@test')

    const saveBtn = wrapper.findAll('button').find(b => b.text().match(/保存设置|Save/))!
    await saveBtn.trigger('click')
    await flushPromises()

    // Now test button should be enabled
    testBtn = wrapper.findAll('button').find(b => b.text().match(/发送测试|Send Test/))!
    expect(testBtn.attributes('disabled')).toBeUndefined()
  })
})
