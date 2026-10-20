import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock3, RotateCw } from 'lucide-react'
import type { Run } from '@/api'
import { formatDate } from '@/lib/date'
import { ErrorNotice, Hint } from './common'
import { LoadingPanel } from './loading'
import { RunStatus } from './run-status'
import { Badge } from './ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from './ui/dialog'

export function RunDetails({
  run,
  onClose,
}: {
  run?: Run
  onClose: () => void
}) {
  const { t, i18n } = useTranslation()
  // Preserve content until Radix completes the closing animation.
  const [displayed, setDisplayed] = useState(run)
  if (run && run !== displayed) {
    setDisplayed(run)
  }
  const detail = run || displayed
  const detailTitle = useRef<HTMLHeadingElement>(null)
  return (
    <Dialog
      open={!!run}
      onOpenChange={(value) => {
        if (!value) {
          onClose()
        }
      }}
    >
      <DialogContent
        className="max-h-[90dvh] overflow-y-auto"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          detailTitle.current?.focus()
        }}
      >
        <DialogHeader>
          <DialogTitle ref={detailTitle} tabIndex={-1}>
            {detail?.account_name}
          </DialogTitle>
          <DialogDescription>
            {detail ? formatDate(detail.started_at, i18n.language) : ''}
          </DialogDescription>
        </DialogHeader>
        {detail && <RunStatus status={detail.status} />}
        <ErrorNotice
          message={detail?.error ? 'error.' + detail.error_code : ''}
        />
        {detail?.status === 'running' && <LoadingPanel />}
        <div className="grid gap-2">
          {detail?.results.map((result) => (
            <div
              key={result.id}
              className="flex flex-col gap-3 rounded-md border px-3 py-3 text-sm"
            >
              <div className="flex items-center justify-between gap-4">
                <span>{t('endpoint.' + result.endpoint)}</span>
                <Badge variant={result.error ? 'danger' : 'success'}>
                  {result.http_status
                    ? 'HTTP ' + result.http_status
                    : t('ui.no_http_response')}
                </Badge>
              </div>
              <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground tabular-nums">
                <Hint label={t('ui.request_duration')}>
                  <span
                    tabIndex={0}
                    className="inline-flex items-center gap-1.5"
                  >
                    <Clock3 aria-hidden="true" className="size-3.5" />
                    {new Intl.NumberFormat(i18n.language).format(
                      result.duration_ms,
                    )}{' '}
                    ms
                  </span>
                </Hint>
                <Hint label={t('ui.request_attempts')}>
                  <span
                    tabIndex={0}
                    className="inline-flex items-center gap-1.5"
                  >
                    <RotateCw aria-hidden="true" className="size-3.5" />
                    {result.attempts}
                  </span>
                </Hint>
                <span className="ms-auto">
                  {formatDate(result.started_at, i18n.language)}
                </span>
              </div>
              <ErrorNotice
                message={result.error ? 'error.' + result.error_code : ''}
              />
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
