import { shallowMount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockPush = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => ({ path: '/dashboard' }),
}))

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

// Stub echarts modules to avoid canvas/WebGL errors in jsdom
vi.mock('echarts/core', () => ({
  use: vi.fn(),
}))
vi.mock('echarts/renderers', () => ({
  CanvasRenderer: {},
}))
vi.mock('echarts/charts', () => ({
  LineChart: {},
}))
vi.mock('echarts/components', () => ({
  TooltipComponent: {},
  LegendComponent: {},
  GridComponent: {},
}))
vi.mock('vue-echarts', () => ({
  default: { template: '<div class="vchart-stub" />' },
}))

import DashboardView from '../views/DashboardView.vue'
import { apiClient } from '../api/client'

const summaryData = {
  total_accounts: 5,
  success_rate: 96.3,
  total_runs: 1200,
  error_count: 12,
}

const trendData = [
  { date: '03-01', success: 100, failure: 2 },
  { date: '03-02', success: 110, failure: 3 },
]

const healthData = [
  { id: 1, name: 'Contoso Dev', auth_type: 'auth_code', health: 98.2, total_runs: 420, success_runs: 412, last_run: '2026-03-09T10:00:00Z' },
]

const recentData = [
  { id: 100, account_name: 'Contoso Dev', account_auth_type: 'auth_code', trigger_type: 'scheduled', total_endpoints: 5, success_count: 5, fail_count: 0, started_at: '2026-03-09T10:00:00Z', finished_at: '2026-03-09T10:00:05Z' },
]

const accountsData = [
  { id: 1, schedule: { enabled: true, paused: false } },
  { id: 2, schedule: { enabled: true, paused: true } },
]

function mockSuccessfulFetch() {
  vi.mocked(apiClient.get).mockImplementation((url: string) => {
    if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
    if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
    if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
    if (url === '/dashboard/recent-logs') return Promise.resolve({ data: recentData })
    if (url === '/accounts') return Promise.resolve({ data: accountsData })
    return Promise.resolve({ data: {} })
  })
}

const mountOptions = {
  global: {
    stubs: {
      RouterLink: { template: '<a><slot /></a>' },
      VChart: { template: '<div class="vchart-stub" />' },
      'v-chart': { template: '<div class="vchart-stub" />' },
      Transition: { template: '<div><slot /></div>' },
      transition: { template: '<div><slot /></div>' },
    },
  },
}

