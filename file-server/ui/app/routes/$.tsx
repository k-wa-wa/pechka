import type { ActionFunction, LoaderFunction } from "@remix-run/node"

const apiURL = new URL(process.env.API_URL || "")

export const loader: LoaderFunction = (args) => {
  const url = new URL(args.request.url)
  url.protocol = apiURL.protocol
  url.host = apiURL.host

  return fetch(
    url.toString(),
    new Request(args.request, { redirect: "manual" })
  )
}

export const action: ActionFunction = (args) => {
  const url = new URL(args.request.url)
  url.protocol = apiURL.protocol
  url.host = apiURL.host

  return fetch(
    url.toString(),
    new Request(args.request, { redirect: "manual" })
  )
}
