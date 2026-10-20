import { useState } from 'react'
import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Settings as SettingsIcon, KeyRound, Link2 } from 'lucide-react'
import { Spinner } from '@/components/loading'
import { Skeleton } from '@/components/ui/skeleton'
import { InputCopy } from '@/components/ui/input-copy'
import { request, errorKey } from '@/api'
import { useResource } from '@/use-resource'
import { ErrorNotice, PageHeading } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Card } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'

export default function Settings() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, error, loading, refresh } = useResource<{
    callback_url: string
  }>('/info')
  const [current, setCurrent] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  async function action(task: () => Promise<void>) {
    setPending(true)
    try {
      await task()
    } catch (error) {
      toast.error(t([errorKey(error), 'error.request_failed']))
    } finally {
      setPending(false)
    }
  }
  return (
    <Card className="mb-14 min-h-96 px-4 pt-5 pb-8 sm:px-7 sm:pt-6">
      <PageHeading
        title="ui.settings"
        icon={SettingsIcon}
        description="ui.manage_administrator_access_and_authorization_settings"
      />
      <ErrorNotice message={error} onRetry={() => void refresh()} />
      <Separator className="my-5" />
      <div className="flex max-w-3xl flex-col gap-8">
        <section className="flex flex-col gap-4">
          <h2 className="flex items-center gap-2 text-sm font-semibold">
            <Link2 className="size-4 text-muted-foreground" />
            {t('ui.oauth_callback_url')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t('ui.register_this_url_as_a_web_redirect_uri_in_microsoft_entra')}
          </p>
          {data ? (
            <InputCopy
              value={data.callback_url}
              label={t('ui.oauth_callback_url')}
            />
          ) : loading ? (
            <Skeleton className="h-10 w-full" />
          ) : null}
        </section>
        <Separator />
        <section className="flex flex-col gap-5">
          <div>
            <h2 className="flex items-center gap-2 text-sm font-semibold">
              <KeyRound className="size-4 text-muted-foreground" />
              {t('ui.management_password')}
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('ui.changing_the_password_signs_out_all_existing_sessions')}
            </p>
          </div>
          <form
            noValidate
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault()
              void action(async () => {
                await request('/password', 'PUT', {
                  current_password: current,
                  new_password: password,
                })
                setCurrent('')
                setPassword('')
                toast.success(t('error.password_changed'))
                window.dispatchEvent(new Event('e5:unauthorized'))
                void navigate('/login')
              })
            }}
          >
            <Field>
              <FieldLabel htmlFor="current-password">
                {t('ui.current_password')}
              </FieldLabel>
              <Input
                id="current-password"
                type="password"
                required
                autoComplete="current-password"
                maxLength={1024}
                value={current}
                onChange={(event) => setCurrent(event.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="new-password">
                {t('ui.new_password')}
              </FieldLabel>
              <Input
                id="new-password"
                type="password"
                required
                minLength={12}
                maxLength={1024}
                autoComplete="new-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
              <FieldDescription>
                {t('ui.use_at_least_12_characters')}
              </FieldDescription>
            </Field>
            <Button className="self-start" type="submit" disabled={pending}>
              {pending ? (
                <Spinner className="me-2" />
              ) : (
                <KeyRound className="me-2 size-4" />
              )}
              {t('ui.change_password')}
            </Button>
          </form>
        </section>
      </div>
    </Card>
  )
}
