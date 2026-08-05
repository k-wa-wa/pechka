import type { Metadata, Viewport } from 'next'
import './globals.css'
import Header from '@/components/Header'
import { LanguageProvider } from '@/lib/i18n/LanguageContext'

export const metadata: Metadata = {
  title: 'pechka',
  description: 'Media streaming platform',
  appleWebApp: {
    title: 'pechka',
    capable: true,
    statusBarStyle: 'black-translucent',
  },
}

export const viewport: Viewport = {
  themeColor: '#0d1117',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="ja">
      <body>
        <LanguageProvider>
          <Header />
          <main style={{ minHeight: 'calc(100vh - 60px)' }}>{children}</main>
        </LanguageProvider>
      </body>
    </html>
  )
}

