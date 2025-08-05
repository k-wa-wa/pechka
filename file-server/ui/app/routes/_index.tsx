import { Anchor, Breadcrumbs, Flex, Stack, Title } from "@mantine/core"
import { Playlist, Video } from "@/src/types"
import Videos from "@/src/components/Videos"
import { Link, useLoaderData } from "@remix-run/react"
import axios from "axios"

export const loader = async () => {
  const playlists: Playlist[] = await axios
    .get(`${process.env.API_URL}/api/playlists`)
    .then((res) => res.data)
    .catch(e => console.log(e))
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
