// SPDX-License-Identifier: Apache-2.0
import { useEffect, useState } from 'react'
import { Monitor, Moon, Sun } from 'lucide-react'
import { PreferenceMenu } from '@/components/shared/PreferenceMenu'
import { useTranslation } from 'react-i18next'
import { applyTheme, readTheme, saveTheme } from '@/lib/theme'
import type { Theme } from '@/lib/theme'

export function ThemeSelector() {
  const { t } = useTranslation()
  const [theme, setTheme] = useState<Theme>(readTheme)
  useEffect(() => {
    const system = matchMedia('(prefers-color-scheme: dark)')
    const update = () => applyTheme(theme)
    update()
    system.addEventListener('change', update)
    return () => system.removeEventListener('change', update)
  }, [theme])

  const icons = { system: <Monitor aria-hidden="true" />, light: <Sun aria-hidden="true" />, dark: <Moon aria-hidden="true" /> }
  return <PreferenceMenu label={t('theme.label')} icon={icons[theme]} value={theme} options={[
    { value: 'light', label: t('theme.light'), icon: icons.light },
    { value: 'dark', label: t('theme.dark'), icon: icons.dark },
    { value: 'system', label: t('theme.system'), description: t('theme.systemHelp'), icon: icons.system },
  ]} onValueChange={value => {
    if (value !== 'light' && value !== 'dark' && value !== 'system') return
    saveTheme(value); setTheme(value)
  }} />
}
