import { LoaderCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'

export function Spinner({ className }: { className?: string }) {
  return (
    <LoaderCircle
      aria-hidden="true"
      className={cn(
        'size-4 animate-spin motion-reduce:animate-none',
        className,
      )}
    />
  )
}

export function LoadingPanel({ fullscreen = false }: { fullscreen?: boolean }) {
  const { t } = useTranslation()
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        'flex items-center justify-center',
        fullscreen ? 'min-h-svh' : 'relative min-h-52',
      )}
    >
      <Spinner className="size-10 opacity-60" />
      <span className="sr-only">{t('ui.loading')}</span>
    </div>
  )
}

export function TableLoading({
  columns = ['', '', '', ''],
}: {
  columns?: string[]
}) {
  const { t } = useTranslation()
  return (
    <div
      role="status"
      aria-live="polite"
      className="overflow-hidden rounded-md border"
    >
      <span className="sr-only">{t('ui.loading')}</span>
      <Table className="table-fixed" aria-hidden="true">
        <TableHeader>
          <TableRow>
            {columns.map((className, index) => (
              <TableHead key={index} className={className}>
                <Skeleton className="h-3 w-12 max-w-full" />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {[0, 1, 2, 3].map((row) => (
            <TableRow key={row}>
              {columns.map((className, index) => (
                <TableCell
                  key={index}
                  className={cn('h-20 sm:h-14', className)}
                >
                  <Skeleton className="h-3 w-3/4" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

export function PageLoading() {
  return (
    <Card className="p-3 py-5 sm:p-6">
      <div aria-hidden="true" className="mb-4 grid gap-3 px-1">
        <Skeleton className="h-6 w-28" />
        <Skeleton className="h-4 w-56 max-w-full" />
      </div>
      <TableLoading />
    </Card>
  )
}
