import { Video, VideoTimestamp } from "@/app/types"
import { Stack, Breadcrumbs, Anchor } from "@mantine/core"
import VideoView from "@/components/VideoView"

async function fetchVideoData(videoId: string): Promise<Video> {
  const data = await fetch(`${process.env.API_URL}/api/videos/${videoId}`, {
    next: { revalidate: 0 },
  })
  return await data.json()
}

async function fetchVideoTimestamps(videoId: string): Promise<VideoTimestamp[]> {
  const data = await fetch(
    `${process.env.API_URL}/api/video-timestamps/${videoId}`,
    {
      next: { revalidate: 0 },
    }
  )
  return await data.json()
}

export default async function VideoPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const videoId = (await params).id
  const [video, timestamps] = await Promise.all([
    fetchVideoData(videoId),
    fetchVideoTimestamps(videoId),
  ])

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

      <VideoView video={video} timestamps={timestamps} />
    </Stack>
  )
}
