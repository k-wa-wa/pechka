import { Anchor, Breadcrumbs, Flex, Stack, Title } from "@mantine/core"
import { Playlist, Video } from "@/src/types"
import Videos from "@/src/components/Videos"
import { Link, useLoaderData } from "@remix-run/react"

export const loader = async () => {
  const data = await fetch(`${process.env.API_URL}/api/playlists`)
  const playlists: Playlist[] = await data.json()
  return playlists
}

export default function VideosPage() {
  const playlists = useLoaderData<typeof loader>()

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
