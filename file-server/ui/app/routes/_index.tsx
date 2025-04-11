import { Anchor, Breadcrumbs, Stack, Title } from "@mantine/core"
import { Playlist, Video } from "@/src/types"
import Videos from "@/src/components/Videos"
import { useLoaderData } from "@remix-run/react"

export const loader = async () => {
  const data = await fetch(`${process.env.API_URL}/api/playlists`)
  const playlists: Playlist[] = await data.json()
  return playlists
}

export default function VideosPage() {
  const playlists = useLoaderData<typeof loader>()

  return (
    <Stack gap="lg">
      {playlists.map(({ title, videos }) => (
        <Stack key={title}>
          <Title order={2}>{title}</Title>
          <Videos videos={videos} />
        </Stack>
      ))}
    </Stack>
  )
}
