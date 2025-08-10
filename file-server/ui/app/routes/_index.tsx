import { Anchor, Breadcrumbs, Flex, Stack, Title } from "@mantine/core"
import { Playlist, Video } from "@/src/types"
import Videos from "@/src/components/Videos"
import { data, Link, useLoaderData } from "@remix-run/react"
import { LoaderFunctionArgs } from "@remix-run/node"

export const loader = async ({ request }: LoaderFunctionArgs) => {
  const requestId =
    request.headers.get("x-request-id") || `ui_${crypto.randomUUID()}`
  const res = await fetch(`${process.env.API_URL}/api/playlists`, {
    headers: {
      "x-request-id": requestId,
    },
  })

  return data((await res.json()) as Playlist[], {
    headers: { "Cache-Control": "public, max-age=10, s-maxage=10" },
  })
}

export default function VideosPage() {
  const { data: playlists } = useLoaderData<typeof loader>()

  return (
    <Stack gap="lg">
      {playlists.map(({ playlist: { title, id }, videos }) => (
        <Stack key={title}>
          <Flex gap="md" align="flex-end">
            <Title order={2}>{title}</Title>
            <Link to={`/playlists/${id}`}>more</Link>
          </Flex>
          <Videos videos={videos.slice(0, 8)} />
        </Stack>
      ))}
    </Stack>
  )
}
