import "@mantine/core/styles.css"

import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "@remix-run/react"
import { ColorSchemeScript, MantineProvider } from "@mantine/core"

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja" style={{ height: "100%" }}>
      <head>
        <meta charSet="utf-8" />
        <meta
          name="viewport"
          content="width=device-width, initial-scale=1, maximum-scale=1.0, user-scalable=no"
        />
        <Meta />
        <Links />
        <ColorSchemeScript />
        <link rel="manifest" href="manifest.json" />
      </head>
      <body style={{ height: "100%" }}>
        <MantineProvider>{children}</MantineProvider>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  )
}

export default function App() {
  return <Outlet />
}
