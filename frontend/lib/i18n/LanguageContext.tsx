'use client'

import React, { createContext, useContext, useState, useEffect } from 'react'

export type Language = 'ja' | 'en'

export type Translations = Record<string, string>

export const dictionaries: Record<Language, Translations> = {
  ja: {
    'nav.home': 'ホーム',
    'nav.admin': '管理画面',
    'header.search': '検索',
    'header.searchHint': '検索 (⌘K)',
    'header.settings': '設定',
    'settings.title': '設定',
    'settings.language': '言語',
    'settings.close': '閉じる',
    'settings.langJa': '日本語',
    'settings.langEn': 'English',
  },
  en: {
    'nav.home': 'Home',
    'nav.admin': 'Admin',
    'header.search': 'Search',
    'header.searchHint': 'Search (⌘K)',
    'header.settings': 'Settings',
    'settings.title': 'Settings',
    'settings.language': 'Language',
    'settings.close': 'Close',
    'settings.langJa': 'Japanese (日本語)',
    'settings.langEn': 'English',
  },
}

interface LanguageContextType {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: string) => string
}

const LanguageContext = createContext<LanguageContextType>({
  language: 'ja',
  setLanguage: () => {},
  t: (key: string) => key,
})

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguageState] = useState<Language>('ja')

  useEffect(() => {
    const saved = localStorage.getItem('pechka_lang') as Language | null
    if (saved && (saved === 'ja' || saved === 'en')) {
      setLanguageState(saved)
    } else {
      const browserLang = navigator.language.startsWith('ja') ? 'ja' : 'en'
      setLanguageState(browserLang)
    }
  }, [])

  const setLanguage = (lang: Language) => {
    setLanguageState(lang)
    localStorage.setItem('pechka_lang', lang)
  }

  const t = (key: string): string => {
    return dictionaries[language]?.[key] ?? dictionaries['ja']?.[key] ?? key
  }

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  return useContext(LanguageContext)
}
