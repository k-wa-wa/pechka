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
      <body className="min-h-full flex flex-col" style={{ background: "var(--gh-bg)", color: "var(--gh-text)" }}>
        <header style={{ background: "var(--gh-surface)", borderBottom: "1px solid var(--gh-border)" }} className="sticky top-0 z-10">
          <div className="max-w-7xl mx-auto px-4 h-14 flex items-center gap-6">
            <Link href="/" className="font-bold text-xl tracking-tight" style={{ color: "var(--nf-red)" }}>pechka</Link>
            <nav className="flex gap-5 text-sm font-medium">
              <Link href="/" style={{ color: "var(--gh-muted)" }} className="hover:text-white transition-colors">コンテンツ</Link>
              <Link href="/search" style={{ color: "var(--gh-muted)" }} className="hover:text-white transition-colors">検索</Link>
              <Link href="/admin" style={{ color: "var(--gh-muted)" }} className="hover:text-white transition-colors">管理</Link>
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
