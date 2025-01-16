import { Video } from "@/app/types"
import { Stack, Title, Text, Breadcrumbs, Anchor } from "@mantine/core"
import Player from "next-video/player"

export default async function VideoPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const videoId = (await params).id
  const data = await fetch(`${process.env.API_URL}/api/videos/${videoId}`, {
    next: { revalidate: 0 },
  })
  const video: Video = await data.json()

  return (
    <Stack>
      <Breadcrumbs>
        {[
          {
            title: "Top",
            href: "/",
          },
          {
            title: "Videos",
            href: "/videos"
          },
          {
            title: video.title,
            href: `/videos/${videoId}`,
          },
        ].map(({ title, href }) => (
          <Anchor key={title} href={href}>
            {title}
          </Anchor>
        ))}
      </Breadcrumbs>

      <Player src={video.url} />

      <Title order={2}>{video.title}</Title>
      <Text>{video.description}</Text>
    </Stack>
  )
}
