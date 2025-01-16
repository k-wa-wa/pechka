import { Video } from "@/app/types"
import { Stack, Title, Text } from "@mantine/core"
import Player from "next-video/player"

export default async function VideoPage({ params }: { params: Promise<{ id: string }> }) {
  const videoId = (await params).id
  const data = await fetch(`${process.env.API_URL}/api/videos/${videoId}`, {
    next: { revalidate: 0 },
  })
  const video: Video = await data.json()

  return (
    <Stack>
      <Player src={video.url} />

      <Title>{video.title}</Title>
      <Text>{video.description}</Text>
    </Stack>
  )
}
