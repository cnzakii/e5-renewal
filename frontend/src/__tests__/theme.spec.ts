import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getStoredThemePreference,
  resolveInitialThemePreference,
  applyThemePreference,
  saveThemePreference,
  THEME_STORAGE_KEY,
} from '../utils/theme'

function stubMatchMedia(prefersDark: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' ? prefersDark : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

describe('theme utilities', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  it('saved theme wins over system preference', () => {
    stubMatchMedia(true) // system prefers dark
    localStorage.setItem(THEME_STORAGE_KEY, 'light')

    expect(resolveInitialThemePreference()).toBe('light')
  })

  it('system preference is used when no saved theme exists', () => {
    stubMatchMedia(true)
    expect(resolveInitialThemePreference()).toBe('dark')

    stubMatchMedia(false)
    expect(resolveInitialThemePreference()).toBe('light')
  })

  it('getStoredThemePreference returns null for invalid values', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'invalid')
    expect(getStoredThemePreference()).toBeNull()
  })

  it('applyThemePreference("dark") adds dark class', () => {
    applyThemePreference('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('applyThemePreference("light") removes dark class', () => {
    document.documentElement.classList.add('dark')
    applyThemePreference('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('saveThemePreference("dark") writes the shared storage key', () => {
    saveThemePreference('dark')
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')
  })
})
