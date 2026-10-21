import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from './i18n'
import { App } from './App'
import { APIError, loadSession, request, type Account, type Run } from './api'
import { ErrorBoundary } from './components/error-boundary'

function requestURL(input: RequestInfo | URL): string {
  return typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.href
      : input.url
}
function requestBody(options?: RequestInit): Record<string, unknown> {
  if (typeof options?.body !== 'string') {
    throw new Error('Expected a JSON request body')
  }
  return JSON.parse(options.body) as Record<string, unknown>
}

const account: Account = {
  id: 1,
  name: 'Example account',
  auth_type: 'delegated',
  client_id: '7dfd26ad-11d1-43df-aae8-77c78347a619',
  tenant_id: 'common',
  enabled: false,
  min_minutes: 120,
  max_minutes: 240,
  next_run_at: null,
  last_run_at: null,
  last_error: '',
  last_error_code: '',
  revision: 1,
  has_client_secret: true,
  has_refresh_token: true,
}
let authenticated: boolean
let lastUpdate: Record<string, unknown> | undefined
let failAccounts: boolean

afterEach(() => document.querySelector('base')?.remove())

beforeEach(async () => {
  authenticated = false
  lastUpdate = undefined
  failAccounts = false
  window.history.replaceState({}, '', '/login')
  await i18n.changeLanguage('zh')
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
      const path = requestURL(input)
      const json = (value: unknown, status = 200) =>
        new Response(JSON.stringify(value), {
          status,
          headers: { 'Content-Type': 'application/json' },
        })
      if (path === '/api/info') {
        return json({
          callback_url: 'http://127.0.0.1:8080/api/oauth/callback',
        })
      }
      if (
        path === '/api/accounts/1' &&
        (!options?.method || options.method === 'GET')
      ) {
        return json(account)
      }
      if (path === '/api/session') {
        return json({ authenticated, csrf: authenticated ? 'csrf-token' : '' })
      }
      if (path === '/api/login') {
        authenticated = true
        return json({ authenticated: true })
      }
      if (path === '/api/logout') {
        authenticated = false
        return json({ authenticated: false })
      }
      if (path === '/api/accounts/1' && options?.method === 'PUT') {
        lastUpdate = requestBody(options)
        return json({ ...account, ...lastUpdate })
      }
      if (path === '/api/accounts') {
        return failAccounts
          ? json(
              {
                code: 'storage_failed',
                error: 'Diagnostic text may change independently.',
              },
              500,
            )
          : json({
              accounts: [account],
              busy: false,
            })
      }
      if (path.startsWith('/api/runs')) {
        return json({ runs: [], busy: false, total: 0, page: 1, page_size: 25 })
      }
      throw new Error('Unexpected request: ' + path)
    }),
  )
})

