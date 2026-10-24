// SPDX-License-Identifier: Apache-2.0
export type Theme = 'light' | 'dark' | 'system'
const preferenceKey = 'picketx.theme'

export function readTheme(): Theme {
  try {
    const saved = localStorage.getItem(preferenceKey)
    if (saved === 'light' || saved === 'dark') return saved
  } catch { /* Theme switching still works when browser storage is unavailable. */ }
  return 'system'
}

export function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark' || (theme === 'system' && matchMedia('(prefers-color-scheme: dark)').matches))
}

export function saveTheme(theme: Theme) {
  try { localStorage.setItem(preferenceKey, theme) } catch { /* Preference remains in memory. */ }
}
