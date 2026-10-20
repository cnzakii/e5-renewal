import { useTranslation } from 'react-i18next'
import { KeyRound, ShieldCheck, ShieldAlert, Timer, Pause } from 'lucide-react'
import { authorizationRequired, type Account } from '@/api'
import { cn } from '@/lib/utils'
import { Hint } from './common'

export function AccountStatus({
  account,
  compact = false,
  showSchedule = false,
}: {
  account: Account
  compact?: boolean
  showSchedule?: boolean
}) {
  const { t } = useTranslation()
  const needsAuthorization = authorizationRequired(account)
  const label = t(
    needsAuthorization
      ? 'ui.needs_authorization'
      : account.last_error
        ? 'ui.needs_attention'
        : account.auth_type === 'delegated'
          ? 'ui.connected'
          : 'ui.configured',
  )
  const Icon = needsAuthorization
    ? KeyRound
    : account.last_error
      ? ShieldAlert
      : ShieldCheck
  const schedule = t(account.enabled ? 'ui.scheduled' : 'ui.paused')
  const ScheduleIcon = account.enabled ? Timer : Pause
  const hint = [
    label,
    t(account.auth_type === 'delegated' ? 'ui.delegated' : 'ui.application'),
    account.last_error &&
      t(['error.' + account.last_error_code, 'error.request_failed']),
  ]
    .filter(Boolean)
    .join(' · ')
  return (
    <div className="flex min-w-0 items-center gap-3 text-[13px]">
      <Hint label={hint}>
        <span
          tabIndex={0}
          role="img"
          aria-label={hint}
          className={cn(
            'inline-flex min-w-0 items-center gap-1.5 rounded-sm focus-visible:outline-2',
            needsAuthorization
              ? 'text-muted-foreground'
              : account.last_error
                ? 'text-destructive'
                : 'text-green-600 dark:text-green-400',
          )}
        >
          <Icon
            aria-hidden="true"
            className="size-4 shrink-0"
            strokeWidth={1.75}
          />
          {!compact && (
            <span className="min-w-0 truncate text-foreground">{label}</span>
          )}
        </span>
      </Hint>
      {showSchedule && (
        <Hint label={schedule}>
          <span
            tabIndex={0}
            role="img"
            aria-label={schedule}
            className="shrink-0 rounded-sm text-muted-foreground focus-visible:outline-2"
          >
            <ScheduleIcon
              aria-hidden="true"
              className="size-4"
              strokeWidth={1.75}
            />
          </span>
        </Hint>
      )}
    </div>
  )
}
