import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zh from './locales/zh.json'
import en from './locales/en.json'

let language = navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
try {
  const saved = localStorage.getItem('language')
  if (saved === 'zh' || saved === 'en') {
    language = saved
  }
} catch {
  /* Storage can be unavailable in private browser contexts. */
}

void i18n.use(initReactI18next).init({
  lng: language,
  fallbackLng: 'en',
  supportedLngs: ['en', 'zh'],
  keySeparator: false,
  nsSeparator: false,
  resources: {
    en: { translation: en },
    zh: { translation: zh },
  },
  interpolation: { escapeValue: false },
})
i18n.on('languageChanged', (language) => {
  document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en'
  try {
    localStorage.setItem('language', language)
  } catch {
    /* Preference persistence is optional. */
  }
})
document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en'
export default i18n
