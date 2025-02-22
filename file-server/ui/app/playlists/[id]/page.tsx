import { Video } from "@/app/types"
import Videos from "@/components/Videos"
import { Anchor, Stack } from "@mantine/core"

export default async function PlaylistPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>
  searchParams: Promise<{ "from-id": string }>
}) {
  const playlistId = (await params).id
  const fromId = (await searchParams)["from-id"]
  const data = await fetch(
    `${process.env.API_URL}/api/videos?from-id=${fromId}`,
    {
      next: { revalidate: 0 },
    }
  )
  const res: {
    videos: Video[]
    nextId: string
  } = await data.json()

  return (
    <Stack gap="lg">
      <Videos videos={res.videos} />
      {res.nextId && (
        <Anchor href={`/playlists/${playlistId}?from-id=${res.nextId}`}>
          次のページ
        </Anchor>
      )}
    </Stack>
  )
}
