'use client'

import React, { createContext, useContext, useState, useEffect } from 'react'

export type Language = 'ja' | 'en'

export type Translations = Record<string, string>

export const dictionaries: Record<Language, Translations> = {
  ja: {
    // Header & Nav
    'nav.home': 'ホーム',
    'nav.admin': '管理画面',
    'header.search': '検索',
    'header.searchHint': '検索 (⌘K)',
    'header.settings': '設定',

    // Settings Modal
    'settings.title': '設定',
    'settings.language': '言語',
    'settings.close': '閉じる',
    'settings.langJa': '日本語',
    'settings.langEn': 'English',

    // Home & Filter
    'home.title': 'コンテンツ一覧',
    'home.noContents': 'コンテンツがありません',
    'filter.all': 'すべて',

    // Search Modal
    'search.placeholder': 'コンテンツを検索...',
    'search.searching': '検索中...',
    'search.noResultsPrefix': '「',
    'search.noResultsSuffix': '」に一致するコンテンツが見つかりません',

    // Content Detail & Player
    'content.published': '公開',
    'content.minutes': '分',
    'content.seconds': '秒',
    'player.loading': '読み込み中...',
    'player.loadingVr': 'VRビューアを読み込み中...',
    'player.notFound': '動画が見つかりません',
    'player.quality': '画質:',

    // Admin & Table
    'admin.title': '管理画面',
    'admin.itemsCount': '件',
    'admin.table.colTitle': 'タイトル',
    'admin.table.colType': '種別',
    'admin.table.colStatus': 'ステータス',
    'admin.table.colTags': 'タグ',
    'admin.table.colUpdatedAt': '更新日時',
    'admin.table.btnEdit': '編集',
    'admin.table.btnSubtitles': '字幕',
    'admin.table.btnArchive': 'アーカイブ',
    'admin.table.btnUnarchive': 'アーカイブ解除',
    'admin.table.badgeArchived': 'アーカイブ済み',
    'admin.table.confirmArchive': 'このコンテンツをアーカイブしますか？公開画面から見えなくなります。',
    'admin.table.confirmUnarchive': 'このコンテンツのアーカイブを解除しますか？',
    'admin.table.archiveError': '処理に失敗しました',
    'admin.table.noContents': 'コンテンツがありません',

    // Edit Modal
    'editModal.title': 'コンテンツを編集',
    'editModal.fieldTitle': 'タイトル',
    'editModal.fieldDescription': '説明',
    'editModal.fieldTags': 'タグ (カンマ区切り)',
    'editModal.fieldStatus': 'ステータス',
    'editModal.btnSave': '保存',
    'editModal.btnCancel': 'キャンセル',
    'editModal.saving': '保存中...',
  },
  en: {
    // Header & Nav
    'nav.home': 'Home',
    'nav.admin': 'Admin',
    'header.search': 'Search',
    'header.searchHint': 'Search (⌘K)',
    'header.settings': 'Settings',

    // Settings Modal
    'settings.title': 'Settings',
    'settings.language': 'Language',
    'settings.close': 'Close',
    'settings.langJa': 'Japanese (日本語)',
    'settings.langEn': 'English',

    // Home & Filter
    'home.title': 'Contents',
    'home.noContents': 'No contents found',
    'filter.all': 'All',

    // Search Modal
    'search.placeholder': 'Search contents...',
    'search.searching': 'Searching...',
    'search.noResultsPrefix': 'No contents matching "',
    'search.noResultsSuffix': '"',

    // Content Detail & Player
    'content.published': 'Published',
    'content.minutes': 'm ',
    'content.seconds': 's',
    'player.loading': 'Loading...',
    'player.loadingVr': 'Loading VR viewer...',
    'player.notFound': 'Video not found',
    'player.quality': 'Quality:',

    // Admin & Table
    'admin.title': 'Admin Panel',
    'admin.itemsCount': 'items',
    'admin.table.colTitle': 'Title',
    'admin.table.colType': 'Type',
    'admin.table.colStatus': 'Status',
    'admin.table.colTags': 'Tags',
    'admin.table.colUpdatedAt': 'Updated At',
    'admin.table.btnEdit': 'Edit',
    'admin.table.btnSubtitles': 'Subtitles',
    'admin.table.btnArchive': 'Archive',
    'admin.table.btnUnarchive': 'Unarchive',
    'admin.table.badgeArchived': 'Archived',
    'admin.table.confirmArchive': 'Archive this content? It will be hidden from the public site.',
    'admin.table.confirmUnarchive': 'Unarchive this content?',
    'admin.table.archiveError': 'Failed to process request',
    'admin.table.noContents': 'No contents found',

    // Edit Modal
    'editModal.title': 'Edit Content',
    'editModal.fieldTitle': 'Title',
    'editModal.fieldDescription': 'Description',
    'editModal.fieldTags': 'Tags (comma separated)',
    'editModal.fieldStatus': 'Status',
    'editModal.btnSave': 'Save',
    'editModal.btnCancel': 'Cancel',
    'editModal.saving': 'Saving...',
  },
}

interface LanguageContextType {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: string) => string
}

const LanguageContext = createContext<LanguageContextType>({
  language: 'en',
  setLanguage: () => {},
  t: (key: string) => key,
})

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguageState] = useState<Language>('en')

  useEffect(() => {
    const saved = localStorage.getItem('pechka_lang') as Language | null
    if (saved && (saved === 'ja' || saved === 'en')) {
      setLanguageState(saved)
    } else {
      setLanguageState('en')
    }
  }, [])

  const setLanguage = (lang: Language) => {
    setLanguageState(lang)
    localStorage.setItem('pechka_lang', lang)
  }

  const t = (key: string): string => {
    return dictionaries[language]?.[key] ?? dictionaries['en']?.[key] ?? key
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
