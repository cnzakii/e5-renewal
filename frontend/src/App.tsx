import {
  lazy,
  Suspense,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from 'react'
import {
  BrowserRouter,
  Link,
  NavLink,
  Navigate,
  Outlet,
  Route,
  Routes,
  useNavigate,
} from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  History as HistoryIcon,
  LogOut,
  UsersRound,
  Settings as SettingsIcon,
  LogIn,
  Plus,
  Menu,
  Ellipsis,
  Languages,
  Lock,
  CircleCheck,
  CircleAlert,
  TriangleAlert,
  Info,
  X,
  Moon,
  Sun,
} from 'lucide-react'
import { toast, Toaster } from 'sonner'
import { request, errorKey, loadSession, type Session } from './api'
import { Preferences, ErrorNotice } from '@/components/common'
import { Brand } from '@/components/brand'
import { Spinner, LoadingPanel, PageLoading } from '@/components/loading'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Field, FieldLabel } from '@/components/ui/field'
import { useTheme, type Theme } from '@/lib/theme'
import { basePath } from '@/lib/base-path'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuGroup,
} from '@/components/ui/dropdown-menu'

const Accounts = lazy(() => import('./pages/Accounts'))
const AccountDetail = lazy(() => import('./pages/AccountDetail'))
const History = lazy(() => import('./pages/History'))
const Settings = lazy(() => import('./pages/Settings'))

function Layout({
  onSessionChange,
  theme,
}: {
  onSessionChange: () => Promise<void>
  theme: Theme
}) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [pending, setPending] = useState(false)
  const links = [
    ['/accounts', 'ui.accounts', UsersRound],
    ['/runs', 'ui.history', HistoryIcon],
    ['/settings', 'ui.settings', SettingsIcon],
  ] as const
  async function signOut() {
    setPending(true)
    try {
      await request('/logout', 'POST', {})
      await onSessionChange()
    } catch (error) {
      toast.error(t([errorKey(error), 'error.request_failed']))
    } finally {
      setPending(false)
    }
  }
  const addAccount = (from: string) => {
    void navigate('/accounts?add=1', { state: { createFrom: from } })
  }
  return (
    <>
      <div className="container">
        <header className="my-4 flex h-14 items-center rounded-md border border-border/60 bg-card px-4 pe-3 sm:px-6 md:h-16">
          <Link to="/accounts" className="me-2 shrink-0 py-2">
            <Brand />
          </Link>
          <nav
            aria-label={t('ui.main_navigation')}
            className="ms-5 hidden items-center gap-1 md:flex"
          >
            {links.slice(0, 2).map(([path, label, Icon]) => (
              <Button key={path} asChild variant="ghost" size="sm">
                <NavLink
                  to={path}
                  className="gap-2 text-muted-foreground aria-[current=page]:bg-accent aria-[current=page]:text-foreground"
                >
                  <Icon className="size-4" strokeWidth={1.5} />
                  {t(label)}
                </NavLink>
              </Button>
            ))}
          </nav>
          <div className="ms-auto flex items-center gap-2">
            <Button
              id="add-account"
              variant="outline"
              size="sm"
              className="hidden min-w-32 md:inline-flex"
              onClick={() => addAccount('add-account')}
            >
              <Plus className="me-1.5 size-4" />
              {t('ui.add_account')}
            </Button>
            <DropdownMenu modal={false}>
              <DropdownMenuTrigger asChild>
                <Button
                  id="navigation-menu"
                  variant="ghost"
                  size="icon"
                  className="size-9"
                  aria-label={t('ui.navigation_menu')}
                >
                  <Ellipsis className="hidden size-5 md:block" />
                  <Menu className="size-5 md:hidden" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="min-w-48 text-sm">
                <DropdownMenuGroup>
                  {links.map(([path, label, Icon], index) => (
                    <DropdownMenuItem
                      key={path}
                      asChild
                      className={index < 2 ? 'md:hidden' : ''}
                    >
                      <Link to={path}>
                        <Icon className="me-2.5 size-4" />
                        {t(label)}
                      </Link>
                    </DropdownMenuItem>
                  ))}
                  <DropdownMenuItem
                    className="md:hidden"
                    onSelect={() => addAccount('navigation-menu')}
                  >
                    <Plus className="me-2.5 size-4" />
                    {t('ui.add_account')}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuItem
                    onSelect={() =>
                      void i18n.changeLanguage(
                        i18n.language === 'zh' ? 'en' : 'zh',
                      )
                    }
                  >
                    <Languages className="me-2.5 size-4" />
                    {t('ui.change_language')}
                    <span className="ms-auto text-xs text-muted-foreground">
                      {i18n.language === 'zh' ? 'English' : '中文'}
                    </span>
                  </DropdownMenuItem>
                  <DropdownMenuItem onSelect={theme.toggle}>
                    {theme.dark ? (
                      <Sun className="me-2.5 size-4" />
                    ) : (
                      <Moon className="me-2.5 size-4" />
                    )}
                    {t('ui.toggle_theme')}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  disabled={pending}
                  onSelect={() => void signOut()}
                >
                  <LogOut className="me-2.5 size-4" />
                  {t('ui.sign_out')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>
      </div>
      <main id="main-content" className="container relative pb-12">
        <Suspense fallback={<PageLoading />}>
          <Outlet />
        </Suspense>
      </main>
    </>
  )
}

