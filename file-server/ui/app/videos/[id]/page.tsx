import { Video } from "@/app/types"
import { Stack, Breadcrumbs, Anchor } from "@mantine/core"
import VideoView from "@/components/VideoView"

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
            href: "/videos",
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

      <VideoView video={video} />
    </Stack>
  )
}
