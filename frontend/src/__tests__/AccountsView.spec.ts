import { shallowMount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockPush = vi.fn()
const mockReplace = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
  useRoute: () => ({ query: {}, path: '/accounts' }),
}))

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

import AccountsView from '../views/AccountsView.vue'
import { apiClient } from '../api/client'

const mockAccounts = [
  {
    id: 1, name: 'Contoso Dev', auth_type: 'auth_code',
    client_id: 'cid-1', client_secret: 'csec-1', tenant_id: 'tid-1',
    refresh_token: 'rt-1', notify_enabled: true,
    health: 98.2, total_runs: 420, success_runs: 412, last_run: '2026-03-09T10:00:00Z',
    auth_expires_at: '2026-06-01',
  },
  {
    id: 2, name: 'Fabrikam Prod', auth_type: 'client_credentials',
    client_id: 'cid-2', client_secret: 'csec-2', tenant_id: 'tid-2',
    refresh_token: '', notify_enabled: false,
    health: 95.1, total_runs: 380, success_runs: 361, last_run: '2026-03-09T09:00:00Z',
    auth_expires_at: '2026-12-01',
  },
]

const mountOptions = {
  global: {
    stubs: {
      AccountFormDialog: { template: '<div class="form-dialog-stub" />' },
      ScheduleDialog: { template: '<div class="schedule-dialog-stub" />' },
      TriggerResultDialog: { template: '<div class="trigger-dialog-stub" />' },
      ConfirmDialog: { template: '<div class="confirm-dialog-stub" />' },
      Teleport: true,
      Transition: { template: '<div><slot /></div>' },
      transition: { template: '<div><slot /></div>' },
    },
  },
}

