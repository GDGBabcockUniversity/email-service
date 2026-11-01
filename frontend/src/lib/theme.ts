import { useSyncExternalStore } from 'react'

type Theme = 'light' | 'dark'
const listeners = new Set<() => void>()
const current = (): Theme => (document.documentElement.classList.contains('dark') ? 'dark' : 'light')

function setTheme(theme: Theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark')
  try { localStorage.setItem('gdg-theme', theme) } catch { /* private mode: the choice just won't persist */ }
  listeners.forEach((l) => l())
}

export function useTheme() {
  const theme = useSyncExternalStore((cb) => { listeners.add(cb); return () => listeners.delete(cb) }, current)
  return [theme, () => setTheme(theme === 'dark' ? 'light' : 'dark')] as const
}
