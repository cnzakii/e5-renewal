import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  type TooltipContentProps,
} from 'recharts'
import { Activity, CheckCheck, ArrowUpRight } from 'lucide-react'
import type { AccountStatistics } from '@/api'
import { Card, CardContent, CardDescription, CardHeader } from './ui/card'
import { successRate } from './activity-chart'
import { formatDay, timeZoneLabel } from '@/lib/date'
import { Button } from './ui/button'

function ChartTooltip({
  active,
  payload,
  label,
  percentage,
  onDaySelect,
}: TooltipContentProps & {
  percentage: boolean
  onDaySelect: (day: string) => void
}) {
  const { t, i18n } = useTranslation()
  useEffect(() => {
    if (active && typeof label === 'string') {
      onDaySelect(label)
    }
  }, [active, label, onDaySelect])
  if (!active || !payload.length) {
    return null
  }
  return (
    <div className="grid min-w-40 gap-2 rounded-lg border border-border/50 bg-popover px-3 py-2 text-xs shadow-md">
      <span className="font-medium">
        {formatDay(String(label), i18n.language)}
      </span>
      {payload.map((item) => (
        <div key={String(item.dataKey)} className="flex items-center gap-2">
          <span
            className="size-2.5 shrink-0 rounded-xs"
            style={{ background: item.color }}
          />
          <span className="flex-1 text-muted-foreground">{item.name}</span>
          <span className="font-medium tabular-nums">
            {percentage
              ? `${Number(item.value)}%`
              : new Intl.NumberFormat(i18n.language).format(Number(item.value))}
          </span>
        </div>
      ))}
      <span className="mt-1 border-t pt-2 text-muted-foreground">
        {t('ui.chart_drilldown_hint')}
      </span>
    </div>
  )
}

export function AccountCharts({
  statistics,
  accountID,
}: {
  statistics?: AccountStatistics
  accountID: number
}) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [activeDay, setActiveDay] = useState<string>()
  const data = (statistics?.daily || []).map((day) => ({
    ...day,
    failures: day.requests - day.successes,
    rate: day.requests
      ? Math.round((day.successes / day.requests) * 100)
      : null,
  }))
  const number = new Intl.NumberFormat(i18n.language)
  const date = new Intl.DateTimeFormat(i18n.language, {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  })
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {[false, true].map((percentage) => {
        const Icon = percentage ? CheckCheck : Activity
        const title = t(percentage ? 'ui.success_rate' : 'ui.graph_requests')
        return (
          <Card
            key={String(percentage)}
            className="min-w-0 px-3 py-5 sm:px-6 sm:py-6"
          >
            <CardHeader className="relative mb-4 gap-1.5 p-0 pe-14">
              <h2 className="flex items-center gap-2 font-semibold">
                <Icon
                  aria-hidden="true"
                  className="size-4 text-muted-foreground"
                />
                {title}
              </h2>
              <CardDescription>
                {t('ui.last_7_days')} · {timeZoneLabel(i18n.language)}
              </CardDescription>
              <span className="absolute end-0 top-0 text-lg font-semibold tabular-nums">
                {percentage
                  ? successRate(statistics)
                  : number.format(statistics?.requests || 0)}
              </span>
            </CardHeader>
            <CardContent
              onClick={() => {
                if (activeDay) {
                  void navigate(
                    `/runs?account_id=${accountID}&date=${activeDay}`,
                  )
                }
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && activeDay) {
                  event.preventDefault()
                  void navigate(
                    `/runs?account_id=${accountID}&date=${activeDay}`,
                  )
                }
              }}
              className="-ms-3 -me-1 h-48 p-0 text-xs [&_.recharts-cartesian-axis-tick_text]:fill-muted-foreground [&_.recharts-cartesian-grid_line]:stroke-border/50 [&_.recharts-curve.recharts-tooltip-cursor]:stroke-border [&_.recharts-surface]:outline-hidden"
            >
              <ResponsiveContainer
                width="100%"
                height="100%"
                initialDimension={{ width: 400, height: 208 }}
              >
                <AreaChart
                  accessibilityLayer
                  syncId={'account-' + accountID}
                  data={data}
                  margin={{ top: 12, right: 18, left: 0, bottom: 0 }}
                >
                  <CartesianGrid vertical={false} />
                  <XAxis
                    dataKey="day"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={10}
                    minTickGap={24}
                    padding={{ left: 6, right: 6 }}
                    tickFormatter={(value) =>
                      date.format(new Date(String(value) + 'T00:00:00Z'))
                    }
                  />
                  <YAxis
                    tickLine={false}
                    axisLine={false}
                    width={46}
                    domain={percentage ? [0, 100] : [0, 'auto']}
                    allowDecimals={false}
                    tickFormatter={(value) =>
                      percentage ? `${value}%` : number.format(Number(value))
                    }
                  />
                  <Tooltip
                    animationDuration={100}
                    cursor={{
                      stroke: 'var(--muted-foreground)',
                      strokeDasharray: '3 3',
                      strokeOpacity: 0.4,
                    }}
                    wrapperStyle={{ pointerEvents: 'none' }}
                    content={(props) => (
                      <ChartTooltip
                        {...props}
                        percentage={percentage}
                        onDaySelect={setActiveDay}
                      />
                    )}
                  />
                  {percentage ? (
                    <Area
                      dataKey="rate"
                      name={t('ui.success_rate')}
                      type="monotoneX"
                      stroke="var(--chart-2)"
                      fill="var(--chart-2)"
                      fillOpacity={0.04}
                      strokeWidth={1.25}
                      isAnimationActive={false}
                      dot={false}
                      activeDot={{ r: 4, strokeWidth: 2, fill: 'var(--card)' }}
                    />
                  ) : (
                    <>
                      <Area
                        dataKey="requests"
                        name={t('ui.calls_total_label')}
                        type="monotoneX"
                        stroke="var(--chart-1)"
                        fill="var(--chart-1)"
                        fillOpacity={0.05}
                        strokeWidth={1.25}
                        isAnimationActive={false}
                      />
                      <Area
                        dataKey="failures"
                        name={t('ui.failed_requests')}
                        type="monotoneX"
                        stroke="var(--destructive)"
                        fill="var(--destructive)"
                        fillOpacity={0.05}
                        strokeWidth={1.25}
                        isAnimationActive={false}
                      />
                    </>
                  )}
                </AreaChart>
              </ResponsiveContainer>
            </CardContent>
            <div className="mt-3 flex min-h-5 items-center justify-center gap-4 text-xs text-muted-foreground">
              {(percentage
                ? [['var(--chart-2)', t('ui.success_rate')]]
                : [
                    ['var(--chart-1)', t('ui.calls_total_label')],
                    ['var(--destructive)', t('ui.failed_requests')],
                  ]
              ).map(([color, label]) => (
                <span key={label} className="inline-flex items-center gap-1.5">
                  <span
                    className="size-2 rounded-xs"
                    style={{ background: color }}
                  />
                  {label}
                </span>
              ))}
            </div>
            <div className="mt-1 text-center">
              <Button
                asChild
                variant="link"
                size="sm"
                className="h-7 text-xs text-muted-foreground"
              >
                <Link
                  to={`/runs?account_id=${accountID}${activeDay ? '&date=' + activeDay : ''}`}
                >
                  {activeDay && formatDay(activeDay, i18n.language) + ' · '}
                  {t(
                    activeDay ? 'ui.view_day_executions' : 'ui.all_executions',
                  )}
                  <ArrowUpRight data-icon="inline-end" />
                </Link>
              </Button>
            </div>
          </Card>
        )
      })}
    </div>
  )
}
