import { type ReactElement, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  CircleAlert,
  Languages,
  Moon,
  Sun,
  type LucideIcon,
} from 'lucide-react'
import type { Theme } from '@/lib/theme'
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
} from '@/components/ui/empty'

export function Preferences({ theme }: { theme: Theme }) {
  const { t, i18n } = useTranslation()
  return (
    <div className="flex items-center gap-1">
      <Hint label={t('ui.change_language')}>
        <Button
          variant="ghost"
          size="icon"
          aria-label={t('ui.change_language')}
          onClick={() =>
            void i18n.changeLanguage(i18n.language === 'zh' ? 'en' : 'zh')
          }
        >
          <Languages className="size-[1.2rem]" strokeWidth={1.5} />
          <span className="sr-only">
            {i18n.language === 'zh' ? 'English' : '中文'}
          </span>
        </Button>
      </Hint>
      <Hint label={t('ui.toggle_theme')}>
        <Button
          variant="ghost"
          size="icon"
          aria-label={t('ui.toggle_theme')}
          onClick={theme.toggle}
        >
          {theme.dark ? (
            <Sun className="size-[1.2rem]" strokeWidth={1.5} />
          ) : (
            <Moon className="size-[1.2rem]" strokeWidth={1.5} />
          )}
        </Button>
      </Hint>
    </div>
  )
}

export function ErrorNotice({
  message,
  onRetry,
}: {
  message: string
  onRetry?: () => void
}) {
  const { t } = useTranslation()
  if (!message) {
    return null
  }
  return (
    <Alert
      variant="destructive"
      role="alert"
      className="[&>svg]:top-1/2 [&>svg]:-translate-y-1/2 [&>svg+div]:translate-y-0"
    >
      <CircleAlert className="size-4" />
      <AlertDescription className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <span className="min-w-0 flex-1 text-[13px]">
          {t([message, 'error.request_failed'])}
        </span>
        {onRetry && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8 shrink-0 text-xs"
            onClick={onRetry}
          >
            {t('ui.retry')}
          </Button>
        )}
      </AlertDescription>
    </Alert>
  )
}

export function PageHeading({
  title,
  description,
  children,
  icon: Icon,
}: {
  icon?: LucideIcon
  title: ReactNode
  description: string
  children?: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <div className="mb-3 grid w-full items-center gap-x-5 gap-y-3 sm:mb-4 md:flex">
      <div className="min-w-0 flex-1 px-2 sm:px-1">
        <h1 className="mb-2 flex items-center gap-2.5 text-card-title font-semibold leading-none tracking-tight">
          {Icon && (
            <Icon
              aria-hidden="true"
              className="size-5 text-muted-foreground"
              strokeWidth={1.5}
            />
          )}
          {typeof title === 'string' ? t(title) : title}
        </h1>
        <p className="text-[13px] leading-5 text-muted-foreground">
          {t(description)}
        </p>
      </div>
      {children && (
        <div className="ms-auto flex w-full shrink-0 items-center justify-end gap-2 md:w-64">
          {children}
        </div>
      )}
    </div>
  )
}

export function Hint({
  label,
  children,
}: {
  label: string
  children: ReactElement
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent sideOffset={6}>{label}</TooltipContent>
    </Tooltip>
  )
}

export function EmptyState({
  icon: Icon,
  title,
  description,
}: {
  icon: LucideIcon
  title: string
  description: string
}) {
  const { t } = useTranslation()
  return (
    <Empty className="py-8 md:p-8">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Icon aria-hidden="true" strokeWidth={1.5} />
        </EmptyMedia>
        <EmptyTitle>{t(title)}</EmptyTitle>
        <EmptyDescription>{t(description)}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  )
}
