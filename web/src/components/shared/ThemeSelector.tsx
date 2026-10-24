// SPDX-License-Identifier: Apache-2.0
import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { applyTheme, readTheme, saveTheme } from '@/lib/theme'
import type { Theme } from '@/lib/theme'

export function ThemeSelector() {
  const { t } = useTranslation()
  const id = useId()
  const [theme, setTheme] = useState<Theme>(readTheme)
  useEffect(() => {
    const system = matchMedia('(prefers-color-scheme: dark)')
    const update = () => applyTheme(theme)
    update()
    system.addEventListener('change', update)
    return () => system.removeEventListener('change', update)
  }, [theme])

  return <>
    <label className="sr-only" htmlFor={id}>{t('theme.label')}</label>
    <select className="theme-select" id={id} value={theme} onChange={e => {
      const selected = e.target.value as Theme
      saveTheme(selected); setTheme(selected)
    }}>
      <option value="system">{t('theme.system')}</option>
      <option value="light">{t('theme.light')}</option>
      <option value="dark">{t('theme.dark')}</option>
    </select>
  </>
}