describe('management interface', () => {
  it('explains the login limit and allows a later successful retry', async () => {
    const originalFetch = fetch
    vi.stubGlobal(
      'fetch',
      vi
        .fn<typeof fetch>()
        .mockImplementationOnce(originalFetch)
        .mockImplementationOnce(() =>
          Promise.resolve(
            new Response(JSON.stringify({ code: 'login_rate_limited' }), {
              status: 429,
              headers: {
                'Retry-After': '12',
                'Content-Type': 'application/json',
              },
            }),
          ),
        )
        .mockImplementation(originalFetch),
    )
    const user = userEvent.setup()
    render(<App />)
    await user.type(
      await screen.findByLabelText('管理密码'),
      'an-administrator-password',
    )
    await user.click(screen.getByRole('button', { name: '登录' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '登录尝试过于频繁，请在 12 秒后重试',
    )
    expect(screen.getByRole('button', { name: '登录' })).toBeEnabled()
    expect(window.location.pathname).toBe('/login')

    await user.click(screen.getByRole('button', { name: '登录' }))
    await screen.findByRole('heading', { name: '账户' })
  })

  it('keeps login, navigation, reloads, and API requests under a base path', async () => {
    const prefix = '/tools/e5'
    const base = document.createElement('base')
    base.href = prefix + '/'
    document.head.append(base)
    window.history.replaceState({}, '', prefix + '/login')
    const originalFetch = globalThis.fetch
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL, options?: RequestInit) => {
        const path = requestURL(input)
        expect(path.startsWith(prefix + '/api/')).toBe(true)
        return originalFetch(path.slice(prefix.length), options)
      }),
    )
    const user = userEvent.setup()
    const page = render(<App />)
    await screen.findByRole('heading', { name: '管理员登录' })
    await user.type(
      screen.getByLabelText('管理密码'),
      'an-administrator-password',
    )
    await user.click(screen.getByRole('button', { name: '登录' }))
    await screen.findByRole('heading', { name: '账户' })
    expect(window.location.pathname).toBe(prefix + '/accounts')
    expect(screen.getByRole('link', { name: '跳转到主内容' })).toHaveAttribute(
      'href',
      prefix + '/accounts#main-content',
    )
    await user.click(screen.getByRole('link', { name: '执行记录' }))
    await act(() => vi.dynamicImportSettled())
    await screen.findByRole('heading', { name: '执行记录' })
    expect(window.location.pathname).toBe(prefix + '/runs')
    page.unmount()
    render(<App />)
    await screen.findByRole('heading', { name: '执行记录' })
    expect(window.location.pathname).toBe(prefix + '/runs')
    await user.click(screen.getByRole('button', { name: '导航菜单' }))
    await user.click(screen.getByRole('menuitem', { name: '退出登录' }))
    await screen.findByRole('heading', { name: '管理员登录' })
    expect(window.location.pathname).toBe(prefix + '/login')
  })

  it('signs in and switches between Chinese and English', async () => {
    const user = userEvent.setup()
    render(<App />)
    await screen.findByRole('heading', { name: '管理员登录' })
    await user.type(
      screen.getByLabelText('管理密码'),
      'an-administrator-password',
    )
    await user.click(screen.getByRole('button', { name: '登录' }))
    await screen.findByRole('heading', { name: '账户' })
    expect(await screen.findByText('Example account')).toBeVisible()
    await user.click(screen.getByRole('button', { name: '导航菜单' }))
    await user.click(screen.getByRole('menuitem', { name: /^切换语言/ }))
    expect(
      await screen.findByRole('heading', { name: 'Accounts' }),
    ).toBeVisible()
    expect(document.documentElement.lang).toBe('en')
  })

  it('edits account metadata without submitting stored credentials', async () => {
    authenticated = true
    window.history.replaceState({}, '', '/accounts')
    const user = userEvent.setup()
    render(<App />)
    await user.click(
      await screen.findByRole('button', { name: '更多操作 Example account' }),
    )
    await user.click(screen.getByRole('menuitem', { name: '编辑' }))
    expect(screen.getByLabelText('Client Secret')).toHaveValue('')
    fireEvent.change(screen.getByLabelText('账户名称'), {
      target: { value: 'Renamed account' },
    })
    await user.click(screen.getByRole('button', { name: '保存账户' }))
    await waitFor(() => expect(lastUpdate?.name).toBe('Renamed account'))
    expect(lastUpdate).not.toHaveProperty('client_secret')
    expect(lastUpdate).not.toHaveProperty('refresh_token')
    expect(lastUpdate?.revision).toBe(1)
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: '更多操作 Example account' }),
      ).toHaveFocus(),
    )
  })

  it('restores focus after cancelling account deletion', async () => {
    authenticated = true
    window.history.replaceState({}, '', '/accounts')
    const user = userEvent.setup()
    render(<App />)
    const trigger = await screen.findByRole('button', {
      name: '更多操作 Example account',
    })
    await user.click(trigger)
    await user.click(screen.getByRole('menuitem', { name: '删除' }))
    await screen.findByRole('alertdialog')
    await user.click(screen.getByRole('button', { name: '取消' }))
    await waitFor(() => expect(trigger).toHaveFocus())
  })

  it('shows a localized fetch failure instead of an empty success state', async () => {
    authenticated = true
    failAccounts = true
    window.history.replaceState({}, '', '/accounts')
    render(<App />)
    expect(await screen.findByRole('alert')).toHaveTextContent('数据操作失败')
    expect(screen.queryByText('尚未添加账户')).not.toBeInTheDocument()
  })

  it('recovers an expired mutation session without retrying the operation', async () => {
    const unauthorized = vi.fn()
    window.addEventListener('e5:unauthorized', unauthorized)
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ code: 'csrf_invalid', error: 'Session expired.' }),
            { status: 403 },
          ),
      ),
    )
    await expect(request('/accounts/1/run', 'POST', {})).rejects.toBeInstanceOf(
      APIError,
    )
    expect(unauthorized).toHaveBeenCalledOnce()
    expect(fetch).toHaveBeenCalledOnce()
    window.removeEventListener('e5:unauthorized', unauthorized)
  })

  it('offers a localized reload action when a page fails to render', async () => {
    const log = vi.spyOn(console, 'error').mockImplementation(() => {})
    function BrokenPage(): never {
      throw new Error('A route module could not load')
    }
    render(
      <ErrorBoundary>
        <BrokenPage />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('无法加载此页面')
    expect(screen.getByRole('button', { name: '重新加载页面' })).toBeVisible()
    log.mockRestore()
  })

  it('uses a localized fallback for unknown server errors', async () => {
    authenticated = true
    const originalFetch = globalThis.fetch
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL, options?: RequestInit) =>
        requestURL(input) === '/api/accounts'
          ? Promise.resolve(
              new Response(
                JSON.stringify({
                  code: 'future_error',
                  error: 'Internal diagnostic.',
                }),
                { status: 500 },
              ),
            )
          : originalFetch(input, options),
      ),
    )
    window.history.replaceState({}, '', '/accounts')
    render(<App />)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '请求失败，请重试。',
    )
    expect(screen.queryByText('Internal diagnostic.')).not.toBeInTheDocument()
  })

  it('shows a spinner while authentication state is loading', async () => {
    let finish!: (response: Response) => void
    const pending = new Promise<Response>((resolve) => {
      finish = resolve
    })
    const originalFetch = globalThis.fetch
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL, options?: RequestInit) =>
        requestURL(input) === '/api/session'
          ? pending
          : originalFetch(input, options),
      ),
    )
    render(<App />)
    const status = screen.getByRole('status')
    expect(status).toHaveTextContent('正在加载…')
    expect(status.querySelector('svg')).toBeInTheDocument()
    finish(
      new Response(JSON.stringify({ authenticated: false, csrf: '' }), {
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await screen.findByRole('heading', { name: '管理员登录' })
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('switches authorization with the styled selector and resets incompatible fields', async () => {
    authenticated = true
    window.history.replaceState({}, '', '/accounts')
    const user = userEvent.setup()
    render(<App />)
    await user.click(
      await screen.findByRole('button', { name: '更多操作 Example account' }),
    )
    await user.click(screen.getByRole('menuitem', { name: '编辑' }))
    expect(
      screen.getByText(i18n.t('ui.delegated_permissions_hint')),
    ).toBeVisible()
    expect(screen.getByText(i18n.t('ui.random_execution_hint'))).toBeVisible()
    await user.click(screen.getByRole('combobox', { name: '授权方式' }))
    await user.click(
      screen.getByRole('option', { name: '应用授权 · 客户端凭据' }),
    )
    expect(screen.getByLabelText('Tenant ID')).toHaveValue('')
    expect(
      screen.queryByLabelText('Refresh Token（可选）'),
    ).not.toBeInTheDocument()
    expect(screen.queryByText('Graph 接口')).not.toBeInTheDocument()
    expect(
      screen.getByText(i18n.t('ui.application_permissions_hint')),
    ).toBeVisible()
  })

  it('opens account creation from the toolbar and user menu', async () => {
    authenticated = true
    window.history.replaceState({}, '', '/accounts')
    const user = userEvent.setup()
    render(<App />)
    await screen.findByRole('heading', { name: '账户' })
    await user.click(screen.getByRole('button', { name: '添加账户' }))
    await screen.findByRole('heading', { name: '添加账户' })
    await user.click(screen.getByRole('button', { name: '取消' }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '添加账户' })).toHaveFocus(),
    )
    const menu = screen.getByRole('button', { name: '导航菜单' })
    await user.click(menu)
    await user.click(screen.getByRole('menuitem', { name: '添加账户' }))
    await screen.findByRole('heading', { name: '添加账户' })
    await user.click(screen.getByRole('button', { name: '取消' }))
    await waitFor(() => expect(menu).toHaveFocus())
  })
})