function Login({
  onLogin,
  theme,
}: {
  onLogin: () => Promise<void>
  theme: Theme
}) {
  const { t } = useTranslation()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!password) {
      setError('ui.enter_your_management_password_to_continue')
      return
    }
    setPending(true)
    setError('')
    try {
      await request('/login', 'POST', { password })
      await onLogin()
    } catch (error) {
      setError(errorKey(error))
    } finally {
      setPending(false)
    }
  }
  return (
    <main id="main-content" className="grid min-h-svh items-center py-12">
      <div className="absolute right-3 top-3">
        <Preferences theme={theme} />
      </div>
      <div className="mx-auto grid w-full max-w-[21.5em] gap-5 px-4 [--border:hsl(30_8%_70%)] dark:[--border:hsl(220_3%_25%)]">
        <div className="text-center">
          <div className="mb-3 flex justify-center">
            <Brand className="[&_img]:h-7" />
          </div>
          <h1 className="text-sm text-muted-foreground">
            {t('ui.administrator_sign_in')}
          </h1>
        </div>
        <form
          noValidate
          onSubmit={(event) => void submit(event)}
          className="grid gap-2.5"
        >
          <Field>
            <FieldLabel className="sr-only" htmlFor="password">
              {t('ui.management_password')}
            </FieldLabel>
            <div className="relative">
              <Lock
                aria-hidden="true"
                className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                id="password"
                className="ps-9"
                type="password"
                required
                maxLength={1024}
                autoComplete="current-password"
                placeholder={t('ui.management_password')}
                disabled={pending}
                aria-invalid={!!error}
                aria-describedby={error ? 'login-error' : undefined}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
          </Field>
          {error && (
            <p
              id="login-error"
              role="alert"
              className="px-1 text-xs text-red-600 dark:text-red-400"
            >
              {t([error, 'error.request_failed'])}
            </p>
          )}
          <Button type="submit" disabled={pending}>
            {pending ? (
              <Spinner className="me-2" />
            ) : (
              <LogIn className="me-2 size-4" />
            )}
            {pending ? t('ui.signing_in') : t('ui.sign_in')}
          </Button>
        </form>
      </div>
    </main>
  )
}

function SessionRoutes({ theme }: { theme: Theme }) {
  const { t } = useTranslation()
  const [session, setSession] = useState<Session>()
  const [error, setError] = useState('')
  const sessionRequest = useRef<AbortController | null>(null)
  async function refreshSession() {
    sessionRequest.current?.abort()
    const controller = new AbortController()
    sessionRequest.current = controller
    setError('')
    setSession(undefined)
    try {
      const session = await loadSession(controller.signal)
      if (!controller.signal.aborted) {
        setSession(session)
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        setError(errorKey(error))
      }
    }
  }
  useEffect(() => {
    void refreshSession()
    const unauthorized = () => {
      setSession(undefined)
      void refreshSession()
    }
    window.addEventListener('e5:unauthorized', unauthorized)
    return () => {
      sessionRequest.current?.abort()
      window.removeEventListener('e5:unauthorized', unauthorized)
    }
  }, [])
  if (error) {
    return (
      <main className="grid min-h-svh place-items-center p-4">
        <div className="absolute right-3 top-3">
          <Preferences theme={theme} />
        </div>
        <Card className="flex w-full max-w-md flex-col gap-5 p-6">
          <Brand />
          <ErrorNotice message={error} onRetry={() => void refreshSession()} />
        </Card>
      </main>
    )
  }
  if (!session) {
    return <LoadingPanel fullscreen />
  }
  return (
    <BrowserRouter basename={basePath() || '/'}>
      <Link to="#main-content" className="sr-only focus:not-sr-only">
        {t('ui.skip_to_content')}
      </Link>
      <Routes>
        <Route
          path="/login"
          element={
            session.authenticated ? (
              <Navigate to="/accounts" replace />
            ) : (
              <Login onLogin={refreshSession} theme={theme} />
            )
          }
        />
        <Route
          element={
            session.authenticated ? (
              <Layout onSessionChange={refreshSession} theme={theme} />
            ) : (
              <Navigate to="/login" replace />
            )
          }
        >
          <Route path="/accounts" element={<Accounts />} />
          <Route path="/accounts/:id" element={<AccountDetail />} />
          <Route path="/runs" element={<History />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/" element={<Navigate to="/accounts" replace />} />
        </Route>
        <Route path="*" element={<Navigate to="/accounts" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export function App() {
  const theme = useTheme()
  const { t } = useTranslation()
  return (
    <>
      {/* Keep notifications mounted while session checks replace route content. */}
      <Toaster
        className="app-toaster"
        position="top-center"
        theme={theme.dark ? 'dark' : 'light'}
        richColors
        duration={5000}
        offset={24}
        mobileOffset={{
          top: 'calc(env(safe-area-inset-top) + 16px)',
          left: 16,
          right: 16,
        }}
        closeButton
        icons={{
          success: <CircleCheck className="size-4" />,
          error: <CircleAlert className="size-4" />,
          warning: <TriangleAlert className="size-4" />,
          info: <Info className="size-4" />,
          loading: <Spinner />,
          close: <X className="size-4" />,
        }}
        containerAriaLabel={t('ui.messages')}
        toastOptions={{
          closeButtonAriaLabel: t('ui.close'),
        }}
      />
      <SessionRoutes theme={theme} />
    </>
  )
}
