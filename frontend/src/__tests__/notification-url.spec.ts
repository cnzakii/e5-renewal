import { describe, expect, it } from 'vitest'

function buildTelegramUrl(botToken: string, chatId: string): string {
  return `telegram://${encodeURIComponent(botToken)}@telegram?chats=${encodeURIComponent(chatId)}`
}

function parseTelegramUrl(raw: string): { botToken: string; chatId: string } | null {
  const match = raw.match(/^telegram:\/\/([^@]+)@telegram\?(.+)$/)
  if (!match) return null

  const botToken = decodeURIComponent(match[1] || '')
  const query = new URLSearchParams(match[2] || '')
  const chatId = query.get('chats') || ''
  if (!botToken || !chatId) return null

  return { botToken, chatId }
}

function buildSmtpUrl(input: {
  username: string
  password: string
  host: string
  port: string
  to: string
  from: string
}): string {
  const authPart = `${encodeURIComponent(input.username)}:${encodeURIComponent(input.password)}`
  const hostPart = `${input.host}:${input.port}`
  const query = new URLSearchParams({ to: input.to, from: input.from })
  return `smtp://${authPart}@${hostPart}/?${query.toString()}`
}

function parseSmtpUrl(raw: string): {
  username: string
  password: string
  host: string
  port: string
  to: string
  from: string
} | null {
  try {
    const url = new URL(raw)
    if (url.protocol !== 'smtp:') return null

    const username = decodeURIComponent(url.username || '')
    const password = decodeURIComponent(url.password || '')
    const host = url.hostname || ''
    const port = url.port || '587'
    const to = url.searchParams.get('to') || ''
    const from = url.searchParams.get('from') || ''

    if (!host || !to || !from) return null

    return { username, password, host, port, to, from }
  } catch {
    return null
  }
}

describe('notification url helpers', () => {
  it('builds and parses telegram url', () => {
    const url = buildTelegramUrl('123456:abcDEF', '@ops_channel')
    expect(url).toBe('telegram://123456%3AabcDEF@telegram?chats=%40ops_channel')

    const parsed = parseTelegramUrl(url)
    expect(parsed).toEqual({ botToken: '123456:abcDEF', chatId: '@ops_channel' })
  })

  it('builds and parses smtp url', () => {
    const url = buildSmtpUrl({
      username: 'ops-bot',
      password: 'p@ss:word',
      host: 'smtp.example.com',
      port: '587',
      to: 'ops@example.com',
      from: 'bot@example.com',
    })

    const parsed = parseSmtpUrl(url)
    expect(parsed).toEqual({
      username: 'ops-bot',
      password: 'p@ss:word',
      host: 'smtp.example.com',
      port: '587',
      to: 'ops@example.com',
      from: 'bot@example.com',
    })
  })

  it('rejects invalid telegram url', () => {
    expect(parseTelegramUrl('telegram://invalid')).toBeNull()
  })

  it('rejects smtp url without mandatory query params', () => {
    expect(parseSmtpUrl('smtp://user:pass@smtp.example.com:587')).toBeNull()
  })
})
