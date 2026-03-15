export type ThemePreference = 'light' | 'dark'
export const THEME_STORAGE_KEY = 'e5-theme'

export function getStoredThemePreference(): ThemePreference | null {
  const value = localStorage.getItem(THEME_STORAGE_KEY)
  return value === 'light' || value === 'dark' ? value : null
}

export function resolveInitialThemePreference(): ThemePreference {
  return getStoredThemePreference() ?? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
}

export function applyThemePreference(theme: ThemePreference) {
  document.documentElement.classList.toggle('dark', theme === 'dark')
}

export function saveThemePreference(theme: ThemePreference) {
  localStorage.setItem(THEME_STORAGE_KEY, theme)
}
