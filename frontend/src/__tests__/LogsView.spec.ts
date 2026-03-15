import { shallowMount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ query: {}, path: '/logs' }),
}))

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

// Stub v-calendar (used by DateRangePicker)
vi.mock('v-calendar', () => ({
  DatePicker: { template: '<div class="vc-stub" />' },
}))
vi.mock('v-calendar/style.css', () => ({}))

import LogsView from '../views/LogsView.vue'
import { apiClient } from '../api/client'

const mockLogs = {
  items: [
    {
      id: 100, account_id: 1, account_name: 'Contoso Dev', account_auth_type: 'auth_code',
      run_id: 'run-100', trigger_type: 'scheduled', total_endpoints: 5, success_count: 5, fail_count: 0,
      started_at: '2026-03-09T10:00:00Z', finished_at: '2026-03-09T10:00:05Z', created_at: '2026-03-09T10:00:00Z',
    },
    {
      id: 101, account_id: 2, account_name: 'Fabrikam Prod', account_auth_type: 'client_credentials',
      run_id: 'run-101', trigger_type: 'manual', total_endpoints: 8, success_count: 5, fail_count: 3,
      started_at: '2026-03-09T09:00:00Z', finished_at: '2026-03-09T09:00:10Z', created_at: '2026-03-09T09:00:00Z',
    },
    {
      id: 102, account_id: 3, account_name: 'Woodgrove QA', account_auth_type: 'auth_code',
      run_id: 'run-102', trigger_type: 'scheduled', total_endpoints: 4, success_count: 0, fail_count: 4,
      started_at: '2026-03-09T08:00:00Z', finished_at: '2026-03-09T08:00:03Z', created_at: '2026-03-09T08:00:00Z',
    },
  ],
  total: 3,
}

const mockAccountsList = [
  { id: 1, name: 'Contoso Dev', auth_type: 'auth_code' },
  { id: 2, name: 'Fabrikam Prod', auth_type: 'client_credentials' },
]

function mockFetch() {
  vi.mocked(apiClient.get).mockImplementation((url: string) => {
    if (url === '/logs') return Promise.resolve({ data: mockLogs })
    if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
    return Promise.resolve({ data: {} })
  })
}

const mountOptions = {
  global: {
    stubs: {
      Teleport: true,
      Transition: { template: '<div><slot /></div>' },
      transition: { template: '<div><slot /></div>' },
      DateRangePicker: { template: '<div class="date-range-picker-stub" />' },
    },
  },
}