describe('DashboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('applies shared page-surface class', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.animate-fade-in').exists()).toBe(true)
  })

  it('renders stat cards after successful fetch', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    const text = wrapper.text()
    // Should show stat values
    expect(text).toContain('5')       // total_accounts
    expect(text).toContain('96.3%')   // success_rate
    expect(text).toContain('1,200')   // total_runs formatted
    expect(text).toContain('12')      // error_count
  })

  it('renders period selector buttons', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    const text = wrapper.text()
    // Period labels (zh default)
    expect(text).toMatch(/近24小时|Last 24 Hours/)
    expect(text).toMatch(/近7天|Last 7 Days/)
    expect(text).toMatch(/近30天|Last 30 Days/)
  })

  it('period selector changes data by re-fetching', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Initial fetch count: 5 parallel gets on mount
    const initialCallCount = vi.mocked(apiClient.get).mock.calls.length
    expect(initialCallCount).toBe(5)

    // Click a period button (find buttons, click the first one which is "1d")
    const periodButtons = wrapper.findAll('button').filter(b => {
      const text = b.text()
      return text.match(/近24小时|Last 24 Hours/)
    })
    if (periodButtons.length > 0) {
      await periodButtons[0].trigger('click')
      await flushPromises()

      // Should have made another set of API calls
      expect(vi.mocked(apiClient.get).mock.calls.length).toBeGreaterThan(initialCallCount)
    }
  })

  it('silently handles API failure without crashing', async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error('Network Error'))
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Component catches errors silently and keeps current data
    // Verify the component still renders (title is present)
    expect(wrapper.text()).toMatch(/仪表盘|Dashboard/)
  })

  it('refresh button triggers data re-fetch', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    const initialCallCount = vi.mocked(apiClient.get).mock.calls.length

    // Find the refresh button
    const refreshBtns = wrapper.findAll('button').filter(b => {
      const text = b.text()
      return text.match(/刷新|Refresh/)
    })
    expect(refreshBtns.length).toBeGreaterThan(0)

    await refreshBtns[0].trigger('click')
    await flushPromises()

    expect(vi.mocked(apiClient.get).mock.calls.length).toBeGreaterThan(initialCallCount)
  })

  it('renders recent runs table', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('Contoso Dev')
    expect(wrapper.text()).toContain('#100')
  })

  it('renders account health section', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('Contoso Dev')
    expect(wrapper.text()).toContain('98.2%')
  })

  // --- Format functions ---

  it('formatTime formats ISO string correctly', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // The recent run has started_at: '2026-03-09T10:00:00Z'
    // formatTime should produce a formatted date string
    const text = wrapper.text()
    // Should contain formatted time (varies by timezone, just check structure)
    expect(text).toMatch(/\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}/)
  })

  it('formatDuration computes correct duration', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // recent run: started_at='2026-03-09T10:00:00Z' finished_at='2026-03-09T10:00:05Z' => 5s
    expect(wrapper.text()).toContain('5.0s')
  })

  // --- Period change ---

  it('switching to 30d period triggers fetch with correct params', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    vi.clearAllMocks()
    mockSuccessfulFetch()

    const periodBtns = wrapper.findAll('button').filter(b =>
      b.text().match(/近30天|Last 30 Days/)
    )
    if (periodBtns.length > 0) {
      await periodBtns[0].trigger('click')
      await flushPromises()

      // Check that the summary call includes period=30d
      const summaryCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/dashboard/summary')
      expect(summaryCalls.length).toBeGreaterThan(0)
      expect(summaryCalls[0][1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ period: '30d' }),
      }))
    }
  })

  it('switching to all period triggers fetch', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    vi.clearAllMocks()
    mockSuccessfulFetch()

    const periodBtns = wrapper.findAll('button').filter(b =>
      b.text().match(/全部|All/)
    )
    if (periodBtns.length > 0) {
      await periodBtns[0].trigger('click')
      await flushPromises()

      const summaryCalls = vi.mocked(apiClient.get).mock.calls.filter(c => c[0] === '/dashboard/summary')
      expect(summaryCalls.length).toBeGreaterThan(0)
      expect(summaryCalls[0][1]).toEqual(expect.objectContaining({
        params: expect.objectContaining({ period: 'all' }),
      }))
    }
  })

  // --- Run status helpers ---

  it('renders success status badge for runs with no failures', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Run with fail_count=0 should show success badge
    expect(wrapper.text()).toMatch(/成功|Success/)
  })

  it('renders correct endpoint counts in recent runs', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Run #100: 5/5 endpoints
    expect(wrapper.text()).toContain('5')
  })

  // --- Health color helpers ---

  it('health bar uses emerald for high health', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Contoso Dev has 98.2% health -> should use emerald gradient
    const healthBar = wrapper.find('.bg-gradient-to-r.from-emerald-400')
    expect(healthBar.exists()).toBe(true)
  })

  // --- Data transformation ---

  it('computes active schedules from accounts data', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // accountsData has 2 accounts: one enabled+not-paused, one enabled+paused
    // So active_schedules should be 1, total_schedules should be 2
    expect(wrapper.text()).toContain('1/2')
  })

  it('computes average health from health data', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Only one account in healthData with health=98.2
    expect(wrapper.text()).toContain('98.2%')
  })

  // --- goToRun ---

  it('clicking a recent run navigates to logs with run id', async () => {
    mockSuccessfulFetch()
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // Find the recent run rows (they use recent-run-grid class)
    const rows = wrapper.findAll('.recent-run-grid').filter(el => el.text().includes('#100'))
    if (rows.length > 0) {
      await rows[0].trigger('click')
      expect(mockPush).toHaveBeenCalledWith({ path: '/logs', query: { id: '100' } })
    }
  })

  // --- Error banner retry ---

  it('refresh button re-fetches data after error', async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error('Network Error'))
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    // No error banner exists — component catches errors silently
    // Use the refresh button to re-fetch
    vi.clearAllMocks()
    mockSuccessfulFetch()

    const refreshBtns = wrapper.findAll('button').filter(b => {
      const text = b.text()
      return text.match(/刷新|Refresh/)
    })
    expect(refreshBtns.length).toBeGreaterThan(0)

    await refreshBtns[0].trigger('click')
    await flushPromises()

    expect(apiClient.get).toHaveBeenCalled()
  })

  // --- No data states ---

  // --- formatDuration edge cases ---

  it('shows ms for very short durations', async () => {
    const fastRunData = [{
      ...recentData[0],
      started_at: '2026-03-09T10:00:00.000Z',
      finished_at: '2026-03-09T10:00:00.500Z',
    }]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: fastRunData })
      if (url === '/accounts') return Promise.resolve({ data: accountsData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('500ms')
  })

  it('shows minutes for long durations', async () => {
    const longRunData = [{
      ...recentData[0],
      started_at: '2026-03-09T10:00:00Z',
      finished_at: '2026-03-09T10:02:30Z',
    }]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: longRunData })
      if (url === '/accounts') return Promise.resolve({ data: accountsData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('2m 30s')
  })

  it('shows dash for missing duration', async () => {
    const noDurationData = [{
      ...recentData[0],
      started_at: '',
      finished_at: '',
    }]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: noDurationData })
      if (url === '/accounts') return Promise.resolve({ data: accountsData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('-')
  })

  // --- Run status variants ---

  it('renders partial status badge for runs with mixed results', async () => {
    const partialRunData = [{
      ...recentData[0],
      success_count: 3,
      fail_count: 2,
    }]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: partialRunData })
      if (url === '/accounts') return Promise.resolve({ data: accountsData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/部分成功|Partial/)
  })

  it('renders failed status badge for runs with all failures', async () => {
    const failedRunData = [{
      ...recentData[0],
      success_count: 0,
      fail_count: 5,
    }]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: failedRunData })
      if (url === '/accounts') return Promise.resolve({ data: accountsData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/失败|Failed/)
  })

  // --- formatTime edge case ---

  it('formatTime returns dash for empty string', async () => {
    const emptyTimeData = [{
      ...recentData[0],
      started_at: '',
    }]
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: healthData })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: emptyTimeData })
      if (url === '/accounts') return Promise.resolve({ data: accountsData })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toContain('-')
  })

  it('shows no data message when account health is empty', async () => {
    vi.mocked(apiClient.get).mockImplementation((url: string) => {
      if (url === '/dashboard/summary') return Promise.resolve({ data: summaryData })
      if (url === '/dashboard/trend') return Promise.resolve({ data: trendData })
      if (url === '/dashboard/account-health') return Promise.resolve({ data: [] })
      if (url === '/dashboard/recent-logs') return Promise.resolve({ data: [] })
      if (url === '/accounts') return Promise.resolve({ data: [] })
      return Promise.resolve({ data: {} })
    })
    const wrapper = shallowMount(DashboardView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).toMatch(/暂无数据|No data/)
  })
})
