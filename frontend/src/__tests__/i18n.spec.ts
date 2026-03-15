// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest'
import { useI18n } from '../i18n'

describe('useI18n', () => {
  beforeEach(() => {
    localStorage.clear()
    // Reset locale to default (zh) by setting it explicitly
    const { setLocale } = useI18n()
    setLocale('zh')
  })

  it('t() returns Chinese translation when locale is zh', () => {
    const { t } = useI18n()
    expect(t('nav.dashboard')).toBe('仪表盘')
    expect(t('login.submit')).toBe('登录')
  })

  it('t() returns English translation when locale is en', () => {
    const { setLocale, t } = useI18n()
    setLocale('en')
    expect(t('nav.dashboard')).toBe('Dashboard')
    expect(t('login.submit')).toBe('Sign In')
  })

  it('t() returns the key itself for missing keys (fallback)', () => {
    const { t } = useI18n()
    expect(t('nonexistent.key')).toBe('nonexistent.key')
  })

  it('setLocale switches language and persists to localStorage', () => {
    const { setLocale, locale } = useI18n()
    setLocale('en')
    expect(locale.value).toBe('en')
    expect(localStorage.getItem('locale')).toBe('en')

    setLocale('zh')
    expect(locale.value).toBe('zh')
    expect(localStorage.getItem('locale')).toBe('zh')
  })

  it('toggleLocale toggles between zh and en', () => {
    const { toggleLocale, locale } = useI18n()
    expect(locale.value).toBe('zh')

    toggleLocale()
    expect(locale.value).toBe('en')

    toggleLocale()
    expect(locale.value).toBe('zh')
  })

  it('localeLabel returns EN when locale is zh, and 中 when locale is en', () => {
    const { localeLabel, setLocale } = useI18n()
    setLocale('zh')
    expect(localeLabel.value).toBe('EN')

    setLocale('en')
    expect(localeLabel.value).toBe('中')
  })

  it('uses Client Secret expiry wording for affected account and settings labels', () => {
    const { t, setLocale } = useI18n()

    setLocale('en')
    expect(t('accounts.expiry')).toBe('Client Secret Expiry')
    expect(t('accounts.expiry.none')).toBe('No Client Secret expiry set')
    expect(t('accounts.expiry.expired')).toBe('Client Secret expired')
    expect(t('accounts.expiry.today')).toBe('Client Secret expires today')
    expect(t('accounts.expiry.remaining')).toBe('Client Secret expires in {days} days')
    expect(t('accounts.form.expiresAt')).toBe('Client Secret Expiry')
    expect(t('accounts.form.expiresAt.hint.authCode')).toBe('Used only for Client Secret expiry reminders')
    expect(t('settings.notification.onAuthExpiry')).toBe('Client Secret Expiry Warning')

    setLocale('zh')
    expect(t('accounts.expiry')).toBe('Client Secret 过期时间')
    expect(t('accounts.expiry.none')).toBe('未设置 Client Secret 过期时间')
    expect(t('accounts.expiry.expired')).toBe('Client Secret 已过期')
    expect(t('accounts.expiry.today')).toBe('Client Secret 今日过期')
    expect(t('accounts.expiry.remaining')).toBe('Client Secret 还剩 {days} 天过期')
    expect(t('accounts.form.expiresAt')).toBe('Client Secret 过期时间')
    expect(t('accounts.form.expiresAt.hint.authCode')).toBe('仅用于 Client Secret 过期提醒')
    expect(t('settings.notification.onAuthExpiry')).toBe('Client Secret 即将过期')
  })

  it('has refresh token mode keys in both locales', () => {
    const { t, setLocale } = useI18n()

    const refreshTokenKeys = [
      'accounts.form.refreshToken.mode.auto',
      'accounts.form.refreshToken.mode.manual',
      'accounts.form.refreshToken.mode.direct',
      'accounts.form.refreshToken.auto.redirectUri',
      'accounts.form.refreshToken.auto.redirectUri.hint',
      'accounts.form.refreshToken.auto.authorize',
      'accounts.form.refreshToken.manual.redirectUri',
      'accounts.form.refreshToken.manual.redirectUri.placeholder',
      'accounts.form.refreshToken.manual.authorize',
      'accounts.form.refreshToken.manual.authorizeUrl',
      'accounts.form.refreshToken.manual.callbackUrl',
      'accounts.form.refreshToken.manual.callbackUrl.placeholder',
      'accounts.form.refreshToken.manual.exchange',
      'accounts.form.refreshToken.manual.exchange.hint',
      'accounts.form.refreshToken.manual.exchangeExpired',
      'accounts.form.refreshToken.redirectUri.invalid',
      'accounts.form.refreshToken.callbackUrl.invalid',
    ]

    setLocale('zh')
    for (const key of refreshTokenKeys) {
      expect(t(key), `zh missing key: ${key}`).not.toBe(key)
    }

    setLocale('en')
    for (const key of refreshTokenKeys) {
      expect(t(key), `en missing key: ${key}`).not.toBe(key)
    }

    // Spot-check specific values
    setLocale('zh')
    expect(t('accounts.form.refreshToken.mode.auto')).toBe('自动获取')
    expect(t('accounts.form.refreshToken.mode.manual')).toBe('手动获取')
    expect(t('accounts.form.refreshToken.mode.direct')).toBe('直接填写')

    setLocale('en')
    expect(t('accounts.form.refreshToken.mode.auto')).toBe('Auto')
    expect(t('accounts.form.refreshToken.mode.manual')).toBe('Manual')
    expect(t('accounts.form.refreshToken.mode.direct')).toBe('Direct')
  })

  it('has notification language labels in both locales', () => {
    const { t, setLocale } = useI18n()

    setLocale('en')
    expect(t('settings.notification.language')).toBe('Notification Language')
    expect(t('settings.notification.language.zh')).toBe('中文')
    expect(t('settings.notification.language.en')).toBe('English')

    setLocale('zh')
    expect(t('settings.notification.language')).toBe('通知语言')
    expect(t('settings.notification.language.zh')).toBe('中文')
    expect(t('settings.notification.language.en')).toBe('English')
  })
})