it('updates an open execution detail when polling completes', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/runs')
  const run: Run = {
    id: 1,
    account_id: 1,
    account_name: 'Example account',
    trigger: 'manual',
    status: 'running',
    error: '',
    error_code: '',
    request_count: 0,
    success_count: 0,
    results: [],
    started_at: '2026-10-16T00:00:00Z',
    finished_at: null,
  }
  const originalFetch = globalThis.fetch
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, options?: RequestInit) =>
      requestURL(input) === '/api/runs'
        ? Promise.resolve(
            new Response(
              JSON.stringify({
                runs: [run],
                busy: run.status === 'running',
                total: 1,
                page: 1,
                page_size: 25,
              }),
            ),
          )
        : originalFetch(input, options),
    ),
  )
  const user = userEvent.setup()
  render(<App />)
  await user.click(await screen.findByRole('button', { name: '查看' }))
  const dialog = screen.getByRole('dialog')
  expect(within(dialog).getByRole('status')).toBeInTheDocument()
  Object.assign(run, {
    status: 'failed',
    error: 'One or more Graph requests failed.',
    error_code: 'graph_requests_failed',
    finished_at: '2026-10-16T00:00:01Z',
    results: [
      {
        endpoint: 'profile',
        id: 1,
        started_at: '2026-10-16T00:00:00Z',
        duration_ms: 100,
        attempts: 1,
        http_status: 403,
        error: 'Graph returned HTTP 403.',
        error_code: 'graph_http_error',
      },
    ],
  })
  await waitFor(
    () => expect(within(dialog).getByText('HTTP 403')).toBeVisible(),
    { timeout: 6500 },
  )
  expect(within(dialog).getByText('Graph 返回了失败响应。')).toBeVisible()
  expect(within(dialog).queryByRole('status')).not.toBeInTheDocument()
  await user.keyboard('{Escape}')
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  )
  expect(screen.getByRole('button', { name: '查看' })).toHaveFocus()
  await user.click(screen.getByRole('button', { name: '查看' }))
  expect(within(screen.getByRole('dialog')).getByText('HTTP 403')).toBeVisible()
}, 10000)

