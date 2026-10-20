import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from './ui/button'

export function Pagination({
  label,
  page,
  pages,
  disabled = false,
  onChange,
}: {
  label: string
  page: number
  pages: number
  disabled?: boolean
  onChange: (page: number) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="mt-4 flex flex-wrap items-center justify-between gap-3 px-1">
      <p className="text-xs text-muted-foreground tabular-nums">{label}</p>
      {pages > 1 && (
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="icon"
            disabled={disabled || page <= 1}
            aria-label={t('ui.previous_page')}
            onClick={() => onChange(page - 1)}
          >
            <ChevronLeft className="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            disabled={disabled || page >= pages}
            aria-label={t('ui.next_page')}
            onClick={() => onChange(page + 1)}
          >
            <ChevronRight className="size-4" />
          </Button>
        </div>
      )}
    </div>
  )
}
