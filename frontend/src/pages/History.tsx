import { useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  ArrowLeft,
  RefreshCw,
  History as HistoryIcon,
  ListFilter,
  Eye,
  Info,
} from 'lucide-react'
import { useResource } from '@/use-resource'
import type { Account, RunPage } from '@/api'
import { formatDate, formatDay } from '@/lib/date'
import { cn } from '@/lib/utils'
import { ErrorNotice, PageHeading, EmptyState } from '@/components/common'
import { Button } from '@/components/ui/button'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TableLoading, Spinner } from '@/components/loading'
import { Card } from '@/components/ui/card'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectGroup,
  SelectItem,
} from '@/components/ui/select'
import { Pagination } from '@/components/pagination'
import { RunDetails } from '@/components/run-details'
import { RunStatus, RunTrigger } from '@/components/run-status'

export default function History() {
  const { t, i18n } = useTranslation()
  const [params, setParams] = useSearchParams()
  const {
    data: accounts,
    error: accountsError,
    refresh: refreshAccounts,
  } = useResource<{ accounts: Account[] }>('/accounts')
  const { data, error, loading, refresh } = useResource<RunPage>(
    '/runs' + (params.size ? '?' + params.toString() : ''),
    5000,
  )
  const [selectedID, setSelectedID] = useState<number>()
  const scoped = !!params.get('account_id')
  const selectedDate = params.get('date')
  const selected = data?.runs.find((run) => run.id === selectedID)
  function filter(key: string, value: string) {
    const next = new URLSearchParams(params)
    if (value === 'all') {
      next.delete(key)
    } else {
      next.set(key, value)
    }
    if (key === 'days') {
      next.delete('date')
    }
    next.delete('page')
    setParams(next, { replace: true })
    setSelectedID(undefined)
  }
  const pages = Math.max(
    1,
    Math.ceil((data?.total || 0) / (data?.page_size || 25)),
  )
  const choices = [
    {
      key: 'account_id',
      label: 'ui.account',
      items: [
        ['all', 'ui.all_accounts'],
        ...(accounts?.accounts.map((a) => [String(a.id), a.name]) || []),
      ],
    },
    {
      key: 'status',
      label: 'ui.result',
      items: [
        ['all', 'ui.all_results'],
        ...['success', 'partial', 'failed', 'running', 'interrupted'].map(
          (v) => [v, 'status.' + v],
        ),
      ],
    },
    {
      key: 'trigger',
      label: 'ui.trigger',
      items: [
        ['all', 'ui.all_triggers'],
        ['manual', 'status.manual'],
        ['scheduled', 'status.scheduled'],
      ],
    },
    {
      key: 'days',
      label: 'ui.time_range',
      items: [
        ['1', 'ui.today'],
        ['7', 'ui.last_7_days'],
        ['30', 'ui.last_30_days'],
        ...(selectedDate
          ? [['date', formatDay(selectedDate, i18n.language)]]
          : []),
      ],
    },
  ]
  return (
    <>
      <Card className="w-full px-3 py-5 sm:px-6 sm:py-6">
        <PageHeading
          title="ui.execution_history"
          icon={HistoryIcon}
          description="ui.recent_graph_requests_and_authorization_errors"
        >
          {params.get('account_id') && (
            <Button asChild variant="outline">
              <Link to="/runs">
                <ArrowLeft className="me-1.5 size-4" />
                {t('ui.all_accounts')}
              </Link>
            </Button>
          )}
          <Button
            variant="outline"
            size="icon"
            aria-label={t('ui.refresh')}
            onClick={() => void refresh()}
          >
            {loading ? <Spinner /> : <RefreshCw className="size-4" />}
          </Button>
        </PageHeading>
        <div className="mb-4 grid grid-cols-2 gap-2 md:flex md:flex-wrap">
          {choices.map(({ key, label, items }) => (
            <Select
              key={key}
              value={
                key === 'days' && selectedDate
                  ? 'date'
                  : params.get(key) || (key === 'days' ? '30' : 'all')
              }
              onValueChange={(v) => {
                if (v !== 'date') {
                  filter(key, v)
                }
              }}
            >
              <SelectTrigger aria-label={t(label)} className="md:w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {items.map(([value, text]) => (
                    <SelectItem key={value} value={value}>
                      {(key === 'account_id' && value !== 'all') ||
                      value === 'date'
                        ? text
                        : t(text)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          ))}
        </div>
        <ErrorNotice
          message={error || accountsError}
          onRetry={() => void Promise.all([refresh(), refreshAccounts()])}
        />
        {loading && !data ? (
          <TableLoading
            columns={
              scoped
                ? [
                    'w-[24%]',
                    'hidden sm:table-cell sm:w-[24%]',
                    'hidden sm:table-cell sm:w-[36%]',
                    'w-16 sm:w-[16%]',
                  ]
                : [
                    'sm:w-[30%]',
                    'hidden sm:table-cell sm:w-[20%]',
                    'hidden sm:table-cell sm:w-[16%]',
                    'hidden md:table-cell md:w-[24%]',
                    'w-16 sm:w-[10%]',
                  ]
            }
          />
        ) : data ? (
          <>
            <div className="overflow-hidden rounded-md border">
              <Table className="table-fixed">
                <TableHeader>
                  <TableRow>
                    {!scoped && (
                      <TableHead className="sm:w-[30%]">
                        {t('ui.account')}
                      </TableHead>
                    )}
                    <TableHead
                      className={
                        scoped
                          ? 'sm:w-[24%]'
                          : 'hidden sm:table-cell sm:w-[20%]'
                      }
                    >
                      {t('ui.result')}
                    </TableHead>
                    <TableHead
                      className={
                        scoped
                          ? 'hidden sm:table-cell sm:w-[24%]'
                          : 'hidden sm:table-cell sm:w-[16%]'
                      }
                    >
                      {t('ui.trigger')}
                    </TableHead>
                    <TableHead
                      className={
                        scoped
                          ? 'hidden sm:table-cell sm:w-[36%]'
                          : 'hidden md:table-cell md:w-[24%]'
                      }
                    >
                      {t('ui.started')}
                    </TableHead>
                    <TableHead
                      className={
                        scoped
                          ? 'w-16 px-2 text-center sm:w-[16%] sm:px-4'
                          : 'w-16 px-2 text-center sm:w-[10%] sm:px-4'
                      }
                    >
                      {t('ui.details')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.runs.map((run) => (
                    <TableRow key={run.id}>
                      {!scoped && (
                        <TableCell className="min-w-0 font-medium">
                          <div className="grid min-w-0 gap-1.5">
                            <span className="truncate">{run.account_name}</span>
                            <div className="sm:hidden">
                              <RunStatus status={run.status} />
                            </div>
                          </div>
                        </TableCell>
                      )}
                      <TableCell
                        className={scoped ? '' : 'hidden sm:table-cell'}
                      >
                        <RunStatus status={run.status} />
                        {scoped && (
                          <span className="mt-1 block text-xs text-muted-foreground sm:hidden">
                            {formatDate(run.started_at, i18n.language)}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="hidden text-muted-foreground sm:table-cell">
                        <RunTrigger trigger={run.trigger} />
                      </TableCell>
                      <TableCell
                        className={cn(
                          'hidden whitespace-nowrap text-muted-foreground tabular-nums',
                          scoped ? 'sm:table-cell' : 'md:table-cell',
                        )}
                      >
                        {formatDate(run.started_at, i18n.language)}
                      </TableCell>
                      <TableCell className="px-2 sm:px-4">
                        <div className="flex justify-center">
                          <Button
                            size="sm"
                            variant="ghost"
                            className="w-10 gap-1.5 px-2 sm:w-auto"
                            aria-label={t('ui.view')}
                            onClick={() => setSelectedID(run.id)}
                          >
                            <Eye className="size-4" />
                            <span className="hidden sm:inline">
                              {t('ui.view')}
                            </span>
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                  {!data.runs.length && (
                    <TableRow>
                      <TableCell
                        colSpan={scoped ? 4 : 5}
                        className="h-44 text-center"
                      >
                        <EmptyState
                          icon={ListFilter}
                          title="ui.no_matching_executions"
                          description="ui.adjust_history_filters"
                        />
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
            <Pagination
              page={data.page || 1}
              pages={pages}
              disabled={loading}
              label={t('ui.history_page', {
                page: data.page || 1,
                pages,
                count: data.total || 0,
              })}
              onChange={(page) => {
                const next = new URLSearchParams(params)
                next.set('page', String(page))
                setParams(next, { replace: true })
              }}
            />
            <p className="mt-3 flex min-h-10 items-start gap-2 px-1 text-xs leading-5 text-muted-foreground sm:min-h-5">
              <Info className="mt-0.5 size-3.5 shrink-0" />
              <span>{t('ui.history_retention')}</span>
            </p>
          </>
        ) : null}
      </Card>
      <RunDetails run={selected} onClose={() => setSelectedID(undefined)} />
    </>
  )
}