it('keeps account filtering available while execution is busy and prevents overlapping actions', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/accounts')
  const originalFetch = fetch
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, options?: RequestInit) =>
      requestURL(input) === '/api/accounts'
        ? Promise.resolve(
            new Response(
              JSON.stringify({
                accounts: [
                  account,
                  { ...account, id: 2, name: 'Other account' },
                ],
                busy: true,
              }),
            ),
          )
        : originalFetch(input, options),
    ),
  )
  const user = userEvent.setup()
  render(<App />)
  await screen.findByRole('link', { name: 'Example account' })
  expect(screen.queryByText(i18n.t('ui.task_running'))).not.toBeInTheDocument()
  expect(
    screen.getByRole('button', { name: '立即执行 Example account' }),
  ).toBeDisabled()
  expect(
    screen.getByRole('button', { name: '更多操作 Example account' }),
  ).toBeDisabled()
  await user.type(screen.getByRole('textbox', { name: '筛选账户' }), 'Example')
  expect(
    screen.queryByRole('link', { name: 'Other account' }),
  ).not.toBeInTheDocument()
  await user.click(screen.getByRole('link', { name: 'Example account' }))
  await act(() => vi.dynamicImportSettled())
  expect(
    await screen.findByRole('heading', { name: 'Example account' }),
  ).toBeVisible()
  expect(window.location.pathname).toBe('/accounts/1')
})

it('retries failed callback information without keeping a loading skeleton', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/settings')
  let failed = true
  const originalFetch = globalThis.fetch
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, options?: RequestInit) =>
      requestURL(input) === '/api/info'
        ? Promise.resolve(
            new Response(
              JSON.stringify(
                failed
                  ? { code: 'storage_failed', error: 'Storage unavailable.' }
                  : {
                      callback_url: 'https://e5.example.com/api/oauth/callback',
                    },
              ),
              { status: failed ? 500 : 200 },
            ),
          )
        : originalFetch(input, options),
    ),
  )
  const user = userEvent.setup()
  await act(async () => {
    render(<App />)
  })
  await act(() => vi.dynamicImportSettled())
  expect(await screen.findByRole('alert')).toHaveTextContent('数据操作失败')
  expect(
    document.querySelector('[data-slot="skeleton"]'),
  ).not.toBeInTheDocument()
  failed = false
  await user.click(screen.getByRole('button', { name: '重试' }))
  expect(
    await screen.findByRole('textbox', { name: 'OAuth 回调地址' }),
  ).toHaveValue('https://e5.example.com/api/oauth/callback')
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})

