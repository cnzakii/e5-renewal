import type { Run } from '@/api'
import { useTranslation } from 'react-i18next'
import {
  Check,
  CircleAlert,
  Clock3,
  Square,
  Timer,
  MousePointer2,
} from 'lucide-react'
import { Badge } from './ui/badge'
import { Spinner } from './loading'

export function RunStatus({ status }: { status: Run['status'] }) {
  const { t } = useTranslation()
  const Icon =
    status === 'success'
      ? Check
      : status === 'partial'
        ? CircleAlert
        : status === 'interrupted'
          ? Square
          : status === 'running'
            ? Clock3
            : CircleAlert
  return (
    <Badge
      className="w-fit gap-1.5 whitespace-nowrap rounded-md px-2 text-[11px] font-medium"
      variant={
        status === 'success'
          ? 'success'
          : status === 'partial'
            ? 'warning'
            : status === 'running' || status === 'interrupted'
              ? 'secondary'
              : 'danger'
      }
    >
      {status === 'running' ? (
        <Spinner className="size-3.5 shrink-0" />
      ) : (
        <Icon className="size-3.5 shrink-0" />
      )}
      {t('status.' + status)}
    </Badge>
  )
}

export function RunTrigger({ trigger }: { trigger: Run['trigger'] }) {
  const { t } = useTranslation()
  const Icon = trigger === 'scheduled' ? Timer : MousePointer2
  return (
    <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-xs text-muted-foreground">
      <Icon aria-hidden="true" className="size-3.5" />
      {t('status.' + trigger)}
    </span>
  )
}
