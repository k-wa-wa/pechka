import {
  Links,
  LiveReload,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "@remix-run/react"
import { Container, MantineProvider, Space } from "@mantine/core"
import "@mantine/core/styles.css"
import Header from "@/src/components/Header"
import { ReactNode } from "react"

export function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body>
        <MantineProvider>
          <Header />
          <Space h="md" />
          <Container size="xl" p={{ base: "sm", sm: "md" }}>
            {children}
          </Container>
        </MantineProvider>
        <Scripts />
        <ScrollRestoration />
        <LiveReload />
      </body>
    </html>
  )
}

export default function App() {
  return <Outlet />
}