it('retries callback information without submitting the account form', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/accounts')
  let infoRequests = 0
  const originalFetch = fetch
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>().mockImplementation((input, options) => {
      if (requestURL(input) === '/api/info') {
        infoRequests++
        return Promise.resolve(
          new Response(
            JSON.stringify(
              infoRequests === 1
                ? { code: 'storage_failed', error: 'Storage unavailable.' }
                : { callback_url: 'https://e5.example.com/api/oauth/callback' },
            ),
            { status: infoRequests === 1 ? 500 : 200 },
          ),
        )
      }
      return originalFetch(input, options)
    }),
  )
  const user = userEvent.setup()
  render(<App />)
  await user.click(
    await screen.findByRole('button', { name: '更多操作 Example account' }),
  )
  await user.click(screen.getByRole('menuitem', { name: '编辑' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('数据操作失败')
  await user.click(screen.getByRole('button', { name: '重试' }))
  expect(
    await screen.findByRole('textbox', { name: 'OAuth 回调地址' }),
  ).toHaveValue('https://e5.example.com/api/oauth/callback')
  expect(infoRequests).toBe(2)
  expect(lastUpdate).toBeUndefined()
  expect(screen.getByRole('dialog')).toBeVisible()
})

it('does not let an obsolete session response replace the current CSRF token', async () => {
  let finish!: (response: Response) => void
  const pending = new Promise<Response>((resolve) => {
    finish = resolve
  })
  const mockFetch = vi
    .fn<typeof fetch>()
    .mockReturnValueOnce(pending)
    .mockResolvedValueOnce(
      new Response(JSON.stringify({ authenticated: true, csrf: 'new-token' })),
    )
    .mockResolvedValueOnce(new Response('{}'))
  vi.stubGlobal('fetch', mockFetch)
  const controller = new AbortController()
  const previous = loadSession(controller.signal)
  controller.abort()
  await loadSession()
  finish(
    new Response(JSON.stringify({ authenticated: false, csrf: 'old-token' })),
  )
  await previous
  await request('/accounts/1/run', 'POST', {})
  expect(
    new Headers(mockFetch.mock.calls[2]?.[1]?.headers).get('X-CSRF-Token'),
  ).toBe('new-token')
})

it('keeps a saved account draft when authorization fails and retries without duplicating it', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/accounts')
  const originalFetch = globalThis.fetch
  let creates = 0
  let updates = 0
  let revision = 0
  const saved = {
    ...account,
    id: 7,
    has_refresh_token: false,
    name: 'Draft account',
  }
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
      const path = requestURL(input)
      const json = (value: unknown, status = 200) =>
        new Response(JSON.stringify(value), { status })
      if (path === '/api/accounts' && options?.method === 'POST') {
        creates++
        revision++
        expect(requestBody(options)).not.toHaveProperty('endpoints')
        return json({ ...saved, revision })
      }
      if (path === '/api/accounts/7' && options?.method === 'PUT') {
        updates++
        revision++
        return json({ ...saved, revision })
      }
      if (path === '/api/accounts/7/authorize') {
        return json(
          { code: 'storage_failed', error: 'Failed to start authorization.' },
          500,
        )
      }
      return originalFetch(input, options)
    }),
  )
  const user = userEvent.setup()
  render(<App />)
  await user.click(await screen.findByRole('button', { name: '添加账户' }))
  await user.type(await screen.findByLabelText('账户名称'), 'Draft account')
  await user.click(screen.getByLabelText('Client ID'))
  await user.paste(account.client_id)
  await user.click(screen.getByLabelText('Client Secret'))
  await user.paste('fixture-only-secret')
  await user.click(screen.getByRole('button', { name: '保存并授权' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('数据操作失败')
  await user.click(screen.getByRole('button', { name: '保存并授权' }))
  await waitFor(() => expect(updates).toBe(1))
  expect(creates).toBe(1)
})

it('completes a manual callback without receiving tokens in the browser', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/accounts')
  const originalFetch = globalThis.fetch
  const saved = { ...account, id: 7, has_refresh_token: false, revision: 1 }
  let exchanged = false
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
      const path = requestURL(input)
      const json = (value: unknown) => new Response(JSON.stringify(value))
      if (path === '/api/accounts' && options?.method === 'POST') {
        return json(saved)
      }
      if (path === '/api/accounts/7/authorize') {
        expect(requestBody(options)).toEqual({
          redirect_uri: 'https://callback.example.com/return',
        })
        return json({
          url: 'https://login.microsoftonline.com/common/oauth2/v2.0/authorize?state=fixture-state',
        })
      }
      if (path === '/api/accounts/7' && options?.method === 'GET') {
        return json(saved)
      }
      if (path === '/api/accounts/7/exchange') {
        expect(requestBody(options)).toEqual({
          callback_url:
            'https://callback.example.com/return?code=fixture-code&state=fixture-state',
        })
        exchanged = true
        return json({ authorized: true })
      }
      return originalFetch(input, options)
    }),
  )
  const user = userEvent.setup()
  render(<App />)
  await user.click(await screen.findByRole('button', { name: '添加账户' }))
  await user.type(await screen.findByLabelText('账户名称'), 'Manual account')
  await user.click(screen.getByLabelText('Client ID'))
  await user.paste(account.client_id)
  await user.click(screen.getByLabelText('Client Secret'))
  await user.paste('fixture-only-secret')
  await user.click(screen.getByRole('combobox', { name: '获取令牌的方式' }))
  await user.click(screen.getByRole('option', { name: '手动粘贴回调地址' }))
  await user.click(await screen.findByLabelText('Redirect URI'))
  await user.paste('https://callback.example.com/return')
  await user.click(screen.getByRole('button', { name: '保存并获取授权链接' }))
  const openLink = await screen.findByRole('link', {
    name: '打开 Microsoft 登录',
  })
  expect(openLink).toHaveAttribute('target', '_blank')
  await user.click(screen.getByLabelText('返回的回调地址'))
  await user.paste(
    'https://callback.example.com/return?code=fixture-code&state=fixture-state',
  )
  await user.click(screen.getByRole('button', { name: '完成授权' }))
  await waitFor(() => expect(exchanged).toBe(true))
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
  )
})

