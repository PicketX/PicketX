import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from '@/locales/en.json'
import zhCN from '@/locales/zh-CN.json'

void i18n.use(initReactI18next).init({
  resources: { en: { translation: en }, 'zh-CN': { translation: zhCN } },
  lng: 'en',
  fallbackLng: 'en',
  supportedLngs: ['en', 'zh-CN'],
  interpolation: { escapeValue: false },
})

i18n.on('languageChanged', (language) => { document.documentElement.lang = language })
export default i18n
