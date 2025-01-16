import { Geist, Geist_Mono } from "next/font/google"

import "@mantine/core/styles.css"

import React, { ReactNode } from "react"
import {
  ColorSchemeScript,
  Container,
  mantineHtmlProps,
  MantineProvider,
  Space,
} from "@mantine/core"
import Header from "@/components/Header"

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
})

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
})

export const metadata = {
  title: "Pechka File Server",
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ja" {...mantineHtmlProps}>
      <head>
        <ColorSchemeScript />
        <meta
          name="viewport"
          content="minimum-scale=1, initial-scale=1, width=device-width, user-scalable=no"
        />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <MantineProvider>
          <Header />
          <Space h="md" />
          <Container size="xl">{children}</Container>
        </MantineProvider>
      </body>
    </html>
  )
}