it('changes history filters and sends them to the paginated API', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/runs')
  const user = userEvent.setup()
  render(<App />)
  await user.click(await screen.findByRole('combobox', { name: '结果' }))
  await user.click(screen.getByRole('option', { name: '部分成功' }))
  await waitFor(() =>
    expect(fetch).toHaveBeenCalledWith(
      '/api/runs?status=partial',
      expect.anything(),
    ),
  )
  expect(await screen.findByText('没有匹配的执行记录')).toBeVisible()
})

it('opens request results directly from an account execution count', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/accounts/1')
  const original = fetch
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
      if (requestURL(input).startsWith('/api/runs')) {
        return new Response(
          JSON.stringify({
            runs: [
              {
                id: 8,
                account_id: 1,
                account_name: 'Example account',
                trigger: 'manual',
                status: 'success',
                error: '',
                error_code: '',
                started_at: '2026-10-16T00:00:00Z',
                finished_at: '2026-10-16T00:00:01Z',
                request_count: 1,
                success_count: 1,
                results: [
                  {
                    id: 1,
                    endpoint: 'profile',
                    http_status: 200,
                    error: '',
                    error_code: '',
                    started_at: '2026-10-16T00:00:00Z',
                    duration_ms: 120,
                    attempts: 1,
                  },
                ],
              },
            ],
            total: 1,
            page: 1,
            page_size: 25,
            busy: false,
          }),
          { headers: { 'Content-Type': 'application/json' } },
        )
      }
      return original(input, options)
    }),
  )
  render(<App />)
  const user = userEvent.setup()
  await user.click(
    await screen.findByRole('button', { name: '查看 1 次请求，其中 1 次成功' }),
  )
  const dialog = screen.getByRole('dialog')
  expect(within(dialog).getByText('HTTP 200')).toBeVisible()
  expect(within(dialog).getByText('个人资料')).toBeVisible()
  const accountCall = vi
    .mocked(fetch)
    .mock.calls.find(([url]) => url === '/api/accounts/1')
  expect(new Headers(accountCall?.[1]?.headers).get('X-Time-Zone')).toBe(
    Intl.DateTimeFormat().resolvedOptions().timeZone,
  )
})

