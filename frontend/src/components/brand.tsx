import mark from '@/assets/wordmark.svg'
import { cn } from '@/lib/utils'

export function Brand({ className }: { className?: string }) {
  return (
    <span className={cn('inline-flex items-center', className)}>
      <img
        src={mark}
        alt=""
        aria-hidden="true"
        className="h-5 w-auto shrink-0 dark:invert"
      />
      <span className="sr-only">E5 Renewal</span>
    </span>
  )
}
