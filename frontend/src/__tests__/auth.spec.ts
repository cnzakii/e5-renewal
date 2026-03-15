// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest'
import { useAuth } from '../stores/auth'

describe('auth store', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    const auth = useAuth()
    auth.clearAuth()
  })

  it('初始状态无 token，未认证', () => {
    const auth = useAuth()
    expect(auth.token.value).toBe('')
    expect(auth.isAuthenticated.value).toBe(false)
  })

  it('记住我为 true 时 token 存入 localStorage', () => {
    const auth = useAuth()
    auth.setAuth('test-token', true)
    expect(auth.token.value).toBe('test-token')
    expect(auth.isAuthenticated.value).toBe(true)
    expect(localStorage.getItem('token')).toBe('test-token')
  })

  it('记住我为 false 时 token 存入 sessionStorage', () => {
    const auth = useAuth()
    auth.setAuth('test-token', false)
    expect(auth.token.value).toBe('test-token')
    expect(sessionStorage.getItem('token')).toBe('test-token')
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('切换记住我模式时只保留当前存储中的 token', () => {
    const auth = useAuth()
    auth.setAuth('persisted-token', true)
    auth.setAuth('session-token', false)

    expect(sessionStorage.getItem('token')).toBe('session-token')
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('登出时清除认证信息', () => {
    const auth = useAuth()
    auth.setAuth('test-token', true)
    auth.clearAuth()
    expect(auth.token.value).toBe('')
    expect(auth.isAuthenticated.value).toBe(false)
    expect(localStorage.getItem('token')).toBeNull()
    expect(sessionStorage.getItem('token')).toBeNull()
  })

  it('初始化时从 localStorage 恢复 token', () => {
    localStorage.setItem('token', 'saved-token')
    const auth = useAuth()
    auth.init()
    expect(auth.token.value).toBe('saved-token')
    expect(auth.isAuthenticated.value).toBe(true)
  })
})
