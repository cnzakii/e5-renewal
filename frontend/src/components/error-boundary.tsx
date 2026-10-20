import { Component, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { CircleAlert, RefreshCw } from 'lucide-react'
import { EmptyState, Preferences } from './common'
import { Button } from './ui/button'
import { Card } from './ui/card'
import { useTheme } from '@/lib/theme'

function Recovery() {
  const { t } = useTranslation()
  const theme = useTheme()
  return (
    <main className="grid min-h-svh place-items-center p-4">
      <div className="absolute right-3 top-3">
        <Preferences theme={theme} />
      </div>
      <Card
        className="flex w-full max-w-md flex-col items-center p-6"
        role="alert"
      >
        <EmptyState
          icon={CircleAlert}
          title="ui.page_unavailable"
          description="ui.page_unavailable_description"
        />
        <Button onClick={() => window.location.reload()}>
          <RefreshCw className="me-2 size-4" />
          {t('ui.reload_page')}
        </Button>
      </Card>
    </main>
  )
}

export class ErrorBoundary extends Component<
  { children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false }
  static getDerivedStateFromError() {
    return { failed: true }
  }
  render() {
    return this.state.failed ? <Recovery /> : this.props.children
  }
}