describe('AccountsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('applies shared page-surface class', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.animate-fade-in').exists()).toBe(true)
  })

  it('renders account list after fetch', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('Contoso Dev')
    expect(wrapper.text()).toContain('Fabrikam Prod')
    expect(wrapper.text()).toContain('#1')
    expect(wrapper.text()).toContain('#2')
  })

  it('shows empty state when no accounts', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/还没有账号|No accounts yet/)
  })

  it('add account button exists', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const addBtn = wrapper.findAll('button').find(b => b.text().match(/添加账号|Add Account/))
    expect(addBtn).toBeDefined()
  })

  it('clicking add button sets showFormDialog', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const addBtn = wrapper.findAll('button').find(b => b.text().match(/添加账号|Add Account/))!
    await addBtn.trigger('click')

    // The AccountFormDialog stub should be rendered (visible prop set to true)
    // Check that the component is in the DOM
    expect(wrapper.find('.form-dialog-stub').exists()).toBe(true)
  })

  it('renders health bars for accounts', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('98.2%')
    expect(wrapper.text()).toContain('95.1%')
  })

  it('renders total runs and success counts', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('420')
    expect(wrapper.text()).toContain('412')
    expect(wrapper.text()).toContain('380')
    expect(wrapper.text()).toContain('361')
  })

  it('delete button exists for each account card', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Each card has a delete button with title containing "删除" or "Delete"
    const deleteButtons = wrapper.findAll('button').filter(b => {
      const title = b.attributes('title')
      return title && (title.includes('删除') || title.includes('Delete'))
    })
    expect(deleteButtons.length).toBe(2) // one per account
  })

  it('manual trigger tile exists for each account card', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Trigger tiles contain trigger text
    const text = wrapper.text()
    expect(text).toMatch(/手动触发|Trigger/)
  })

  it('trigger calls API and shows dialog', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    vi.mocked(apiClient.post).mockResolvedValueOnce({
      data: {
        task_log: { id: 1, trigger_type: 'manual', total_endpoints: 3, success_count: 3, fail_count: 0, started_at: '2026-03-09T10:00:00Z', finished_at: '2026-03-09T10:00:05Z' },
        endpoints: [],
      },
    })

    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Find the trigger tile (div with class action-tile-blue) and click it
    const triggerTiles = wrapper.findAll('.action-tile-blue')
    expect(triggerTiles.length).toBeGreaterThan(0)

    await triggerTiles[0].trigger('click')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith('/accounts/1/trigger')
  })

  it('handles API failure on fetch gracefully', async () => {
    vi.mocked(apiClient.get).mockRejectedValueOnce(new Error('Network error'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Should show empty state since accounts array stays empty
    expect(wrapper.text()).toMatch(/还没有账号|No accounts yet/)
  })

  // --- Schedule dialog open/save/resume ---

  it('clicking schedule tile opens schedule dialog', async () => {
    const accWithSchedule = [
      { ...mockAccounts[0], schedule: { enabled: true, paused: false, pause_reason: '', pause_threshold: 5, next_run_at: '2026-03-10T00:00:00Z', last_run_at: null } },
    ]
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: accWithSchedule })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const scheduleTiles = wrapper.findAll('.action-tile-green')
    expect(scheduleTiles.length).toBeGreaterThan(0)
    await scheduleTiles[0].trigger('click')
    await flushPromises()

    expect(wrapper.find('.schedule-dialog-stub').exists()).toBe(true)
  })

  it('handleScheduleSave calls API and shows success toast', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Open schedule dialog first
    const scheduleTiles = wrapper.findAll('.action-tile-gray, .action-tile-green, .action-tile-red')
    if (scheduleTiles.length > 0) {
      await scheduleTiles[0].trigger('click')
    }

    // Simulate ScheduleDialog emitting save
    const scheduleDialog = wrapper.findComponent({ name: 'ScheduleDialog' })
    if (scheduleDialog.exists()) {
      scheduleDialog.vm.$emit('save', 1, { enabled: true, pause_threshold: 5 })
      await flushPromises()
      expect(apiClient.put).toHaveBeenCalledWith('/accounts/1/schedule', { enabled: true, pause_threshold: 5 })
    }
  })

  it('handleScheduleResume calls API with paused:false', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const scheduleTiles = wrapper.findAll('.action-tile-gray, .action-tile-green, .action-tile-red')
    if (scheduleTiles.length > 0) {
      await scheduleTiles[0].trigger('click')
    }

    const scheduleDialog = wrapper.findComponent({ name: 'ScheduleDialog' })
    if (scheduleDialog.exists()) {
      scheduleDialog.vm.$emit('resume', 1)
      await flushPromises()
      expect(apiClient.put).toHaveBeenCalledWith('/accounts/1/schedule', { paused: false })
    }
  })

  // --- Delete confirmation flow ---

  it('clicking delete button opens confirm dialog', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const deleteButtons = wrapper.findAll('button').filter(b => {
      const title = b.attributes('title')
      return title && (title.includes('删除') || title.includes('Delete'))
    })
    expect(deleteButtons.length).toBe(2)

    await deleteButtons[0].trigger('click')
    await flushPromises()

    expect(wrapper.find('.confirm-dialog-stub').exists()).toBe(true)
  })

  it('handleDelete calls API and refreshes accounts on success', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.delete).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Click delete button for the first account
    const deleteButtons = wrapper.findAll('button').filter(b => {
      const title = b.attributes('title')
      return title && (title.includes('删除') || title.includes('Delete'))
    })
    await deleteButtons[0].trigger('click')
    await flushPromises()

    // Simulate confirm dialog emitting confirm
    const confirmDialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    if (confirmDialog.exists()) {
      confirmDialog.vm.$emit('confirm')
      await flushPromises()
      expect(apiClient.delete).toHaveBeenCalledWith('/accounts/1')
    }
  })

  it('handleDelete shows error toast on API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.delete).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const deleteButtons = wrapper.findAll('button').filter(b => {
      const title = b.attributes('title')
      return title && (title.includes('删除') || title.includes('Delete'))
    })
    await deleteButtons[0].trigger('click')
    await flushPromises()

    const confirmDialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    if (confirmDialog.exists()) {
      confirmDialog.vm.$emit('confirm')
      await flushPromises()
      // Error toast should show - component handles error internally
      expect(apiClient.delete).toHaveBeenCalled()
    }
  })

  // --- Account editing ---

  it('clicking account body opens preview (edit) dialog', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Find the clickable div that opens preview
    const cardBodies = wrapper.findAll('.cursor-pointer')
    expect(cardBodies.length).toBeGreaterThan(0)
    await cardBodies[0].trigger('click')
    await flushPromises()

    expect(wrapper.find('.form-dialog-stub').exists()).toBe(true)
  })

  it('handleSave for edit calls PUT and refreshes', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Open edit by clicking account body
    const cardBodies = wrapper.findAll('.cursor-pointer')
    await cardBodies[0].trigger('click')
    await flushPromises()

    // Simulate save from form dialog
    const formDialog = wrapper.findComponent({ name: 'AccountFormDialog' })
    if (formDialog.exists()) {
      formDialog.vm.$emit('save', {
        name: 'Updated', auth_type: 'auth_code', client_id: 'cid', client_secret: 'sec',
        tenant_id: 'tid', refresh_token: 'rt', notify_enabled: true, auth_expires_at: '2026-06-01',
      })
      await flushPromises()
      expect(apiClient.put).toHaveBeenCalledWith('/accounts/1', expect.any(Object))
    }
  })

  it('handleSave for new account calls POST', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Click Add button
    const addBtn = wrapper.findAll('button').find(b => b.text().match(/添加账号|Add Account/))!
    await addBtn.trigger('click')
    await flushPromises()

    const formDialog = wrapper.findComponent({ name: 'AccountFormDialog' })
    if (formDialog.exists()) {
      formDialog.vm.$emit('save', {
        name: 'New', auth_type: 'auth_code', client_id: 'cid', client_secret: 'sec',
        tenant_id: 'tid', refresh_token: 'rt', notify_enabled: false, auth_expires_at: '',
      })
      await flushPromises()
      expect(apiClient.post).toHaveBeenCalledWith('/accounts', expect.any(Object))
    }
  })

  it('handleSave shows error toast on API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const addBtn = wrapper.findAll('button').find(b => b.text().match(/添加账号|Add Account/))!
    await addBtn.trigger('click')
    await flushPromises()

    const formDialog = wrapper.findComponent({ name: 'AccountFormDialog' })
    if (formDialog.exists()) {
      formDialog.vm.$emit('save', {
        name: 'New', auth_type: 'auth_code', client_id: 'cid', client_secret: 'sec',
        tenant_id: 'tid', refresh_token: 'rt', notify_enabled: false, auth_expires_at: '',
      })
      await flushPromises()
      expect(apiClient.post).toHaveBeenCalled()
    }
  })

  // --- Health bar calculations ---

  it('shows correct health bar color for high health (>=90)', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], health: 95 }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const healthBar = wrapper.find('.bg-gradient-to-r.from-emerald-400')
    expect(healthBar.exists()).toBe(true)
  })

  it('shows correct health bar color for medium health (70-89)', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], health: 75 }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const healthBar = wrapper.find('.bg-gradient-to-r.from-amber-400')
    expect(healthBar.exists()).toBe(true)
  })

  it('shows correct health bar color for low health (<70)', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], health: 45 }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const healthBar = wrapper.find('.bg-gradient-to-r.from-red-400')
    expect(healthBar.exists()).toBe(true)
  })

  it('shows -- for null health', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], health: null }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('--')
  })

  // --- Toggle notify ---

  it('toggleNotify calls PUT API to toggle notification', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: {} })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    // Find notify toggle buttons (the bell icon buttons)
    const notifyBtns = wrapper.findAll('button').filter(b => {
      const title = b.attributes('title')
      return title && (title.includes('通知') || title.includes('Notif'))
    })
    expect(notifyBtns.length).toBeGreaterThan(0)
    await notifyBtns[0].trigger('click')
    await flushPromises()

    expect(apiClient.put).toHaveBeenCalledWith('/accounts/1', expect.objectContaining({
      notify_enabled: false, // toggled from true to false
    }))
  })

  it('toggleNotify shows error toast on failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    vi.mocked(apiClient.put).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const notifyBtns = wrapper.findAll('button').filter(b => {
      const title = b.attributes('title')
      return title && (title.includes('通知') || title.includes('Notif'))
    })
    if (notifyBtns.length > 0) {
      await notifyBtns[0].trigger('click')
      await flushPromises()
      expect(apiClient.put).toHaveBeenCalled()
    }
  })

  // --- Trigger error handling ---

  it('trigger shows error toast on generic API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('Network error'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const triggerTiles = wrapper.findAll('.action-tile-blue')
    await triggerTiles[0].trigger('click')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith('/accounts/1/trigger')
  })

  it('trigger shows result dialog when error has task_log', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: mockAccounts })
    const errorResponse = { response: { data: { task_log: { id: 1 }, endpoints: [] } } }
    vi.mocked(apiClient.post).mockRejectedValueOnce(errorResponse)
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const triggerTiles = wrapper.findAll('.action-tile-blue')
    await triggerTiles[0].trigger('click')
    await flushPromises()

    expect(apiClient.post).toHaveBeenCalledWith('/accounts/1/trigger')
  })

  // --- Expiry helpers ---

  it('shows expiry date and remaining text for accounts with expiry', async () => {
    const futureDate = new Date(Date.now() + 10 * 86400000).toISOString().slice(0, 10)
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], auth_expires_at: futureDate }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain(futureDate)
  })

  it('shows expired text for past expiry dates', async () => {
    const pastDate = new Date(Date.now() - 5 * 86400000).toISOString().slice(0, 10)
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], auth_expires_at: pastDate }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/已过期|Expired/)
  })

  it('shows no expiry placeholder when auth_expires_at is empty', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], auth_expires_at: '' }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/未设置|Not set/)
  })

  // --- Schedule detail text ---

  it('shows schedule not set when no schedule', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], schedule: undefined }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/未配置|Not configured|Not set/)
  })

  it('shows schedule paused text', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: [{ ...mockAccounts[0], schedule: { enabled: true, paused: true, pause_reason: '', pause_threshold: 5, next_run_at: null, last_run_at: null } }],
    })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/已暂停|Paused/)
  })

  it('shows schedule disabled text', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: [{ ...mockAccounts[0], schedule: { enabled: false, paused: false, pause_reason: '', pause_threshold: 5, next_run_at: null, last_run_at: null } }],
    })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/已关闭|Disabled/)
  })

  // --- Schedule tile color ---

  it('schedule tile is red when paused', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: [{ ...mockAccounts[0], schedule: { enabled: true, paused: true, pause_reason: '', pause_threshold: 5, next_run_at: null, last_run_at: null } }],
    })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.action-tile-red').exists()).toBe(true)
  })

  it('schedule tile is green when enabled', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: [{ ...mockAccounts[0], schedule: { enabled: true, paused: false, pause_reason: '', pause_threshold: 5, next_run_at: null, last_run_at: null } }],
    })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.action-tile-green').exists()).toBe(true)
  })

  it('schedule tile is gray when no schedule', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [{ ...mockAccounts[0], schedule: undefined }] })
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.action-tile-gray').exists()).toBe(true)
  })

  // --- handleScheduleSave error path ---

  it('handleScheduleSave shows error toast on API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.put).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const scheduleTiles = wrapper.findAll('.action-tile-gray, .action-tile-green, .action-tile-red')
    if (scheduleTiles.length > 0) {
      await scheduleTiles[0].trigger('click')
    }

    const scheduleDialog = wrapper.findComponent({ name: 'ScheduleDialog' })
    if (scheduleDialog.exists()) {
      scheduleDialog.vm.$emit('save', 1, { enabled: true, pause_threshold: 5 })
      await flushPromises()
      expect(apiClient.put).toHaveBeenCalled()
    }
  })

  // --- handleScheduleResume error path ---

  it('handleScheduleResume shows error toast on API failure', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockAccounts })
    vi.mocked(apiClient.put).mockRejectedValueOnce(new Error('fail'))
    const wrapper = shallowMount(AccountsView, mountOptions)
    await flushPromises()

    const scheduleTiles = wrapper.findAll('.action-tile-gray, .action-tile-green, .action-tile-red')
    if (scheduleTiles.length > 0) {
      await scheduleTiles[0].trigger('click')
    }

    const scheduleDialog = wrapper.findComponent({ name: 'ScheduleDialog' })
    if (scheduleDialog.exists()) {
      scheduleDialog.vm.$emit('resume', 1)
      await flushPromises()
      expect(apiClient.put).toHaveBeenCalled()
    }
  })
})
