import type { Metadata } from "next";
import { Geist } from "next/font/google";
import Link from "next/link";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "pechka",
  description: "ホームメディア基盤",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ja" className={`${geistSans.variable} h-full antialiased`}>
      <body className="min-h-full flex flex-col bg-gray-50">
        <header className="bg-white border-b border-gray-200 sticky top-0 z-10">
          <div className="max-w-7xl mx-auto px-4 h-14 flex items-center gap-6">
            <Link href="/" className="font-bold text-lg text-gray-900">pechka</Link>
            <nav className="flex gap-4 text-sm">
              <Link href="/" className="text-gray-600 hover:text-gray-900">コンテンツ</Link>
              <Link href="/search" className="text-gray-600 hover:text-gray-900">検索</Link>
              <Link href="/admin" className="text-gray-600 hover:text-gray-900">管理</Link>
            </nav>
          </div>
        </header>
        <main className="flex-1 max-w-7xl mx-auto px-4 py-6 w-full">
          {children}
        </main>
      </body>
    </html>
  );
}
