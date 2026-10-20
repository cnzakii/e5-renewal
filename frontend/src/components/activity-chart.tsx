import { useState, type PointerEvent } from 'react'
import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { AccountStatistics } from '@/api'
import { cn } from '@/lib/utils'
import { formatDay } from '@/lib/date'
import { Hint } from './common'
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip'

export function ActivityChart({
  statistics,
  accountID,
}: {
  statistics?: AccountStatistics
  accountID: number
}) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [active, setActive] = useState<number | null>(null)
  const days = statistics?.daily || []
  const maximum = Math.max(1, ...days.map((day) => day.requests))
  const points = days
    .map(
      (day, index) => `${2 + index * 16},${26 - (day.requests / maximum) * 22}`,
    )
    .join(' ')
  const selected = active === null ? undefined : days[active]
  function point(event: PointerEvent<SVGSVGElement>) {
    const bounds = event.currentTarget.getBoundingClientRect()
    setActive(
      Math.max(
        0,
        Math.min(
          days.length - 1,
          Math.round(
            ((event.clientX - bounds.left) / bounds.width) * (days.length - 1),
          ),
        ),
      ),
    )
  }
  function openDay() {
    if (selected) {
      void navigate(`/runs?account_id=${accountID}&date=${selected.day}`)
    }
  }
  return (
    <Tooltip open={!!selected}>
      <TooltipTrigger asChild>
        <svg
          viewBox="0 0 100 30"
          role="button"
          tabIndex={days.length ? 0 : -1}
          aria-label={
            t('ui.activity_chart') + ' · ' + t('ui.chart_drilldown_hint')
          }
          className="h-8 w-full min-w-0 max-w-28 cursor-pointer overflow-visible rounded-sm focus-visible:outline-2 focus-visible:outline-ring"
          onPointerMove={point}
          onPointerDown={point}
          onPointerLeave={() => setActive(null)}
          onFocus={() => setActive(days.length - 1)}
          onBlur={() => setActive(null)}
          onClick={openDay}
          onKeyDown={(event) => {
            if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
              event.preventDefault()
              setActive((index) =>
                Math.max(
                  0,
                  Math.min(
                    days.length - 1,
                    (index ?? 6) + (event.key === 'ArrowLeft' ? -1 : 1),
                  ),
                ),
              )
            }
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault()
              openDay()
            }
            if (event.key === 'Escape') {
              setActive(null)
            }
          }}
        >
          <path d="M2 26H98" stroke="var(--border)" fill="none" />
          {!!days.length && (
            <>
              <polygon
                points={`2,28 ${points} 98,28`}
                fill="var(--chart-1)"
                fillOpacity={0.06}
              />
              <polyline
                points={points}
                fill="none"
                stroke={
                  statistics?.requests ? 'var(--chart-1)' : 'var(--border)'
                }
                strokeOpacity={0.65}
                strokeWidth={1.3}
                strokeLinejoin="round"
              />
            </>
          )}
          {selected && (
            <>
              <line
                x1={2 + active! * 16}
                x2={2 + active! * 16}
                y1={2}
                y2={28}
                stroke="var(--muted-foreground)"
                strokeDasharray="2 2"
                strokeOpacity={0.5}
              />
              <circle
                cx={2 + active! * 16}
                cy={26 - (selected.requests / maximum) * 22}
                r={2.5}
                fill="var(--card)"
                stroke="var(--chart-1)"
                strokeWidth={1.5}
              />
            </>
          )}
        </svg>
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={8} className="grid gap-1">
        {selected && (
          <>
            <span className="font-medium">
              {formatDay(selected.day, i18n.language)}
            </span>
            <span>
              {t('ui.calls_total', { count: selected.requests })} ·{' '}
              {t('ui.successful_requests_count', {
                successes: selected.successes,
                count: selected.requests,
              })}
            </span>
            <span className="text-muted-foreground">
              {t('ui.chart_drilldown_hint')}
            </span>
          </>
        )}
      </TooltipContent>
    </Tooltip>
  )
}

export function successRate(statistics?: AccountStatistics) {
  return statistics?.requests
    ? Math.round((statistics.successes / statistics.requests) * 100) + '%'
    : '—'
}

export function SuccessMeter({
  statistics,
}: {
  statistics?: AccountStatistics
}) {
  const { t, i18n } = useTranslation()
  const rate = statistics?.requests
    ? Math.round((statistics.successes / statistics.requests) * 100)
    : undefined
  return (
    <Hint
      label={t('ui.successful_requests_count', {
        successes: statistics?.successes || 0,
        count: statistics?.requests || 0,
      })}
    >
      <div
        tabIndex={0}
        className="flex max-w-36 items-center gap-2 rounded-sm text-xs tabular-nums focus-visible:outline-2"
        role={rate === undefined ? undefined : 'meter'}
        aria-label={t('ui.success_rate')}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={rate}
      >
        <span className="w-10 shrink-0">
          {rate === undefined
            ? '—'
            : new Intl.NumberFormat(i18n.language, {
                style: 'percent',
                maximumFractionDigits: 0,
              }).format(rate / 100)}
        </span>
        <span
          aria-hidden="true"
          className="grid h-1.5 min-w-8 flex-1 overflow-hidden rounded-full bg-muted/70"
        >
          <span
            className={cn(
              'h-full rounded-full',
              rate === undefined
                ? 'bg-muted'
                : rate >= 95
                  ? 'bg-green-500/45'
                  : rate >= 70
                    ? 'bg-amber-500/45'
                    : 'bg-red-500/45',
            )}
            style={{ width: `${rate || 0}%` }}
          />
        </span>
      </div>
    </Hint>
  )
}