describe('account list bounds', () => {
  function renderAccounts(count: number) {
    authenticated = true
    window.history.replaceState({}, '', '/accounts')
    const originalFetch = fetch
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockImplementation((input, options) => {
        if (requestURL(input) === '/api/accounts') {
          return Promise.resolve(
            new Response(
              JSON.stringify({
                accounts: Array.from({ length: count }, (_, index) => ({
                  ...account,
                  id: index + 1,
                  name: `Account ${index + 1}`,
                })),
                busy: false,
              }),
            ),
          )
        }
        return originalFetch(input, options)
      }),
    )
    render(<App />)
  }

  it('keeps the table bounded while paging', async () => {
    const user = userEvent.setup()
    renderAccounts(26)
    await screen.findByRole('link', { name: 'Account 1' })
    const table = screen.getByRole('table')
    // Count hidden rows too so the DOM bound cannot be satisfied with CSS alone.
    expect(within(table).getAllByRole('row', { hidden: true })).toHaveLength(26)
    expect(
      within(table).queryByRole('link', { name: 'Account 26' }),
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '下一页' }))
    expect(
      await within(table).findByRole('link', { name: 'Account 26' }),
    ).toBeVisible()
    expect(within(table).getAllByRole('row', { hidden: true })).toHaveLength(2)
    expect(
      within(table).queryByRole('link', { name: 'Account 1' }),
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '下一页' })).toBeDisabled()
  })

  it('searches accounts outside the rendered page', async () => {
    const user = userEvent.setup()
    renderAccounts(60)
    await screen.findByRole('link', { name: 'Account 1' })
    const table = screen.getByRole('table')
    expect(within(table).getAllByRole('row', { hidden: true })).toHaveLength(26)
    expect(
      within(table).queryByRole('link', { name: 'Account 58' }),
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('textbox', { name: '筛选账户' }))
    await user.paste('Account 58')
    expect(
      await within(table).findByRole('link', { name: 'Account 58' }),
    ).toBeVisible()
    expect(within(table).getAllByRole('row', { hidden: true })).toHaveLength(2)
  })
})

it('shows the authorization outcome when entering an already loaded account route', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/accounts')
  const previous = render(<App />)
  await screen.findByRole('heading', { name: '账户' })
  previous.unmount()
  window.history.replaceState({}, '', '/accounts?authorization=failed')
  render(<App />)
  const notice = await screen.findByText(
    i18n.t(
      'ui.authorization_failed_verify_the_callback_url_credentials_and_permissions_then_try_again',
    ),
  )
  await waitFor(() => expect(notice).toBeVisible())
})

it('preserves the password-change notice while the session reloads', async () => {
  authenticated = true
  window.history.replaceState({}, '', '/settings')
  const originalFetch = fetch
  let reloading = false
  let finishSession: (() => void) | undefined
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>().mockImplementation((input, options) => {
      if (requestURL(input) === '/api/password') {
        authenticated = false
        reloading = true
        return Promise.resolve(
          new Response(JSON.stringify({ authenticated: false })),
        )
      }
      if (requestURL(input) === '/api/session' && reloading) {
        return new Promise<Response>((resolve) => {
          finishSession = () =>
            resolve(
              new Response(JSON.stringify({ authenticated: false, csrf: '' })),
            )
        })
      }
      return originalFetch(input, options)
    }),
  )
  const user = userEvent.setup()
  render(<App />)
  await user.type(
    await screen.findByLabelText('当前密码'),
    'current-test-only-password',
  )
  await user.type(screen.getByLabelText('新密码'), 'new-test-only-password')
  await user.click(screen.getByRole('button', { name: '修改密码' }))
  await waitFor(() => expect(finishSession).toBeDefined())
  const notice = await screen.findByText(i18n.t('error.password_changed'))
  await waitFor(() => expect(notice).toBeVisible())
  finishSession?.()
  await screen.findByRole('heading', { name: '管理员登录' })
  expect(notice).toBeVisible()
})