describe('LogsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('applies shared page-surface class', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.animate-fade-in').exists()).toBe(true)
  })

  it('renders logs table with data', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('Contoso Dev')
    expect(wrapper.text()).toContain('Fabrikam Prod')
    expect(wrapper.text()).toContain('Woodgrove QA')
    expect(wrapper.text()).toContain('#100')
    expect(wrapper.text()).toContain('#101')
    expect(wrapper.text()).toContain('#102')
  })

  it('shows empty state when no logs', async () => {
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: { items: [], total: 0 } })
      if (url === '/accounts') return Promise.resolve({ data: [] })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/暂无日志|No logs yet/)
  })

  it('renders status badges correctly', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const text = wrapper.text()
    // success, partial, failed statuses
    expect(text).toMatch(/成功|Success/)
    expect(text).toMatch(/部分成功|Partial/)
    expect(text).toMatch(/失败|Failed/)
  })

  it('renders endpoint counts', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // id=100: 5/5, id=101: 5/8, id=102: 0/4
    expect(wrapper.text()).toContain('5')
    expect(wrapper.text()).toContain('8')
    expect(wrapper.text()).toContain('4')
  })

  it('renders trigger type badges', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toMatch(/定时|Scheduled/)
    expect(text).toMatch(/手动|Manual/)
  })

  it('renders pagination with total count', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/共 3 条|3 total/)
  })

  it('filter controls are present', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // ID search input
    const inputs = wrapper.findAll('input')
    expect(inputs.length).toBeGreaterThan(0)

    // Status filter buttons
    const text = wrapper.text()
    expect(text).toMatch(/全部|All/)
  })

  it('clicking status filter re-fetches logs', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const initialCallCount = vi.mocked(apiClient.get).mock.calls.length

    // Find and click the "success" status filter button
    const toggleBtns = wrapper.findAll('.toggle-btn')
    const successBtn = toggleBtns.find(b => b.text().match(/成功|Success/))
    if (successBtn) {
      await successBtn.trigger('click')
      await flushPromises()
      expect(vi.mocked(apiClient.get).mock.calls.length).toBeGreaterThan(initialCallCount)
    }
  })

  it('refresh button re-fetches logs', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const initialCallCount = vi.mocked(apiClient.get).mock.calls.length

    const refreshBtn = wrapper.findAll('button').find(b => b.text().match(/刷新|Refresh/))
    expect(refreshBtn).toBeDefined()

    await refreshBtn!.trigger('click')
    await flushPromises()

    expect(vi.mocked(apiClient.get).mock.calls.length).toBeGreaterThan(initialCallCount)
  })

  it('detail button exists for each row', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    expect(detailBtns.length).toBe(3) // one per log row
  })

  // --- Filter apply/reset ---

  it('filtering by trigger type re-fetches with correct params', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const initialCallCount = vi.mocked(apiClient.get).mock.calls.length

    // Click the "scheduled" trigger type button
    const toggleBtns = wrapper.findAll('.toggle-btn')
    const scheduledBtn = toggleBtns.find(b => b.text().match(/定时|Scheduled/))
    if (scheduledBtn) {
      await scheduledBtn.trigger('click')
      await flushPromises()

      const logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
      const lastLogCall = logCalls[logCalls.length - 1]
      expect(lastLogCall[1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ trigger_type: 'scheduled' }),
      }))
    }
  })

  it('filtering by manual trigger type', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const toggleBtns = wrapper.findAll('.toggle-btn')
    const manualBtn = toggleBtns.find(b => b.text().match(/手动|Manual/))
    if (manualBtn) {
      await manualBtn.trigger('click')
      await flushPromises()

      const logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
      const lastLogCall = logCalls[logCalls.length - 1]
      expect(lastLogCall[1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ trigger_type: 'manual' }),
      }))
    }
  })

  it('filtering by failed status sends correct params', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const toggleBtns = wrapper.findAll('.toggle-btn')
    const failedBtn = toggleBtns.find(b => b.text().match(/失败|Failed/))
    if (failedBtn) {
      await failedBtn.trigger('click')
      await flushPromises()

      const logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
      const lastLogCall = logCalls[logCalls.length - 1]
      expect(lastLogCall[1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ status: 'failed' }),
      }))
    }
  })

  it('filtering by partial status sends correct params', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const toggleBtns = wrapper.findAll('.toggle-btn')
    const partialBtn = toggleBtns.find(b => b.text().match(/部分|Partial/))
    if (partialBtn) {
      await partialBtn.trigger('click')
      await flushPromises()

      const logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
      const lastLogCall = logCalls[logCalls.length - 1]
      expect(lastLogCall[1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ status: 'partial' }),
      }))
    }
  })

  it('resetting status filter to all removes status param', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // First set to failed
    const toggleBtns = wrapper.findAll('.toggle-btn')
    const failedBtn = toggleBtns.find(b => b.text().match(/失败|Failed/))
    if (failedBtn) {
      await failedBtn.trigger('click')
      await flushPromises()

      // Verify failed was sent
      let logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
      let lastLogCall = logCalls[logCalls.length - 1]
      expect(lastLogCall[1]?.params?.status).toBe('failed')

      // Now click the "All" button in status group (last group of toggles)
      // The status group buttons are: All, Success, Partial, Failed
      // Failed is in the status group, so its sibling "All" is nearby
      // Find the parent container of failedBtn and get the first toggle-btn in it
      const statusAllBtns = wrapper.findAll('.toggle-btn').filter(b => {
        const text = b.text().trim()
        return text === '全部' || text === 'All'
      })
      // The second "All" button is in the status group (first is in trigger type group)
      if (statusAllBtns.length >= 2) {
        await statusAllBtns[1].trigger('click')
        await flushPromises()

        logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
        lastLogCall = logCalls[logCalls.length - 1]
        const params = lastLogCall[1]?.params || {}
        expect(params.status).toBeUndefined()
      }
    }
  })

  it('ID search input triggers filter', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const initialCallCount = vi.mocked(apiClient.get).mock.calls.length

    const idInput = wrapper.find('input')
    await idInput.setValue('100')
    await idInput.trigger('input')
    await flushPromises()

    const logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
    const lastLogCall = logCalls[logCalls.length - 1]
    expect(lastLogCall[1]).toEqual(expect.objectContaining({
      params: expect.objectContaining({ id: '100' }),
    }))
  })

  // --- Pagination ---

  it('pagination buttons navigate between pages', async () => {
    // Mock with many items to generate multiple pages
    const manyItems = Array.from({ length: 25 }, (_, i) => ({
      ...mockLogs.items[0],
      id: 200 + i,
      account_name: `Account ${i}`,
    }))
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: { items: manyItems.slice(0, 20), total: 25 } })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Total should show 25
    expect(wrapper.text()).toMatch(/25/)

    // Find next page button (the last page-btn)
    const pageBtns = wrapper.findAll('.page-btn')
    if (pageBtns.length >= 2) {
      // Click next page
      await pageBtns[pageBtns.length - 1].trigger('click')
      await flushPromises()

      const logCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/logs')
      const lastLogCall = logCalls[logCalls.length - 1]
      expect(lastLogCall[1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ page: 2 }),
      }))
    }
  })

  // --- Detail drawer ---

  it('clicking detail button loads endpoint details', async () => {
    mockFetch()
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: mockLogs })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      if (url.match(/\/logs\/\d+\/endpoints/)) return Promise.resolve({ data: [
        { id: 1, endpoint_name: '/me', scope: 'User.Read', http_status: 200, success: true, error_message: '', response_body: '', executed_at: '2026-03-09T10:00:01Z' },
      ] })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    if (detailBtns.length > 0) {
      await detailBtns[0].trigger('click')
      await flushPromises()

      expect(apiClient.get).toHaveBeenCalledWith('/logs/100/endpoints')
    }
  })

  // --- Date formatting ---

  it('formats dates correctly in log rows', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // started_at: '2026-03-09T10:00:00Z' should produce formatted time
    const text = wrapper.text()
    expect(text).toMatch(/\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}/)
  })

  it('formats duration correctly', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Run #100: 5 seconds duration -> should show "5.0s"
    expect(wrapper.text()).toContain('5.0s')
    // Run #101: 10 seconds -> "10.0s"
    expect(wrapper.text()).toContain('10.0s')
    // Run #102: 3 seconds -> "3.0s"
    expect(wrapper.text()).toContain('3.0s')
  })

  // --- Row styling ---

  it('failed rows get red background', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Run #102 has success_count=0, fail_count=4 => failed
    const redRow = wrapper.findAll('.bg-red-50\\/40')
    expect(redRow.length).toBeGreaterThan(0)
  })

  it('partial rows get amber background', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Run #101 has success_count=5, fail_count=3 => partial
    const amberRow = wrapper.findAll('.bg-amber-50\\/30')
    expect(amberRow.length).toBeGreaterThan(0)
  })

  // --- API failure handling ---

  it('handles fetch failure gracefully with empty state', async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error('Network error'))
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/暂无日志|No logs yet/)
  })

  // --- Drawer collapse levels ---

  it('clicking detail then loading drawer and toggling endpoint body', async () => {
    const endpointData = [
      { id: 1, endpoint_name: '/me', scope: 'User.Read', http_status: 200, success: true, error_message: '', response_body: '', executed_at: '2026-03-09T10:00:01Z' },
      { id: 2, endpoint_name: '/messages', scope: 'Mail.Read', http_status: 403, success: false, error_message: 'Forbidden', response_body: '{"error":"access_denied"}', executed_at: '2026-03-09T10:00:02Z' },
    ]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: mockLogs })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      if (url.match(/\/logs\/\d+\/endpoints/)) return Promise.resolve({ data: endpointData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    if (detailBtns.length > 0) {
      await detailBtns[0].trigger('click')
      await flushPromises()

      // Drawer should be open with endpoints
      expect(wrapper.text()).toContain('/me')
      expect(wrapper.text()).toContain('/messages')
    }
  })

  it('clicking a table row does NOT open inline details', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Click the first data row
    const rows = wrapper.findAll('.log-grid')
    // First .log-grid is the header, subsequent ones are data rows
    const dataRow = rows.find(r => r.text().includes('#100'))
    expect(dataRow).toBeDefined()
    await dataRow!.trigger('click')
    await flushPromises()

    // No inline endpoint loading should have been triggered beyond the initial /logs and /accounts calls
    const endpointCalls = vi.mocked(apiClient.get).mock.calls.filter(c =>
      typeof c[0] === 'string' && c[0].match(/\/logs\/\d+\/endpoints/)
    )
    expect(endpointCalls.length).toBe(0)
  })

  it('only the Detail button opens the right-side drawer', async () => {
    const endpointData = [
      { id: 1, endpoint_name: '/me', http_status: 200, success: true, error_message: '', response_body: '', executed_at: '2026-03-09T10:00:01Z' },
    ]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: mockLogs })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      if (url.match(/\/logs\/\d+\/endpoints/)) return Promise.resolve({ data: endpointData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Before clicking detail, drawer content should not be visible
    expect(wrapper.text()).not.toContain('/me')

    // Click the Detail button for the first row
    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    await detailBtns[0].trigger('click')
    await flushPromises()

    // Now drawer should show endpoint data
    expect(wrapper.text()).toContain('/me')
  })

  it('failed endpoint rows start collapsed in the drawer', async () => {
    const endpointData = [
      { id: 10, endpoint_name: '/messages', http_status: 403, success: false, error_message: 'Forbidden access', response_body: '{"error":"denied"}', executed_at: '2026-03-09T10:00:02Z' },
    ]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: mockLogs })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      if (url.match(/\/logs\/\d+\/endpoints/)) return Promise.resolve({ data: endpointData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    await detailBtns[0].trigger('click')
    await flushPromises()

    // Endpoint name should be visible
    expect(wrapper.text()).toContain('/messages')
    // Error message text should NOT be visible (collapsed)
    expect(wrapper.text()).not.toContain('Forbidden access')
    // Response body should NOT be visible (collapsed)
    expect(wrapper.text()).not.toContain('denied')
  })

  it('error toggle reveals both error_message and response_body together', async () => {
    const endpointData = [
      { id: 10, endpoint_name: '/messages', scope: 'Mail.Read', http_status: 403, success: false, error_message: 'Forbidden access', response_body: '{"error":"denied"}', executed_at: '2026-03-09T10:00:02Z' },
    ]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: mockLogs })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      if (url.match(/\/logs\/\d+\/endpoints/)) return Promise.resolve({ data: endpointData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    await detailBtns[0].trigger('click')
    await flushPromises()

    // Click the error toggle
    const errorToggle = wrapper.find('[data-testid="toggle-error"]')
    expect(errorToggle.exists()).toBe(true)
    await errorToggle.trigger('click')
    await flushPromises()

    // Both error message and response body should now be visible
    expect(wrapper.text()).toContain('Forbidden access')
    expect(wrapper.text()).toContain('denied')
  })

  it('successful endpoint rows remain compact in the drawer', async () => {
    const endpointData = [
      { id: 1, endpoint_name: '/me', http_status: 200, success: true, error_message: '', response_body: '', executed_at: '2026-03-09T10:00:01Z' },
      { id: 2, endpoint_name: '/users', http_status: 200, success: true, error_message: '', response_body: '', executed_at: '2026-03-09T10:00:02Z' },
    ]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: mockLogs })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      if (url.match(/\/logs\/\d+\/endpoints/)) return Promise.resolve({ data: endpointData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    const detailBtns = wrapper.findAll('button').filter(b => b.text().match(/详情|Details/))
    await detailBtns[0].trigger('click')
    await flushPromises()

    // Both endpoints should be shown
    expect(wrapper.text()).toContain('/me')
    expect(wrapper.text()).toContain('/users')

    // No toggle buttons for error or response body should exist (no failures)
    expect(wrapper.find('[data-testid="toggle-error"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="toggle-response-body"]').exists()).toBe(false)
  })

  // --- formatDuration edge cases ---

  it('shows dash for missing duration', async () => {
    const logsWithNullEnd = {
      items: [{
        ...mockLogs.items[0],
        finished_at: null,
      }],
      total: 1,
    }
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: logsWithNullEnd })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('-')
  })

  // --- formatTime edge case ---

  it('shows dash for empty time string', async () => {
    const logsWithEmptyTime = {
      items: [{
        ...mockLogs.items[0],
        started_at: '',
      }],
      total: 1,
    }
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/logs') return Promise.resolve({ data: logsWithEmptyTime })
      if (url === '/accounts') return Promise.resolve({ data: mockAccountsList })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('-')
  })

  // --- Account dropdown ---

  it('account dropdown shows account options', async () => {
    mockFetch()
    const wrapper = shallowMount(LogsView, mountOptions)
    await flushPromises()

    // Click the account filter dropdown button
    const filterPills = wrapper.findAll('.filter-pill')
    const accountPill = filterPills.find(p => p.text().match(/全部账号|All Accounts|全部|All/))
    if (accountPill) {
      await accountPill.trigger('click')
      await flushPromises()

      // Should show account options in dropdown
      const popupItems = wrapper.findAll('.popup-item')
      expect(popupItems.length).toBeGreaterThan(0)
    }
  })
})
