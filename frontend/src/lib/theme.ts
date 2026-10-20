import { useEffect, useState } from 'react'

export function preferredDark() {
  try {
    const saved = localStorage.getItem('theme')
    return saved
      ? saved === 'dark'
      : matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return false
  }
}

export function applyTheme(dark: boolean) {
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
  try {
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  } catch {
    /* Persistence is optional. */
  }
}

export type Theme = { dark: boolean; toggle: () => void }

export function useTheme(): Theme {
  const [dark, setDark] = useState(preferredDark)
  useEffect(() => applyTheme(dark), [dark])
  return { dark, toggle: () => setDark((value) => !value) }
}
