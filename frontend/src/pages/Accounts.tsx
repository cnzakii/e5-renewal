import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  KeyRound,
  Play,
  Ellipsis,
  RefreshCw,
  PencilLine,
  UsersRound,
  SearchX,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { request, errorKey, type Account } from '@/api'
import { useResource } from '@/use-resource'
import { ErrorNotice, PageHeading, Hint, EmptyState } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { AccountStatus } from '@/components/account-status'
import { ActivityChart, SuccessMeter } from '@/components/activity-chart'
import { Pagination } from '@/components/pagination'
import { AccountDialog } from '@/components/account-dialog'
import { TableLoading, Spinner } from '@/components/loading'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'

export default function Accounts() {
  const { t, i18n } = useTranslation()
  const { data, error, loading, refresh } = useResource<{
    accounts: Account[]
    busy: boolean
  }>('/accounts', 5000)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const actionTrigger = useRef<HTMLButtonElement | null>(null)
  const createTrigger = useRef<HTMLElement | null>(null)
  const location = useLocation()
  const [editing, setEditing] = useState<Account>()
  const [dialog, setDialog] = useState(false)
  const [deleting, setDeleting] = useState<Account>()
  const [pending, setPending] = useState(false)
  const [params, setParams] = useSearchParams()
  useEffect(() => {
    if (params.get('add') === '1' && data) {
      createTrigger.current = document.getElementById(
        (location.state as { createFrom?: string } | null)?.createFrom ===
          'navigation-menu'
          ? 'navigation-menu'
          : 'add-account',
      )
      setEditing(undefined)
      setDialog(true)
      const next = new URLSearchParams(params)
      next.delete('add')
      setParams(next, { replace: true, state: null })
      return
    }
    const outcome = params.get('authorization')
    if (outcome) {
      if (outcome === 'success') {
        toast.success(t('ui.authorization_completed'), {
          id: 'authorization-outcome',
        })
      } else {
        toast.error(
          t(
            'ui.authorization_failed_verify_the_callback_url_credentials_and_permissions_then_try_again',
          ),
          { id: 'authorization-outcome' },
        )
      }
      setParams({}, { replace: true })
    }
  }, [params, setParams, t, data, location.state])
  async function action(task: () => Promise<void>) {
    setPending(true)
    try {
      await task()
      await refresh()
    } catch (error) {
      toast.error(t([errorKey(error), 'error.request_failed']))
    } finally {
      setPending(false)
    }
  }
  const accounts =
    data?.accounts.filter((account) =>
      account.name.toLowerCase().includes(search.trim().toLowerCase()),
    ) || []
  const pages = Math.max(1, Math.ceil(accounts.length / 25))
  const currentPage = Math.min(page, pages)
  const visibleAccounts = accounts.slice(
    (currentPage - 1) * 25,
    currentPage * 25,
  )
  const busy = pending || data?.busy
  return (
    <>
      <Card
        className="w-full px-3 py-5 sm:px-6 sm:py-6"
        role="region"
        aria-label={t('ui.account_list')}
      >
        <PageHeading
          title="ui.accounts"
          icon={UsersRound}
          description="ui.manage_authorization_schedules_and_execution_status"
        >
          <Input
            className="h-9 min-w-0 flex-1 ps-3 text-[13px] md:w-56"
            aria-label={t('ui.filter_accounts')}
            placeholder={t('ui.filter_accounts_placeholder')}
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(1)
            }}
          />
          <Button
            variant="outline"
            size="icon"
            className="size-9"
            aria-label={t('ui.refresh')}
            onClick={() => void refresh()}
          >
            {loading ? <Spinner /> : <RefreshCw className="size-4" />}
          </Button>
        </PageHeading>
        <ErrorNotice message={error} onRetry={() => void refresh()} />
        {loading && !data ? (
          <TableLoading
            columns={[
              'md:w-[28%]',
              'hidden md:table-cell md:w-[18%]',
              'hidden sm:table-cell md:w-[26%]',
              'hidden sm:table-cell md:w-[18%]',
              'w-20 md:w-[10%]',
            ]}
          />
        ) : data ? (
          <div className="overflow-hidden rounded-md border">
            <Table className="table-fixed">
              <TableHeader>
                <TableRow>
                  <TableHead className="md:w-[28%]">
                    {t('ui.account')}
                  </TableHead>
                  <TableHead className="hidden md:table-cell md:w-[18%]">
                    {t('ui.account_state')}
                  </TableHead>
                  <TableHead className="hidden w-44 sm:table-cell md:w-[26%]">
                    {t('ui.calls_7d')}
                  </TableHead>
                  <TableHead className="hidden w-36 sm:table-cell md:w-[18%]">
                    {t('ui.success_rate')}
                  </TableHead>
                  <TableHead className="w-20 text-center md:w-[10%]">
                    {t('ui.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visibleAccounts.map((account) => (
                  <TableRow key={account.id}>
                    <TableCell className="px-3 sm:px-4">
                      <div className="flex min-w-0 flex-col gap-2">
                        <Link
                          to={'/accounts/' + account.id}
                          className="flex min-w-0 items-center gap-2 font-medium transition-colors hover:text-muted-foreground"
                        >
                          <span
                            aria-hidden="true"
                            className={cn(
                              'size-2 shrink-0 rounded-full',
                              account.last_error
                                ? 'bg-red-500'
                                : account.auth_type === 'delegated' &&
                                    !account.has_refresh_token
                                  ? 'bg-yellow-500'
                                  : account.enabled
                                    ? 'bg-green-500'
                                    : 'bg-primary/40',
                            )}
                          />
                          <span className="truncate">{account.name}</span>
                        </Link>
                        <div className="flex items-center gap-3 md:hidden">
                          <AccountStatus
                            account={account}
                            compact
                            showSchedule
                          />
                          <div className="ms-auto flex min-w-0 items-center gap-2 text-xs tabular-nums sm:hidden">
                            <span className="w-16">
                              <ActivityChart
                                statistics={account.statistics}
                                accountID={account.id}
                              />
                            </span>
                            <span
                              aria-label={
                                t('ui.calls_7d') +
                                ': ' +
                                (account.statistics?.requests || 0)
                              }
                            >
                              {account.statistics?.requests || 0}
                            </span>
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
                      <AccountStatus account={account} showSchedule />
                    </TableCell>
                    <TableCell className="hidden sm:table-cell">
                      <div className="flex items-center gap-3">
                        <span className="min-w-9 shrink-0 text-sm tabular-nums">
                          {account.statistics
                            ? new Intl.NumberFormat(i18n.language).format(
                                account.statistics.requests,
                              )
                            : '—'}
                          <span className="ms-1 text-xs text-muted-foreground">
                            {t('ui.calls_unit')}
                          </span>
                        </span>
                        <ActivityChart
                          statistics={account.statistics}
                          accountID={account.id}
                        />
                      </div>
                    </TableCell>
                    <TableCell className="hidden sm:table-cell">
                      <SuccessMeter statistics={account.statistics} />
                    </TableCell>
                    <TableCell className="px-2 sm:px-4">
                      <div className="flex justify-center gap-1">
                        <Hint label={t('ui.run_now')}>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-8"
                            disabled={
                              busy ||
                              (account.auth_type === 'delegated' &&
                                !account.has_refresh_token)
                            }
                            aria-label={t('ui.run_now') + ' ' + account.name}
                            onClick={() =>
                              void action(async () => {
                                await request(
                                  '/accounts/' + account.id + '/run',
                                  'POST',
                                  {},
                                )
                                toast.info(t('ui.task_started'))
                              })
                            }
                          >
                            <Play className="size-4" />
                          </Button>
                        </Hint>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="size-8"
                              disabled={busy}
                              aria-label={
                                t('ui.more_actions') + ' ' + account.name
                              }
                              onClick={(event) => {
                                actionTrigger.current = event.currentTarget
                              }}
                            >
                              <Ellipsis className="size-5" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end" className="min-w-44">
                            {account.auth_type === 'delegated' && (
                              <DropdownMenuItem
                                onSelect={() =>
                                  void action(async () => {
                                    const result = await request<{
                                      url: string
                                    }>(
                                      '/accounts/' + account.id + '/authorize',
                                      'POST',
                                      {},
                                    )
                                    window.location.assign(result.url)
                                  })
                                }
                              >
                                <KeyRound className="me-2.5 size-4" />
                                {t('ui.authorize')}
                              </DropdownMenuItem>
                            )}
                            <DropdownMenuItem
                              onSelect={() => {
                                setEditing(account)
                                setDialog(true)
                              }}
                            >
                              <PencilLine className="me-2.5 size-4" />
                              {t('ui.edit')}
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive focus:text-destructive"
                              onSelect={() => setDeleting(account)}
                            >
                              <Trash2 className="me-2.5 size-4" />
                              {t('ui.delete')}
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {!accounts.length && (
                  <TableRow>
                    <TableCell colSpan={5} className="h-44 text-center">
                      <EmptyState
                        icon={data?.accounts.length ? SearchX : UsersRound}
                        title={
                          data?.accounts.length
                            ? 'ui.no_matching_accounts'
                            : 'ui.no_accounts_yet'
                        }
                        description={
                          data?.accounts.length
                            ? 'ui.try_another_account_name'
                            : 'ui.add_an_account_complete_authorization_then_enable_its_schedule'
                        }
                      />
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        ) : null}
        {data && (
          <Pagination
            page={currentPage}
            pages={pages}
            onChange={setPage}
            label={t(pages > 1 ? 'ui.accounts_page' : 'ui.accounts_count', {
              page: currentPage,
              pages,
              count: accounts.length,
            })}
          />
        )}
      </Card>
      {dialog && (
        <AccountDialog
          account={editing}
          open={dialog}
          onOpenChange={setDialog}
          onSaved={refresh}
          returnFocus={editing ? actionTrigger.current : createTrigger.current}
        />
      )}
      <AlertDialog
        open={!!deleting}
        onOpenChange={(value) => {
          if (!pending && !value) {
            setDeleting(undefined)
          }
        }}
      >
        <AlertDialogContent
          onCloseAutoFocus={(event) => {
            const target = actionTrigger.current?.isConnected
              ? actionTrigger.current
              : document.getElementById('add-account')
            if (target) {
              event.preventDefault()
              target.focus()
            }
          }}
        >
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('ui.delete_account_confirm')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'ui.the_credentials_and_execution_history_for_will_be_permanently_deleted',
                { name: deleting?.name },
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>
              {t('ui.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={pending}
              onClick={(event) => {
                event.preventDefault()
                void action(async () => {
                  await request('/accounts/' + deleting!.id, 'DELETE')
                  setDeleting(undefined)
                  toast.success(t('ui.account_deleted'))
                })
              }}
            >
              {pending ? (
                <Spinner className="me-2" />
              ) : (
                <Trash2 className="me-2 size-4" />
              )}
              {t(pending ? 'ui.deleting' : 'ui.delete_account')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
