import { useRef, useState } from 'react'
import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  ArrowLeft,
  ArrowUpRight,
  History,
  KeyRound,
  PencilLine,
  Play,
  RefreshCw,
  Timer,
  Pause,
  Clock3,
  Fingerprint,
  CalendarClock,
  Activity,
} from 'lucide-react'
import { toast } from 'sonner'
import { request, errorKey, type Account, type RunPage } from '@/api'
import { useResource } from '@/use-resource'
import { formatDate } from '@/lib/date'
import { AccountDialog } from '@/components/account-dialog'
import { AccountStatus } from '@/components/account-status'
import { AccountCharts } from '@/components/account-charts'
import { ErrorNotice, Hint, EmptyState } from '@/components/common'
import { PageLoading, Spinner, LoadingPanel } from '@/components/loading'
import { RunStatus, RunTrigger } from '@/components/run-status'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { RunDetails } from '@/components/run-details'
import { InputCopy } from '@/components/ui/input-copy'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@/components/ui/table'

export default function AccountDetail() {
  const { id } = useParams()
  const { t, i18n } = useTranslation()
  const {
    data: account,
    error,
    loading,
    refresh,
  } = useResource<Account>('/accounts/' + id, 5000)
  const {
    data: history,
    error: historyError,
    refresh: refreshHistory,
  } = useResource<RunPage>('/runs?account_id=' + id, 5000)
  const [selectedID, setSelectedID] = useState<number>()
  const selected = history?.runs.find((run) => run.id === selectedID)
  const [editing, setEditing] = useState(false)
  const [pending, setPending] = useState(false)
  const editTrigger = useRef<HTMLButtonElement>(null)
  async function action(operation: () => Promise<void>) {
    setPending(true)
    try {
      await operation()
      await Promise.all([refresh(), refreshHistory()])
    } catch (error) {
      toast.error(t([errorKey(error), 'error.request_failed']))
    } finally {
      setPending(false)
    }
  }
  if (!account) {
    return error ? (
      <Card className="p-6">
        <ErrorNotice message={error} onRetry={() => void refresh()} />
      </Card>
    ) : (
      <PageLoading />
    )
  }
  const ScheduleIcon = account.enabled ? Timer : Pause
  return (
    <>
      <div className="mb-3">
        <Button asChild variant="ghost" size="sm">
          <Link to="/accounts">
            <ArrowLeft data-icon="inline-start" />
            {t('ui.all_accounts')}
          </Link>
        </Button>
      </div>
      <Card className="mb-4">
        <div className="flex flex-wrap items-center justify-between gap-4 px-4 pb-4 pt-4 sm:px-6">
          <h1 className="min-w-0 break-words text-xl font-semibold sm:text-[1.375rem]">
            {account.name}
          </h1>
          <div className="flex flex-wrap items-center gap-2">
            <Hint label={t('ui.refresh')}>
              <Button
                variant="outline"
                size="icon"
                aria-label={t('ui.refresh')}
                onClick={() => void Promise.all([refresh(), refreshHistory()])}
              >
                {loading ? <Spinner /> : <RefreshCw />}
              </Button>
            </Hint>
            <Hint label={t('ui.edit')}>
              <Button
                ref={editTrigger}
                variant="outline"
                size="icon"
                aria-label={t('ui.edit')}
                onClick={() => setEditing(true)}
              >
                <PencilLine />
              </Button>
            </Hint>
            {account.auth_type === 'delegated' && (
              <Button
                variant="outline"
                className="w-10 px-0 sm:w-auto sm:px-4"
                aria-label={t('ui.authorize')}
                disabled={pending}
                onClick={() =>
                  void action(async () => {
                    const result = await request<{ url: string }>(
                      '/accounts/' + account.id + '/authorize',
                      'POST',
                      {},
                    )
                    window.location.assign(result.url)
                  })
                }
              >
                <KeyRound className="size-4 sm:me-2" />
                <span className="hidden sm:inline">{t('ui.authorize')}</span>
              </Button>
            )}
            <Button
              disabled={
                pending ||
                history?.busy ||
                (account.auth_type === 'delegated' &&
                  !account.has_refresh_token)
              }
              onClick={() =>
                void action(async () => {
                  await request('/accounts/' + account.id + '/run', 'POST', {})
                  toast.info(t('ui.task_started'))
                })
              }
            >
              {pending ? <Spinner /> : <Play data-icon="inline-start" />}
              {t('ui.run_now')}
            </Button>
          </div>
        </div>
        <CardContent className="px-4 pb-4 sm:px-6">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-3 text-sm">
            <AccountStatus account={account} />
            <Separator orientation="vertical" className="h-4" />
            <Hint label={t('ui.authorization')}>
              <span tabIndex={0} className="inline-flex items-center gap-1.5">
                <Fingerprint
                  aria-hidden="true"
                  className="size-4 text-muted-foreground"
                />
                {t(
                  account.auth_type === 'delegated'
                    ? 'ui.delegated'
                    : 'ui.application',
                )}
              </span>
            </Hint>
            <Separator orientation="vertical" className="h-4" />
            <span className="inline-flex items-center gap-1.5">
              <ScheduleIcon
                aria-hidden="true"
                className="size-4 text-muted-foreground"
              />
              {t(account.enabled ? 'ui.scheduled' : 'ui.paused')}
            </span>
            <Hint label={t('ui.interval')}>
              <span
                tabIndex={0}
                className="inline-flex items-center gap-1.5 tabular-nums"
              >
                <Clock3
                  aria-hidden="true"
                  className="size-4 text-muted-foreground"
                />
                {account.min_minutes}–{account.max_minutes} {t('ui.min')}
              </span>
            </Hint>
            {account.enabled && (
              <Hint label={t('ui.next_execution')}>
                <span
                  tabIndex={0}
                  className="inline-flex items-center gap-1.5 tabular-nums"
                >
                  <CalendarClock
                    aria-hidden="true"
                    className="size-4 text-muted-foreground"
                  />
                  {formatDate(account.next_run_at, i18n.language)}
                </span>
              </Hint>
            )}
          </div>
          <div className="mt-3 empty:hidden">
            <ErrorNotice
              message={
                error ||
                (account.last_error ? 'error.' + account.last_error_code : '')
              }
              onRetry={error ? () => void refresh() : undefined}
            />
          </div>
        </CardContent>
        <Separator />
        <div className="grid gap-3 px-4 py-3 sm:grid-cols-2 sm:px-6 [&_input]:h-8 [&_button]:size-8">
          <div className="grid gap-1.5">
            <span className="text-xs text-muted-foreground">Tenant ID</span>
            <InputCopy label="Tenant ID" value={account.tenant_id} />
          </div>
          <div className="grid gap-1.5">
            <span className="text-xs text-muted-foreground">Client ID</span>
            <InputCopy label="Client ID" value={account.client_id} />
          </div>
        </div>
      </Card>
      <AccountCharts statistics={account.statistics} accountID={account.id} />
      <Card className="mt-4 px-3 py-5 sm:px-6 sm:py-6">
        <div className="mb-4 flex items-center justify-between gap-3">
          <h2 className="flex items-center gap-2 font-semibold">
            <History
              aria-hidden="true"
              className="size-4 text-muted-foreground"
            />
            {t('ui.recent_executions')}
          </h2>
          <Button asChild variant="outline" size="sm">
            <Link to={'/runs?account_id=' + account.id}>
              <History data-icon="inline-start" />
              {t('ui.all_executions')}
            </Link>
          </Button>
        </div>
        <ErrorNotice
          message={historyError}
          onRetry={() => void refreshHistory()}
        />
        {!history && !historyError && <LoadingPanel />}
        {history &&
          (history.runs.length ? (
            <div className="overflow-hidden rounded-md border">
              <Table className="table-fixed">
                <TableHeader>
                  <TableRow>
                    <TableHead className="sm:w-[22%]">
                      {t('ui.result')}
                    </TableHead>
                    <TableHead className="hidden sm:table-cell sm:w-[20%]">
                      {t('ui.trigger')}
                    </TableHead>
                    <TableHead className="hidden sm:table-cell sm:w-[38%]">
                      {t('ui.started')}
                    </TableHead>
                    <TableHead className="w-24 sm:w-[20%]">
                      {t('ui.graph_requests')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {history.runs.slice(0, 5).map((run) => (
                    <TableRow key={run.id}>
                      <TableCell>
                        <RunStatus status={run.status} />
                        <span className="mt-1 block text-xs text-muted-foreground sm:hidden">
                          {formatDate(run.started_at, i18n.language)}
                        </span>
                      </TableCell>
                      <TableCell className="hidden sm:table-cell">
                        <RunTrigger trigger={run.trigger} />
                      </TableCell>
                      <TableCell className="hidden text-xs text-muted-foreground tabular-nums sm:table-cell">
                        {formatDate(run.started_at, i18n.language)}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 gap-1.5 px-2 text-xs tabular-nums"
                          onClick={() => setSelectedID(run.id)}
                          aria-label={t('ui.view_requests', {
                            successes: run.success_count,
                            count: run.request_count,
                          })}
                        >
                          {run.success_count}
                          <span className="text-muted-foreground">
                            / {run.request_count}
                          </span>
                          <ArrowUpRight className="size-3.5 text-muted-foreground" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <EmptyState
              icon={Activity}
              title="ui.no_executions_yet"
              description="ui.adjust_history_filters"
            />
          ))}
      </Card>
      <RunDetails run={selected} onClose={() => setSelectedID(undefined)} />
      {editing && (
        <AccountDialog
          account={account}
          open={editing}
          onOpenChange={setEditing}
          onSaved={refresh}
          returnFocus={editTrigger.current}
        />
      )}
    </>
  )
}
