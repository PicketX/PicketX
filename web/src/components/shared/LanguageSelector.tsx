// SPDX-License-Identifier: Apache-2.0
import { Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PreferenceMenu } from '@/components/shared/PreferenceMenu'

export function LanguageSelector() {
  const { t, i18n } = useTranslation()
  return <PreferenceMenu label={t('language')} icon={<Languages aria-hidden="true" />} value={i18n.resolvedLanguage ?? 'en'} options={[
    { value: 'en', label: 'English', lang: 'en' },
    { value: 'zh-CN', label: '简体中文', lang: 'zh-CN' },
  ]} onValueChange={value => {
    if (value === 'en' || value === 'zh-CN') void i18n.changeLanguage(value)
  }} />
}
