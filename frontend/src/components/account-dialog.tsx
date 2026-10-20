import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ArrowLeft,
  ArrowUpRight,
  KeyRound,
  Save,
  Clock3,
  UserRound,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  request,
  errorKey,
  accountInput,
  authorizationRequired,
  type Account,
  type AccountInput,
  type AuthType,
} from '@/api'
import { useResource } from '@/use-resource'
import { ErrorNotice } from '@/components/common'
import { Spinner } from '@/components/loading'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { InputCopy } from '@/components/ui/input-copy'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSet,
  FieldLegend,
} from '@/components/ui/field'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectGroup,
  SelectItem,
} from '@/components/ui/select'

type Method = 'browser' | 'callback' | 'token'

export function AccountDialog({
  account,
  open,
  onOpenChange,
  onSaved,
  returnFocus,
}: {
  account?: Account
  open: boolean
  onOpenChange: (open: boolean) => void
  onSaved: () => Promise<void>
  returnFocus?: HTMLElement | null
}) {
  const { t } = useTranslation()
  const {
    data: info,
    error: infoError,
    refresh: reloadInfo,
  } = useResource<{ callback_url: string }>('/info')
  const [stored, setStored] = useState(account)
  const [form, setForm] = useState<AccountInput>(() =>
    account
      ? accountInput(account)
      : {
          name: '',
          auth_type: 'delegated',
          client_id: '',
          tenant_id: 'common',
          enabled: false,
          min_minutes: 120,
          max_minutes: 240,
          revision: 0,
        },
  )
  const [method, setMethod] = useState<Method>('browser')
  const [secret, setSecret] = useState('')
  const [refreshToken, setRefreshToken] = useState('')
  const [redirectURI, setRedirectURI] = useState('')
  const [authorizationURL, setAuthorizationURL] = useState('')
  const [callbackURL, setCallbackURL] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')

  const delegated = form.auth_type === 'delegated'
  const needsAuthorization =
    delegated &&
    (!stored?.has_refresh_token ||
      authorizationRequired(stored) ||
      form.auth_type !== stored.auth_type ||
      form.client_id !== stored.client_id ||
      form.tenant_id !== stored.tenant_id ||
      !!secret)
  const authorizeAfterSave =
    delegated &&
    method !== 'token' &&
    (needsAuthorization || method === 'callback')
  const set = <K extends keyof AccountInput>(key: K, value: AccountInput[K]) =>
    setForm((previous) => ({ ...previous, [key]: value }))

  async function submit(event: FormEvent) {
    event.preventDefault()

    if (
      delegated &&
      method === 'token' &&
      needsAuthorization &&
      !refreshToken
    ) {
      setError('error.refresh_token_required')
      return
    }

    if (authorizeAfterSave && method === 'callback' && !redirectURI.trim()) {
      setError('error.redirect_uri_invalid')
      return
    }

    setPending(true)
    setError('')

    try {
      const saved = await request<Account>(
        stored ? '/accounts/' + stored.id : '/accounts',
        stored ? 'PUT' : 'POST',
        {
          ...form,
          revision: stored?.revision || 0,
          enabled:
            form.enabled &&
            (!needsAuthorization || (method === 'token' && !!refreshToken)),
          ...(secret ? { client_secret: secret } : {}),
          ...(method === 'token' && refreshToken
            ? { refresh_token: refreshToken }
            : {}),
        },
      )
      // Keep the saved draft when starting authorization fails, so retry updates it.
      setStored(saved)
      setForm(accountInput(saved))
      setSecret('')
      setRefreshToken('')
      await onSaved()

      if (authorizeAfterSave) {
        const result = await request<{ url: string }>(
          '/accounts/' + saved.id + '/authorize',
          'POST',
          method === 'callback' ? { redirect_uri: redirectURI.trim() } : {},
        )
        if (method === 'browser') {
          window.location.assign(result.url)
        } else {
          setAuthorizationURL(result.url)
        }
      } else {
        onOpenChange(false)
        toast.success(t('ui.account_saved'))
      }
    } catch (error) {
      setError(errorKey(error))
    } finally {
      setPending(false)
    }
  }

  async function complete(event: FormEvent) {
    event.preventDefault()
    if (!stored) {
      return
    }

    setPending(true)
    setError('')

    try {
      const latest = await request<Account>('/accounts/' + stored.id)
      if (
        stored.has_refresh_token ||
        !latest.has_refresh_token ||
        latest.revision <= stored.revision ||
        latest.auth_type !== stored.auth_type ||
        latest.client_id !== stored.client_id ||
        latest.tenant_id !== stored.tenant_id
      ) {
        await request('/accounts/' + stored.id + '/exchange', 'POST', {
          callback_url: callbackURL.trim(),
        })
      }
      await onSaved()
      onOpenChange(false)
      toast.success(t('ui.authorization_completed'))
    } catch (error) {
      setError(errorKey(error))
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!pending) {
          onOpenChange(value)
        }
      }}
    >
      <DialogContent
        className="max-h-[90dvh] grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden p-0 sm:max-w-2xl"
        onCloseAutoFocus={(event) => {
          if (returnFocus?.isConnected) {
            event.preventDefault()
            returnFocus.focus()
          }
        }}
      >
        <DialogHeader className="px-6 pt-6 pb-5">
          <DialogTitle>
            {t(
              authorizationURL
                ? 'ui.complete_authorization'
                : account
                  ? 'ui.edit_account'
                  : 'ui.add_account',
            )}
          </DialogTitle>
          <DialogDescription>
            {t(
              authorizationURL
                ? 'ui.manual_authorization_hint'
                : 'ui.connect_a_microsoft_application_stored_credentials_are_never_displayed',
            )}
          </DialogDescription>
        </DialogHeader>
        <form
          noValidate
          onSubmit={(event) =>
            void (authorizationURL ? complete(event) : submit(event))
          }
          className="flex min-h-0 flex-col overflow-hidden"
        >
          <Separator />
          <div className="flex min-h-0 flex-col gap-5 overflow-y-auto px-6 py-5">
            {authorizationURL ? (
              <FieldGroup>
                <Field>
                  <FieldLabel>{t('ui.authorization_link')}</FieldLabel>
                  <InputCopy
                    value={authorizationURL}
                    label={t('ui.authorization_link')}
                  />
                </Field>
                <Button asChild variant="outline">
                  <a href={authorizationURL} target="_blank" rel="noreferrer">
                    <ArrowUpRight className="me-2 size-4" />
                    {t('ui.open_microsoft_sign_in')}
                  </a>
                </Button>
                <Field>
                  <FieldLabel htmlFor="callback-url">
                    {t('ui.callback_url')}
                  </FieldLabel>
                  <Input
                    id="callback-url"
                    value={callbackURL}
                    onChange={(event) => setCallbackURL(event.target.value)}
                    placeholder="https://…?code=…&state=…"
                    maxLength={20000}
                    autoComplete="off"
                  />
                  <FieldDescription>
                    {t('ui.paste_callback_hint')}
                  </FieldDescription>
                </Field>
              </FieldGroup>
            ) : (
              <>
                <FieldSet disabled={pending}>
                  <FieldLegend>
                    <UserRound className="size-4 text-muted-foreground" />
                    {t('ui.account')}
                  </FieldLegend>
                  <FieldGroup className="sm:grid-cols-2">
                    <Field>
                      <FieldLabel htmlFor="account-name">
                        {t('ui.account_name')}
                      </FieldLabel>
                      <Input
                        id="account-name"
                        required
                        maxLength={120}
                        value={form.name}
                        onChange={(event) => set('name', event.target.value)}
                        placeholder={t('ui.personal_microsoft_365')}
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="auth-type">
                        {t('ui.authorization')}
                      </FieldLabel>
                      <Select
                        value={form.auth_type}
                        onValueChange={(value) => {
                          setForm((previous) => ({
                            ...previous,
                            auth_type: value as AuthType,
                            tenant_id:
                              value === 'application' &&
                              ['common', 'organizations', 'consumers'].includes(
                                previous.tenant_id,
                              )
                                ? ''
                                : previous.tenant_id,
                            enabled: false,
                          }))
                          setRefreshToken('')
                          setMethod('browser')
                          setError('')
                        }}
                      >
                        <SelectTrigger id="auth-type">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="delegated">
                              {t('ui.delegated_browser_authorization')}
                            </SelectItem>
                            <SelectItem value="application">
                              {t('ui.application_client_credentials')}
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </Field>
                  </FieldGroup>
                </FieldSet>
                <Separator />
                <FieldSet disabled={pending}>
                  <FieldLegend>
                    <KeyRound className="size-4 text-muted-foreground" />
                    {t('ui.microsoft_application')}
                  </FieldLegend>
                  <FieldGroup>
                    <FieldGroup className="sm:grid-cols-3">
                      <Field>
                        <FieldLabel htmlFor="tenant-id">Tenant ID</FieldLabel>
                        <Input
                          id="tenant-id"
                          required
                          value={form.tenant_id}
                          onChange={(event) =>
                            set('tenant_id', event.target.value)
                          }
                        />
                      </Field>
                      <Field className="sm:col-span-2">
                        <FieldLabel htmlFor="client-id">Client ID</FieldLabel>
                        <Input
                          id="client-id"
                          required
                          value={form.client_id}
                          onChange={(event) =>
                            set('client_id', event.target.value)
                          }
                        />
                      </Field>
                    </FieldGroup>
                    <Field>
                      <FieldLabel htmlFor="client-secret">
                        Client Secret
                      </FieldLabel>
                      <Input
                        id="client-secret"
                        type="password"
                        autoComplete="new-password"
                        required={!stored}
                        maxLength={4096}
                        value={secret}
                        onChange={(event) => setSecret(event.target.value)}
                        placeholder={
                          stored
                            ? t('ui.leave_empty_to_keep_the_stored_secret')
                            : ''
                        }
                      />
                    </Field>
                    {delegated ? (
                      <>
                        <Field>
                          <FieldLabel htmlFor="token-method">
                            {t('ui.token_method')}
                          </FieldLabel>
                          <Select
                            value={method}
                            onValueChange={(value) => {
                              setMethod(value as Method)
                              setRefreshToken('')
                              setError('')
                            }}
                          >
                            <SelectTrigger id="token-method">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectGroup>
                                <SelectItem value="browser">
                                  {t('ui.browser_authorization')}
                                </SelectItem>
                                <SelectItem value="callback">
                                  {t('ui.manual_callback')}
                                </SelectItem>
                                <SelectItem value="token">
                                  {t('ui.existing_refresh_token')}
                                </SelectItem>
                              </SelectGroup>
                            </SelectContent>
                          </Select>
                        </Field>
                        {method === 'browser' && (
                          <Field>
                            <FieldLabel>
                              {t('ui.oauth_callback_url')}
                            </FieldLabel>
                            {info ? (
                              <InputCopy
                                value={info.callback_url}
                                label={t('ui.oauth_callback_url')}
                              />
                            ) : !infoError ? (
                              <Skeleton className="h-10 w-full" />
                            ) : null}
                            <ErrorNotice
                              message={infoError}
                              onRetry={() => void reloadInfo()}
                            />
                            <FieldDescription>
                              {t('ui.browser_authorization_hint')}
                            </FieldDescription>
                          </Field>
                        )}
                        {method === 'callback' && (
                          <Field>
                            <FieldLabel htmlFor="redirect-uri">
                              Redirect URI
                            </FieldLabel>
                            <Input
                              id="redirect-uri"
                              type="url"
                              value={redirectURI}
                              onChange={(event) =>
                                setRedirectURI(event.target.value)
                              }
                              placeholder="https://…"
                              autoComplete="off"
                            />
                            <FieldDescription>
                              {t('ui.registered_redirect_hint')}
                            </FieldDescription>
                          </Field>
                        )}
                        {method === 'token' && (
                          <Field>
                            <FieldLabel htmlFor="refresh-token">
                              Refresh Token
                            </FieldLabel>
                            <Input
                              id="refresh-token"
                              type="password"
                              autoComplete="off"
                              maxLength={16384}
                              value={refreshToken}
                              onChange={(event) =>
                                setRefreshToken(event.target.value)
                              }
                              placeholder={
                                stored?.has_refresh_token
                                  ? t(
                                      'ui.leave_empty_to_keep_the_stored_secret',
                                    )
                                  : ''
                              }
                            />
                          </Field>
                        )}
                      </>
                    ) : null}
                    <FieldDescription>
                      {t(
                        delegated
                          ? 'ui.delegated_permissions_hint'
                          : 'ui.application_permissions_hint',
                      )}
                    </FieldDescription>
                  </FieldGroup>
                </FieldSet>
                <Separator />
                <FieldSet disabled={pending}>
                  <FieldLegend>
                    <Clock3 className="size-4 text-muted-foreground" />
                    {t('ui.schedule')}
                  </FieldLegend>
                  <FieldDescription>
                    {t('ui.random_execution_hint')}
                  </FieldDescription>
                  <FieldGroup className="grid-cols-2">
                    <Field>
                      <FieldLabel htmlFor="min-minutes">
                        {t('ui.minimum_interval_minutes')}
                      </FieldLabel>
                      <Input
                        id="min-minutes"
                        type="number"
                        required
                        min={1}
                        max={43200}
                        value={form.min_minutes}
                        onChange={(event) =>
                          set('min_minutes', Number(event.target.value))
                        }
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="max-minutes">
                        {t('ui.maximum_interval_minutes')}
                      </FieldLabel>
                      <Input
                        id="max-minutes"
                        type="number"
                        required
                        min={form.min_minutes}
                        max={43200}
                        value={form.max_minutes}
                        onChange={(event) =>
                          set('max_minutes', Number(event.target.value))
                        }
                      />
                    </Field>
                  </FieldGroup>
                  <label className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={
                        form.enabled && (!needsAuthorization || !!refreshToken)
                      }
                      disabled={needsAuthorization && !refreshToken}
                      onCheckedChange={(value) =>
                        set('enabled', value === true)
                      }
                    />
                    {t('ui.enable_automatic_execution')}
                  </label>
                  {needsAuthorization && !refreshToken && (
                    <FieldDescription>
                      {t('error.authorization_required')}
                    </FieldDescription>
                  )}
                </FieldSet>
              </>
            )}
          </div>
          <Separator />
          <div className="flex shrink-0 flex-col gap-3 px-6 py-4">
            <ErrorNotice message={error} />
            <DialogFooter>
              {authorizationURL && (
                <Button
                  type="button"
                  variant="ghost"
                  disabled={pending}
                  onClick={() => {
                    setAuthorizationURL('')
                    setCallbackURL('')
                    setError('')
                  }}
                >
                  <ArrowLeft className="me-2 size-4" />
                  {t('ui.back')}
                </Button>
              )}
              <Button
                type="button"
                variant="outline"
                disabled={pending}
                onClick={() => onOpenChange(false)}
              >
                {t('ui.cancel')}
              </Button>
              <Button type="submit" className="min-w-40" disabled={pending}>
                {pending ? (
                  <Spinner className="me-2" />
                ) : authorizationURL || authorizeAfterSave ? (
                  <ArrowUpRight className="me-2 size-4" />
                ) : (
                  <Save className="me-2 size-4" />
                )}
                {t(
                  pending
                    ? 'ui.saving'
                    : authorizationURL
                      ? 'ui.complete_authorization'
                      : authorizeAfterSave
                        ? method === 'callback'
                          ? 'ui.get_authorization_link'
                          : 'ui.save_and_authorize'
                        : 'ui.save_account',
                )}
              </Button>
            </DialogFooter>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
